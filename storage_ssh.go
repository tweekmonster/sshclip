package sshclip

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHRegister passes get/put requests to the remote SSH server and caches
// registers locally.  It opens an additional event channel to monitor
// registers set by other clients.
type SSHRegister struct {
	sync.RWMutex
	ch       ssh.Channel
	putChan  chan RegisterItem
	storage  *MemoryRegister
	requests <-chan *ssh.Request
}

// NewSSHRegister creates a new SSHRegister.
func NewSSHRegister(host string, port int, storage *MemoryRegister) (*SSHRegister, error) {
	conn, err := SSHClientConnect(host, port)
	if err != nil {
		return nil, err
	}

	ch, reqs, err := conn.OpenChannel("sshclip", nil)
	if err != nil {
		return nil, err
	}

	cli := &SSHRegister{
		ch:       ch,
		putChan:  make(chan RegisterItem, 4),
		storage:  storage,
		requests: reqs,
	}

	return cli, nil

}

func (c *SSHRegister) putItem(item RegisterItem, notify bool) error {
	data := make([]byte, item.Size())
	if _, err := io.ReadAtLeast(item, data, item.Size()); err != nil {
		return err
	}

	if notify {
		return c.storage.Put(uint8(item.Index()), item.Attributes(), data)
	}

	return c.storage.PutItem(uint8(item.Index()), item.Attributes(), data)
}

func (c *SSHRegister) syncRegister(reg uint8) {
	t := time.Now()
	Dlog("Syncing register '%c'", reg)
	item, err := GetRegister(c.ch, reg)
	if err != nil {
		return
	}

	existing, err := c.storage.GetItem(uint8(item.Index()))
	if err == nil && t.Before(existing.Time()) {
		// The cache was updated in the time it took to get the remote register.
		Dlog("Cache is newer than the retrieved register")
		return
	}

	if err := c.putItem(item, true); err != nil {
		Elog(err)
	}
}

func (c *SSHRegister) Run() {
	go c.handleDeferredPut()
	defer c.Close()

	for req := range c.requests {
		switch req.Type {
		case "sync":
			var syncItem RegisterItemHash
			if err := binary.Read(bytes.NewBuffer(req.Payload), binary.BigEndian, &syncItem); err == nil {
				if item, err := c.storage.GetItem(syncItem.Register); err == nil {
					if !item.EqualsHash(syncItem) {
						Dlog("Sync register: %c", syncItem.Register)
						// Use a goroutine to prevent blocking normal caching.
						go c.syncRegister(syncItem.Register)
					}
				}
			}
		}

		if req.WantReply {
			req.Reply(false, nil)
		}
	}

	Dlog("SSHRegister shutdown")
}

func (c *SSHRegister) handleDeferredPut() {
	for item := range c.putChan {
		data := make([]byte, item.Size())
		if _, err := item.Read(data); err != nil {
			continue
		}
		PutRegister(c.ch, uint8(item.Index()), item.Attributes(), data)
	}
}

func (c *SSHRegister) Close() error {
	close(c.putChan)
	return c.ch.Close()
}

// Get register data from the remote SSH server.
func (c *SSHRegister) Get(reg uint8) (RegisterItem, error) {
	if !IsValidIndex(reg) {
		return nil, ErrInvalidIndex
	}

	if item, err := c.storage.GetItem(reg); err == nil {
		return item, nil
	}

	item, err := GetRegister(c.ch, reg)
	if err != nil {
		return nil, err
	}

	if err := c.putItem(item, false); err != nil {
		return nil, err
	}

	return item, nil
}

// Put register data into the remote SSH server.
func (c *SSHRegister) Put(reg uint8, attrs uint8, data []byte) (err error) {
	if !IsValidIndex(reg) {
		return ErrInvalidIndex
	}

	if err := c.storage.PutItem(reg, attrs, data); err != nil {
		return err
	}

	item, err := c.storage.GetItem(reg)
	if err != nil {
		return err
	}

	c.putChan <- item

	return nil
}

func (c *SSHRegister) List() ([]RegisterItemHash, error) {
	return c.storage.List()
}
