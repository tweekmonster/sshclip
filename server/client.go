package server

import (
	"io"
	"net"
	"sync"

	"github.com/tweekmonster/sshclip"

	"golang.org/x/crypto/ssh"
)

type clientConnection struct {
	sync.Mutex
	server   *server
	conn     *ssh.ServerConn
	channels map[string]ssh.Channel
}

func newClientConnection(c net.Conn, s *server) (*clientConnection, error) {
	conn, channels, requests, err := ssh.NewServerConn(c, &s.config)
	if err != nil {
		return nil, err
	}

	go ssh.DiscardRequests(requests)

	cli := &clientConnection{
		server:   s,
		conn:     conn,
		channels: make(map[string]ssh.Channel),
	}

	go cli.handleChannels(channels)

	return cli, nil
}

func (c *clientConnection) handleChannels(channels <-chan ssh.NewChannel) {
	for ch := range channels {
		// 'sshclip' is the standard get/put/notification channel.
		chname := ch.ChannelType()
		switch chname {
		case "sshclip":
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

	for {
		err := sshclip.HandlePayload(c.server.storage, ch)
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
