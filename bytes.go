package sshclip

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// MaxPayloadSize is the maximum size of the data that can be sent/received.
// It's just under 8 MiB.
const MaxPayloadSize = (1 << 23) - 1
const ProtocolVersion = 1

const (
	OpErr     uint8 = iota // An error is being reported
	OpSuccess              // Success.
	OpList                 // Request to list registers.
	OpSync                 // Request to sync registers.
	OpGet                  // Request to get register data.
	OpPut                  // Request to put data into a register.
	OpStop                 // Request to stop the server.
	OpAccept               // Request to accept something.
	OpReject               // Request to reject something.
)

const (
	VimCharAttr int = 1 << iota
	VimLineAttr
	VimBlockAttr
	URLAttr
)

var ErrOldProto = errors.New("protocol out of date")
var ErrPayloadTooLarge = errors.New("payload is too large")

func OpHeader(op uint8) []byte {
	return []byte{ProtocolVersion, op}
}

func ReadOp(r io.Reader) (uint8, error) {
	var i uint8
	if err := binary.Read(r, binary.BigEndian, &i); err != nil {
		return 0, err
	}

	if i < ProtocolVersion {
		return 0, ErrOldProto
	}

	if err := binary.Read(r, binary.BigEndian, &i); err != nil {
		return 0, err
	}

	return i, nil
}

func SizeFromBytes(b [3]byte) int {
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}

func SizeToBytes(size int) (b []byte) {
	b = make([]byte, 3)
	b[0] = byte(size >> 16)
	b[1] = byte(size >> 8 & 0xff)
	b[2] = byte(size & 0xff)
	return
}

func ReadPayloadData(r io.Reader) ([]byte, error) {
	var sb [3]byte
	if _, err := r.Read(sb[:]); err != nil {
		return nil, err
	}

	size := SizeFromBytes(sb)
	if size > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	data := make([]byte, size)
	n, err := r.Read(data)
	if err != nil {
		return nil, err
	}

	if n != size {
		return nil, fmt.Errorf("data size mismatch, expected %d but got %d", size, n)
	}

	return data, nil
}
