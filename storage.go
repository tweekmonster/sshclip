package sshclip

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

var ErrTooLarge = errors.New("Storage data too large")
var ErrNotExist = errors.New("item does not exist")

type RegisterItem interface {
	io.Reader
	Attributes() int
	Size() int
	Register() int
}

type Register interface {
	Get(index int) (RegisterItem, error)
	Put(index int, data []byte, attrs int) error
}

type MemoryRegisterItem struct {
	Index int
	Attrs int
	Data  []byte
}

func (m *MemoryRegisterItem) Read(b []byte) (int, error) {
	return bytes.NewReader(m.Data).Read(b)
}

func (m *MemoryRegisterItem) Attributes() int {
	return m.Attrs
}

func (m *MemoryRegisterItem) Size() int {
	return len(m.Data)
}

func (m *MemoryRegisterItem) Register() int {
	return m.Index
}

type MemoryRegister struct {
	sync.RWMutex
	items map[int]*MemoryRegisterItem
}

// IsValidIndex returns true if an index is valid.  Indexes are based on Vim's
// registers.  The permitted registers are [a-z*+].  Registers [A-Z] means that
// data is appended.
func IsValidIndex(index int) bool {
	return (index > 64 && index < 91) || (index > 96 && index < 123) || (index > 41 && index < 44)
}

func NewMemoryRegister() *MemoryRegister {
	return &MemoryRegister{
		items: map[int]*MemoryRegisterItem{},
	}
}

func (m *MemoryRegister) Get(index int) (RegisterItem, error) {
	m.RLock()
	defer m.RUnlock()

	if index > 64 && index < 91 {
		index += 32
	}

	if item, ok := m.items[index]; ok {
		return item, nil
	}

	return nil, ErrNotExist
}

func (m *MemoryRegister) Put(index int, data []byte, attrs int) error {
	m.Lock()
	defer m.Unlock()

	if index > 64 && index < 91 {
		index += 32
		// Try to append.  Fallthrough to storing if index doesn't exist.
		if item, ok := m.items[index]; ok {
			if len(item.Data)+len(data) > MaxPayloadSize {
				return ErrTooLarge
			}
			item.Data = append(item.Data, data...)
			return nil
		}
	}

	m.items[index] = &MemoryRegisterItem{
		Attrs: attrs,
		Data:  data,
		Index: index,
	}

	return nil
}
