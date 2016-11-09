package server

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/tweekmonster/sshclip"

	"golang.org/x/crypto/ssh"
)

type clientConnection struct {
	sync.Mutex
	server   *server
	conn     *ssh.ServerConn
	channels map[string]ssh.Channel
	key      ssh.PublicKey
}

func newClientConnection(c net.Conn, s *server) (*clientConnection, error) {
	conn, channels, requests, err := ssh.NewServerConn(c, &s.config)
	if err != nil {
		return nil, err
	}

	cli := &clientConnection{
		server:   s,
		conn:     conn,
		channels: make(map[string]ssh.Channel),
	}

	if keyStr, ok := conn.Permissions.Extensions["pubkey"]; ok {
		cli.key, err = ssh.ParsePublicKey([]byte(keyStr))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("pubkey missing from extensions")
	}

	go ssh.DiscardRequests(requests)
	go cli.handleChannels(channels)

	return cli, nil
}

func (c *clientConnection) handleChannels(channels <-chan ssh.NewChannel) {
	for ch := range channels {
		chname := ch.ChannelType()
		switch {
		case strings.HasPrefix(chname, "sshclip"):
			// "sshclip" is the standard get/put/notification channel.
			// "sshclip-keys" is the channel for listing/approving client keys.
			newChan, newRequests, err := ch.Accept()
			if err != nil {
				break
			}

			go ssh.DiscardRequests(newRequests)
			go c.serviceChannel(chname, newChan)
		default:
			ch.Reject(ssh.UnknownChannelType, "Unknown channel type: "+chname)
		}
	}

	c.server.removeClient(c)
}

func (c *clientConnection) serviceChannel(name string, ch ssh.Channel) {
	sshclip.Log("New %s channel from %s", name, c.conn.RemoteAddr())

	c.Lock()
	c.channels[name] = ch
	c.Unlock()

	var err error

loop:
	for {
		switch name {
		case "sshclip":
			err = sshclip.HandlePayload(c.server.storage, ch)
		case "sshclip-keys":
			err = sshclip.HandleKeyPayload(c.key, ch)
		default:
			break loop
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			sshclip.Elog(err)
		}
	}

	c.Lock()
	delete(c.channels, name)
	c.Unlock()
}
