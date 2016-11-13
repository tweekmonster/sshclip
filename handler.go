package sshclip

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

var ErrClientKey = errors.New("can't manage own key")

func ReadError(r io.Reader) error {
	var sb [3]byte
	if _, err := r.Read(sb[:]); err != nil {
		return fmt.Errorf("couldn't read error: %s", err)
	}
	errBytes := make([]byte, SizeFromBytes(sb))
	if _, err := r.Read(errBytes); err != nil {
		return fmt.Errorf("couldn't read error: %s", err)
	}

	return errors.New(string(errBytes))
}

// GetRegister makes an OpGet request.
func GetRegister(rw io.ReadWriter, reg uint8) (RegisterItem, error) {
	out := OpHeader(OpGet)
	_, err := rw.Write(append(out, reg))
	if err != nil {
		return nil, err
	}

	op, err := ReadOp(rw)
	if err != nil {
		return nil, err
	}

	switch op {
	case OpSuccess:
		var attrs uint8
		if err := binary.Read(rw, binary.BigEndian, &attrs); err != nil {
			return nil, err
		}

		data, err := ReadPayloadData(rw)
		if err != nil {
			return nil, err
		}

		return NewMemoryRegisterItem(reg, attrs, data), nil
	case OpErr:
		return nil, ReadError(rw)
	}

	return nil, fmt.Errorf("Unexpected byte: %02x", op)
}

// PutRegister makes an OpPut request.
func PutRegister(rw io.ReadWriter, reg uint8, attrs uint8, data []byte) error {
	out := OpHeader(OpPut)
	out = append(out, reg)
	out = append(out, attrs)
	out = append(out, SizeToBytes(len(data))...)
	out = append(out, data...)

	if _, err := rw.Write(out); err != nil {
		return err
	}

	op, err := ReadOp(rw)
	if err != nil {
		return err
	}

	switch op {
	case OpSuccess:
		return nil
	case OpErr:
		return ReadError(rw)
	}

	return fmt.Errorf("Unexpected byte: %02x", op)
}

// ListRegisters lists the registers.
func ListRegisters(rw io.ReadWriter) ([]RegisterItemHash, error) {
	if _, err := rw.Write(OpHeader(OpList)); err != nil {
		return nil, err
	}

	op, err := ReadOp(rw)
	if err != nil {
		return nil, err
	}

	switch op {
	case OpSuccess:
		var length uint8
		if err := binary.Read(rw, binary.BigEndian, &length); err != nil {
			return nil, err
		}

		regs := make([]RegisterItemHash, length)
		if err := binary.Read(rw, binary.BigEndian, &regs); err != nil {
			return nil, err
		}

		return regs, nil

	case OpErr:
		return nil, ReadError(rw)
	}

	return nil, fmt.Errorf("Unexpected byte: %02x", op)
}

// SyncRegister makes an OpSync request and synchronizes the registers.
func SyncRegister(rw io.ReadWriter, reg Register) error {
	Dlog("Making sync request")
	remoteReg, err := ListRegisters(rw)
	if err != nil {
		return err
	}

	remoteRegMap := make(map[uint8]RegisterItemHash)

	for _, r := range remoteReg {
		remoteRegMap[r.Register] = r
	}

	for _, i := range Registers {
		remote, hasRemote := remoteRegMap[i]
		local, _ := reg.Get(i)

		if !hasRemote && local == nil {
			continue
		}

		Dlog("Reg: %c, Remote: %t, Local: %t", i, hasRemote, local != nil)

		switch {
		case !hasRemote && local != nil:
			PutRegister(rw, i, local.Attributes(), local.Bytes())
		case hasRemote && local == nil:
			if item, err := GetRegister(rw, i); err == nil {
				reg.Put(i, item.Attributes(), item.Bytes())
			}
		case hasRemote && local != nil:
			if !local.EqualsHash(remote) {
				if local.Time().UnixNano() > remote.Time {
					PutRegister(rw, i, local.Attributes(), local.Bytes())
				} else {
					if item, err := GetRegister(rw, i); err == nil {
						reg.Put(i, item.Attributes(), item.Bytes())
					}
				}
			}
		}
	}

	return nil
}

