package sshclip

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
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
	Hash() RegisterItemHash
	EqualsHash(h RegisterItemHash) bool
}

// Register is a storage for Register data.
type Register interface {
	Get(reg uint8) (RegisterItem, error)
	Put(reg, attrs uint8, data []byte) error
	List() ([]RegisterItemHash, error)
}

type RegisterItemHash struct {
	Register uint8
	Hash     [16]byte
}

// MemoryRegisterItem is an in-memory Register entry.
type MemoryRegisterItem struct {
	created time.Time
	hash    [16]byte
	index   uint8
	attrs   uint8
	data    []byte
}

func NewMemoryRegisterItem(reg, attrs uint8, data []byte) *MemoryRegisterItem {
	h := md5.New()
	h.Write([]byte{attrs})
	h.Write(data)
	var md5sum [16]byte
	copy(md5sum[:], h.Sum(nil))

	return &MemoryRegisterItem{
		created: time.Now(),
		hash:    md5sum,
		index:   reg,
		attrs:   attrs,
		data:    data,
	}
}

// Read register data into b.
func (m *MemoryRegisterItem) Read(b []byte) (int, error) {
	return bytes.NewReader(m.data).Read(b)
}

// Time the register item was created.
func (m *MemoryRegisterItem) Time() time.Time {
	return m.created
}

// Hash of the register item's attributes + data.
func (m *MemoryRegisterItem) Hash() RegisterItemHash {
	return RegisterItemHash{
		Register: m.index,
		Hash:     m.hash,
	}
}

func (m *MemoryRegisterItem) EqualsHash(h RegisterItemHash) bool {
	return bytes.Equal(m.hash[:], h.Hash[:])
}

// Attributes for the register item.
func (m *MemoryRegisterItem) Attributes() uint8 {
	return m.attrs
}

// Size of the register item's data.
func (m *MemoryRegisterItem) Size() int {
	return len(m.data)
}

// Index of the register item in the register.
func (m *MemoryRegisterItem) Index() int {
	return int(m.index)
}

// MemoryRegister is an in-memory register.
type MemoryRegister struct {
	sync.RWMutex
	Notify chan uint16
	items  map[uint8]*MemoryRegisterItem
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
		Notify: make(chan uint16),
		items:  map[uint8]*MemoryRegisterItem{},
	}
}

// GetItem gets an item from the register, without a notification.
func (m *MemoryRegister) GetItem(reg uint8) (*MemoryRegisterItem, error) {
	m.RLock()
	defer m.RUnlock()

	if reg > 64 && reg < 91 {
		reg += 32
	}

	if !IsValidIndex(reg) {
		return nil, ErrInvalidIndex
	}

	if item, ok := m.items[reg]; ok {
		return item, nil
	}

	return nil, ErrNotExist
}

// Get an item from the register and create a notification.
func (m *MemoryRegister) Get(reg uint8) (RegisterItem, error) {
	item, err := m.GetItem(reg)
	if err != nil {
		return nil, err
	}

	defer func() {
		m.Notify <- binary.BigEndian.Uint16([]byte{OpGet, reg})
	}()

	return item, nil
}

// PutItem puts an item into the register, without a notification.
func (m *MemoryRegister) PutItem(reg, attrs uint8, data []byte) error {
	m.Lock()
	defer m.Unlock()

	if !IsValidIndex(reg) {
		return ErrInvalidIndex
	}

	if reg > 64 && reg < 91 {
		reg += 32
		// Try to append.  Fallthrough to storing if reg doesn't exist.
		if item, ok := m.items[reg]; ok {
			if len(item.data)+len(data) > MaxPayloadSize {
				return ErrTooLarge
			}
			item.data = append(item.data, data...)
			return nil
		}
	}

	m.items[reg] = NewMemoryRegisterItem(reg, attrs, data)
	return nil
}

// Put an item into the register and create a notification.
func (m *MemoryRegister) Put(reg, attrs uint8, data []byte) error {
	if err := m.PutItem(reg, attrs, data); err != nil {
		return err
	}

	defer func() {
		m.Notify <- binary.BigEndian.Uint16([]byte{OpPut, reg})
	}()

	return nil
}

// List register item hashes.
func (m *MemoryRegister) List() ([]RegisterItemHash, error) {
	m.RLock()
	defer m.RUnlock()

	var items []RegisterItemHash
	for _, item := range m.items {
		items = append(items, item.Hash())
	}

	return items, nil
}
