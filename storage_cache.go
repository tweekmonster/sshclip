package sshclip

import (
	"sync"
	"time"
)

type remoteQueueItem struct {
	created time.Time
	reg     uint8
	attrs   uint8
	data    []byte
}

type CachedStorage struct {
	sync.RWMutex
	backend  Register
	items    map[uint8]RegisterItem
	putQueue chan remoteQueueItem
}

func NewCacheStorage(backend Register) *CachedStorage {
	c := &CachedStorage{
		backend:  backend,
		items:    map[uint8]RegisterItem{},
		putQueue: make(chan remoteQueueItem, 4),
	}

	go c.putBackground()

	return c
}

func (c *CachedStorage) Get(reg uint8) (RegisterItem, error) {
	c.RLock()
	defer c.RUnlock()

	if !IsValidIndex(reg) {
		return nil, ErrInvalidIndex
	}

	if item, ok := c.items[reg]; ok {
		Dlog("Cache hit for register: %c", reg)
		return item, nil
	}

	item, err := c.backend.Get(reg)
	if err != nil {
		return nil, err
	}

	c.Lock()
	defer c.Unlock()
	c.items[reg] = item

	return item, nil
}

func (c *CachedStorage) putBackground() {
	last := time.Now()

	for item := range c.putQueue {
		if item.created.After(last) {
			Dlog("Put remote reg: %c", item.reg)
			c.backend.Put(item.reg, item.attrs, item.data)
			last = item.created
		}
	}
}

func (c *CachedStorage) Put(reg, attrs uint8, data []byte) error {
	c.Lock()
	defer c.Unlock()

	if !IsValidIndex(reg) {
		return ErrInvalidIndex
	}

	item := NewMemoryRegisterItem(reg, attrs, data)
	c.items[reg] = item

	c.putQueue <- remoteQueueItem{
		created: time.Now(),
		reg:     reg,
		attrs:   attrs,
		data:    data,
	}
	return nil
}
