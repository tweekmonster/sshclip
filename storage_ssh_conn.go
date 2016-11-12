package sshclip

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

func (c *SSHRegister) tryConnect() error {
	conn, err := SSHClientConnect(c.host, c.port)
	if err != nil {
		return err
	}

	ch, reqs, err := conn.OpenChannel("sshclip", nil)
	if err != nil {
		return err
	}

	c.ch = ch
	c.requests = reqs
	c.putChan = make(chan RegisterItem, 4)

	return nil
}

func (c *SSHRegister) handleRequest(req *ssh.Request) {
	if req.Type == "sync" {
		var syncItem RegisterItemHash

		if err := binary.Read(bytes.NewBuffer(req.Payload), binary.BigEndian, &syncItem); err == nil {
			if item, err := c.storage.GetItem(syncItem.Register); err == nil {
				if !item.EqualsHash(syncItem) {
					Dlog("Sync register: %c", syncItem.Register)
					c.syncRegister(syncItem.Register)
				}
			}
		}
	}

	if req.WantReply {
		req.Reply(false, nil)
	}
}

func (c *SSHRegister) deferredPutRoutine() {
	for item := range c.putChan {
		data := make([]byte, item.Size())
		if _, err := item.Read(data); err != nil {
			continue
		}
		if c.ch != nil {
			PutRegister(c.ch, uint8(item.Index()), item.Attributes(), data)
		}
	}

	Dlog("Put routine stopped")
}

func (c *SSHRegister) Run() {
	var retryCount int
	var retryDelay time.Duration
	stopEvents := CreateListener(Interrupt, Terminate)

mainloop:
	for {
		c.ch = nil
		c.requests = nil

		select {
		case <-stopEvents:
			break mainloop
		default:
			if err := c.tryConnect(); err != nil {
				if _, ok := err.(net.Error); !ok {
					// Only retry on net.Error
					Elog(err)
					break mainloop
				}

				// Start retrying every 1 second and gradually back off to retrying
				// every 5 seconds.
				if retryCount <= 50 && retryCount%10 == 0 {
					retryCount++
					retryDelay = time.Second * time.Duration((retryCount/10)+1)
				}

				Dlog("Retry in: %s", retryDelay)
				time.Sleep(retryDelay)
				continue mainloop
			} else {
				Dlog("SSHRegister connected")
				retryCount = 0
				retryDelay = 0
				go c.deferredPutRoutine()
			}
		}

	requestloop:
		for {
			select {
			case <-stopEvents:
				break mainloop
			case req, ok := <-c.requests:
				if !ok {
					break requestloop
				}

				go c.handleRequest(req)
			}
		}

		c.Close()
	}

	Dlog("SSHRegister shutdown")
}

func (c *SSHRegister) Close() error {
	if c.putChan != nil {
		close(c.putChan)
		c.putChan = nil
	}

	if c.ch != nil {
		return c.ch.Close()
	}

	return nil
}
