package sshclip

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func readError(r io.Reader) error {
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
		return nil, readError(rw)
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
		return readError(rw)
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
		return nil, readError(rw)
	}

	return nil, fmt.Errorf("Unexpected byte: %02x", op)
}

// SyncRegister makes an OpSync request and synchronizes the registers.
func SyncRegister(rw io.ReadWriter, reg Register) error {
	regs, err := ListRegisters(rw)
	if err != nil {
		return err
	}

	for _, r := range regs {
		var sync bool
		item, err := reg.Get(r.Register)
		switch err {
		case ErrNotExist:
			sync = true
		case nil:
			sync = !item.EqualsHash(r)
		default:
			return err
		}

		if sync {
			item, err := GetRegister(rw, r.Register)
			if err != nil {
				return err
			}

			data := make([]byte, item.Size())
			if _, err := item.Read(data); err != nil {
				return err
			}

			if err := reg.Put(uint8(item.Index()), item.Attributes(), data); err != nil {
				return err
			}

			Dlog("Synced register '%c'", item.Index())
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
			var sb [3]byte
			if _, err := channel.Read(sb[:]); err != nil {
				Elog("error reading error:", err)
				return nil
			}

			size := SizeFromBytes(sb)
			if size > MaxPayloadSize {
				Elog("error message is too large")
				return nil
			}

			errBytes := make([]byte, size)
			if _, err := channel.Read(errBytes); err != nil {
				Elog("couldn't send error message:", err)
				return nil
			}

			Elog("Error from remote:", string(errBytes))
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