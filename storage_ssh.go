package sshclip

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHRegister passes get/put requests to the remote SSH server and caches
// registers locally.  It opens an additional event channel to monitor
// registers set by other clients.
type SSHRegister struct {
	sync.RWMutex
	ch      ssh.Channel
	putChan chan RegisterItem
	items   map[uint8]RegisterItem
}

// NewSSHRegister creates a new SSHRegister.
func NewSSHRegister(host string, port int) (*SSHRegister, error) {
	clientKey, err := GetClientKey()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(clientKey),
		},
	}

	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), config)
	if err != nil {
		return nil, err
	}

	ch, reqs, err := conn.OpenChannel("sshclip", nil)
	if err != nil {
		return nil, err
	}

	cli := &SSHRegister{
		ch:      ch,
		putChan: make(chan RegisterItem, 4),
		items:   make(map[uint8]RegisterItem),
	}

	go cli.handleRequests(reqs)
	go cli.handleDeferredPut()

	return cli, nil

}

func (c *SSHRegister) syncRegister(reg uint8) {
	t := time.Now()
	Dlog("Syncing register '%c'", reg)
	item, err := GetRegister(c.ch, reg)
	if err != nil {
		return
	}

	c.Lock()
	existing, ok := c.items[uint8(item.Index())]
	if ok && t.Before(existing.Time()) {
		// The cache was updated in the time it took to get the remote register.
		Dlog("Cache is newer than the retrieved register")
		return
	}
	c.items[uint8(item.Index())] = item
	c.Unlock()
}

func (c *SSHRegister) handleRequests(requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "sync":
			Dlog("Syncing: %#v", req.Payload)
			var syncItem RegisterItemHash
			if err := binary.Read(bytes.NewBuffer(req.Payload), binary.BigEndian, &syncItem); err == nil {
				c.Lock()
				if item, ok := c.items[syncItem.Register]; ok {
					if !item.EqualsHash(syncItem) {
						// Use a goroutine to prevent blocking normal caching.
						go c.syncRegister(syncItem.Register)
					}
				}
				c.Unlock()
			}
		}

		if req.WantReply {
			req.Reply(false, nil)
		}
	}

	c.Close()

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

	c.RLock()
	if item, ok := c.items[reg]; ok {
		c.RUnlock()
		return item, nil
	}

	item, err := GetRegister(c.ch, reg)
	if err != nil {
		c.RUnlock()
		return nil, err
	}
	c.RUnlock()

	c.Lock()
	c.items[reg] = item
	c.Unlock()

	return item, nil
}

// Put register data into the remote SSH server.
func (c *SSHRegister) Put(reg uint8, attrs uint8, data []byte) (err error) {
	if !IsValidIndex(reg) {
		return ErrInvalidIndex
	}

	c.Lock()
	item := NewMemoryRegisterItem(reg, attrs, data)
	c.items[reg] = item
	c.Unlock()

	c.putChan <- item

	return nil
}

func (c *SSHRegister) List() ([]RegisterItemHash, error) {
	c.Lock()
	defer c.Unlock()
	return ListRegisters(c.ch)
}