// HandlePayload is the main handler for reading channel/stream data.  Any
// thing it writes out may be read by the same function on the other end if
// storage operates over the network.
func HandlePayload(storage Register, channel io.ReadWriteCloser) error {
	// Wrapped in an inner function to make writing an error simpler.
	err := func() error {
		op, err := ReadOp(channel)
		if err != nil {
			return err
		}

		switch op {
		case OpGet:
			var reg uint8
			if err := binary.Read(channel, binary.BigEndian, &reg); err != nil {
				return err
			}

			item, err := storage.Get(reg)
			if err != nil {
				return err
			}

			size := item.Size()
			if size > MaxPayloadSize {
				return ErrPayloadTooLarge
			}

			out := OpHeader(OpSuccess)
			out = append(out, byte(item.Attributes()))
			out = append(out, SizeToBytes(item.Size())...)
			if _, err := channel.Write(out); err != nil {
				return err
			}

			if _, err := io.CopyN(channel, item, int64(size)); err != nil {
				return err
			}

			return nil

		case OpPut:
			var b [2]byte
			if _, err := channel.Read(b[:]); err != nil {
				return err
			}

			reg := b[0]
			attrs := b[1]

			data, err := ReadPayloadData(channel)
			if err != nil {
				return err
			}

			if err := storage.Put(reg, attrs, data); err != nil {
				return err
			}

			channel.Write(OpHeader(OpSuccess))
			return nil

		case OpList:
			items, err := storage.List()
			if err != nil {
				return err
			}
			header := OpHeader(OpSuccess)
			header = append(header, byte(len(items)))
			if _, err := channel.Write(header); err != nil {
				return err
			}
			if err := binary.Write(channel, binary.BigEndian, items); err != nil {
				return err
			}
			return nil

		case OpErr:
			// This should not return any errors because it's the part that reports
			// errors!
			err := ReadError(channel)
			Elog("Error from remote:", err)
			return nil

		case OpStop:
			channel.Write(OpHeader(OpSuccess))
			ListenLoopStop()
			return nil
		}

		return fmt.Errorf("Unknown op: %02x", op)
	}()

	if err != nil && err != io.EOF {
		header := OpHeader(OpErr)
		errStr := err.Error()
		header = append(header, SizeToBytes(len(errStr))...)
		header = append(header, []byte(errStr)...)
		channel.Write(header)
	}

	return err
}

func HandleKeyPayload(clientKey ssh.PublicKey, channel io.ReadWriter) error {
	err := func() error {
		op, err := ReadOp(channel)
		if err != nil {
			return err
		}

		switch op {
		case OpList:
			var reviewKeys []KeyReviewItem

			for _, k := range allKeys() {
				reviewKeys = append(reviewKeys, KeyReviewItem{
					FingerPrint: FingerPrintBytes(k.PublicKey),
					KeyRecord:   k.KeyRecord,
				})
			}

			channel.Write(OpHeader(OpSuccess))
			enc := gob.NewEncoder(channel)
			enc.Encode(reviewKeys)
			return nil

		case OpAccept:
			fingerprint := make([]byte, 32)
			channel.Read(fingerprint)

			if bytes.Equal(FingerPrintBytes(clientKey), fingerprint) {
				return ErrClientKey
			}

			key, err := FindFingerPrint(fingerprint)
			if err != nil {
				return err
			}

			key.State = "authorized"
			AddKey("authorized", key)

			if KeyExists("rejected", key) {
				RemoveKey("rejected", key)
			}

			channel.Write(OpHeader(OpSuccess))
			return nil

		case OpReject:
			fingerprint := make([]byte, 32)
			channel.Read(fingerprint)

			if bytes.Equal(FingerPrintBytes(clientKey), fingerprint) {
				return ErrClientKey
			}

			key, err := FindFingerPrint(fingerprint)
			if err != nil {
				return err
			}

			key.State = "rejected"
			AddKey("rejected", key)

			if KeyExists("authorized", key) {
				RemoveKey("authorized", key)
			}

			channel.Write(OpHeader(OpSuccess))
			return nil
		}

		return fmt.Errorf("Unknown op: %02x", op)
	}()

	if err != nil && err != io.EOF {
		header := OpHeader(OpErr)
		errStr := err.Error()
		header = append(header, SizeToBytes(len(errStr))...)
		header = append(header, []byte(errStr)...)
		channel.Write(header)
	}

	return err
}
