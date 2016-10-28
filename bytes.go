package sshclip

import (
	"errors"
	"fmt"
	"io"
)

// MaxPayloadSize is the maximum size of the data that can be sent/received.
// It's just under 8 MiB.
const MaxPayloadSize = (1 << 23) - 1
const ProtocolVersion = 1

const (
	OpInvalid int = iota - 1
	OpNoop
	OpList
	OpGet
	OpPut
)

const (
	VimCharAttr int = 1 << iota
	VimLineAttr
	VimBlockAttr
	URLAttr
)

var ErrOldProto = errors.New("protocol out of date")
var ErrPayloadTooLarge = errors.New("payload is too large")

// MakePayloadHeader returns 4 bytes that represents a payload header.
//
// The main header bits are:
//   Protocol(8), Operation(8), Attributes(8), Register(8)
//
// Protocol is 5 bits, 0-31.  It will only increment when there's a
// change to the main header layout (mainly increasing the number of
// attributes), which should be very rare.
//
// Operation is 1 bit boolean.  Either OpGet or OpPut.
//
// Attributes is 10 bit field.  These are basic flags for describing the data.
//
// Register is an 8 bit character.
func MakePayloadHeader(op int, attr int, register int) (header []byte) {
	header = make([]byte, 4)
	header[0] = byte(ProtocolVersion)
	header[1] = byte(op)
	header[2] = byte(attr)
	header[3] = byte(register)
	return
}

// ReadPayloadHeader reads the first 3 bytes and returns the Operation,
// Attributes, and Register.
func ReadPayloadHeader(r io.Reader) (op int, attrs int, reg int, err error) {
	op = OpInvalid

	var header [4]byte
	_, err = io.ReadAtLeast(r, header[:], 4)
	if err != nil {
		return
	}

	Dlog("Header: %#v", header)
	Dlog("Protocol: %d", header[0])

	if int(header[0]) < ProtocolVersion {
		// Set the error, but still return the header bytes.
		err = ErrOldProto
	}

	op = int(header[1])
	attrs = int(header[2])
	reg = int(header[3])
	return
}

// ReadPayload reads arbitrarily sized data from an io.Reader.  The header is a
// 24bit int indicating the size of the remaining data.  Only the first 23 bits
// are used making the maximum number of bytes 8388607, which is just under
// 8MiB.  This should be enough storage for your Harry Potter fan fiction.
func ReadPayload(r io.Reader) (payload []byte, err error) {
	var header [3]byte
	_, err = io.ReadAtLeast(r, header[:], 3)
	if err != nil {
		return nil, err
	}

	size := int(header[0])<<16 | int(header[1])<<8 | int(header[2])
	Dlog("Incoming size:", size)

	if size > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	payload = make([]byte, size)

	n, err := io.ReadAtLeast(r, payload, size)
	if err != nil {
		return nil, err
	}

	Dlog("Read bytes: %s", payload)

	if n != size {
		// Can't risk the remaining bytes putting gibberish in the registers.
		// The client will reconnect if it needs to.
		return nil, fmt.Errorf("Got %d bytes instead of %d", n, size)
	}

	return payload, nil
}

func SizeToBytes(size int) (sizeBytes []byte, err error) {
	if size > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	sizeBytes = make([]byte, 3)
	sizeBytes[0] = byte(size >> 16)
	sizeBytes[1] = byte(size >> 8 & 0xff)
	sizeBytes[2] = byte(size & 0xff)
	return
}

func SendRegister(w io.Writer, item RegisterItem) (int, error) {
	sizeBytes, err := SizeToBytes(item.Size())
	header := MakePayloadHeader(OpPut, item.Attributes(), item.Register())
	Dlog("Size bytes: %#v", sizeBytes)

	header = append(header, sizeBytes[:]...)

	n, err := w.Write(header)
	if err != nil {
		return n, err
	}

	n2, err := io.CopyN(w, item, int64(item.Size()))
	return int(n2), err
}
