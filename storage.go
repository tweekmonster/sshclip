package sshclip

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrTooLarge = errors.New("storage data too large")
var ErrNotExist = errors.New("item does not exist")
var ErrInvalidIndex = errors.New("invalid index")

// RegisterItem is an entry in the Register.
type RegisterItem interface {
	io.Reader
	Index() int
	Attributes() uint8
	Size() int
	Time() time.Time
}

// Register is a storage for Register data.
type Register interface {
	Get(reg uint8) (RegisterItem, error)
	Put(reg, attrs uint8, data []byte) error
}

// MemoryRegisterItem is an in-memory Register entry.
type MemoryRegisterItem struct {
	Updated       time.Time
	RegisterIndex uint8
	Attrs         uint8
	Data          []byte
}

// Read register data into b.
func (m *MemoryRegisterItem) Read(b []byte) (int, error) {
	return bytes.NewReader(m.Data).Read(b)
}

func (m *MemoryRegisterItem) Time() time.Time {
	return m.Updated
}

// Attributes for the register item.
func (m *MemoryRegisterItem) Attributes() uint8 {
	return m.Attrs
}

// Size of the register item's data.
func (m *MemoryRegisterItem) Size() int {
	return len(m.Data)
}

// Index of the register item in the register.
func (m *MemoryRegisterItem) Index() int {
	return int(m.RegisterIndex)
}

// MemoryRegister is an in-memory register.
type MemoryRegister struct {
	sync.RWMutex
	items map[uint8]*MemoryRegisterItem
}

// IsValidIndex returns true if a reg is valid.  Indexes are based on Vim's
// registers.  The permitted registers are [a-z*+].  Registers [A-Z] means that
// data is appended.
func IsValidIndex(reg uint8) bool {
	return (reg > 64 && reg < 91) || (reg > 96 && reg < 123) || (reg > 41 && reg < 44)
}

// NewMemoryRegister creates a new MemoryRegister.
func NewMemoryRegister() *MemoryRegister {
	return &MemoryRegister{
		items: map[uint8]*MemoryRegisterItem{},
	}
}

// Get an item from the MemoryRegister.
func (m *MemoryRegister) Get(reg uint8) (RegisterItem, error) {
	m.RLock()
	defer m.RUnlock()

	if !IsValidIndex(reg) {
		return nil, ErrInvalidIndex
	}

	if reg > 64 && reg < 91 {
		reg += 32
	}

	if item, ok := m.items[reg]; ok {
		return item, nil
	}

	return nil, ErrNotExist
}

// Put an item into the MemoryRegister.
func (m *MemoryRegister) Put(reg, attrs uint8, data []byte) error {
	m.Lock()
	defer m.Unlock()

	if !IsValidIndex(reg) {
		return ErrInvalidIndex
	}

	if reg > 64 && reg < 91 {
		reg += 32
		// Try to append.  Fallthrough to storing if reg doesn't exist.
		if item, ok := m.items[reg]; ok {
			if len(item.Data)+len(data) > MaxPayloadSize {
				return ErrTooLarge
			}
			item.Data = append(item.Data, data...)
			return nil
		}
	}

	m.items[reg] = &MemoryRegisterItem{
		Attrs:         attrs,
		Data:          data,
		RegisterIndex: reg,
	}

	return nil
}
