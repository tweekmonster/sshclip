package sshclip

import (
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHRegister passes get/put requests to the remote SSH server and caches
// registers locally.  It opens an additional event channel to monitor
// registers set by other clients.
type SSHRegister struct {
	ch         ssh.Channel
	putChan    chan RegisterItem
	storage    *MemoryRegister
	requests   <-chan *ssh.Request
	host       string
	port       int
	connWaiter chan bool
	connOnce   sync.Once
}

// NewSSHRegister creates a new SSHRegister.
func NewSSHRegister(host string, port int, storage *MemoryRegister) (*SSHRegister, error) {
	// XXX: connWaiter and connOnce are used to signal that there was at least one
	// attempt to connect.  With the many goroutines in use, it's possible to call
	// Get() or Put() before a connection has been made.  This would cause the
	// client that spawns the monitor to recieve nothing.
	// A refactor should be considered in the future.
	return &SSHRegister{
		storage:    storage,
		host:       host,
		port:       port,
		connWaiter: make(chan bool),
	}, nil
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

// Get register data from the remote SSH server.
func (c *SSHRegister) Get(reg uint8) (RegisterItem, error) {
	c.connWaiter <- true

	if !IsValidIndex(reg) {
		return nil, ErrInvalidIndex
	}

	if item, err := c.storage.GetItem(reg); err == nil || c.ch == nil {
		return item, err
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
	c.connWaiter <- true

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

	if c.ch != nil && c.putChan != nil {
		c.putChan <- item
	}

	return nil
}

func (c *SSHRegister) List() ([]RegisterItemHash, error) {
	c.connWaiter <- true
	return c.storage.List()
}
