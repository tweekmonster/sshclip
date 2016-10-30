package sshclip

import (
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type SSHRegister struct {
	ch   ssh.Channel
	reqs <-chan *ssh.Request
	dead bool
}

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
		ch:   ch,
		reqs: reqs,
	}

	go func() {
		for req := range reqs {
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
		cli.dead = true
	}()

	return cli, nil

}

func (c *SSHRegister) Close() error {
	return c.ch.Close()
}

func (c *SSHRegister) Get(reg uint8) (RegisterItem, error) {
	return GetRegister(c.ch, reg)
}

func (c *SSHRegister) Put(reg uint8, attrs uint8, data []byte) (err error) {
	return PutRegister(c.ch, reg, attrs, data)
}
