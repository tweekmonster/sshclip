package server

import (
	"io"

	"github.com/tweekmonster/sshclip"
	"golang.org/x/crypto/ssh"
)

type serverClient struct {
	conn    *ssh.ServerConn
	key     ssh.PublicKey
	channel ssh.Channel
	storage sshclip.Register
}

func newClient(storage sshclip.Register, conn *ssh.ServerConn, key ssh.PublicKey, ch ssh.NewChannel) (*serverClient, error) {
	channel, requests, err := ch.Accept()
	if err != nil {
		return nil, err
	}

	s := &serverClient{
		conn:    conn,
		key:     key,
		channel: channel,
		storage: storage,
	}

	go s.sessionLoop()
	go ssh.DiscardRequests(requests)

	return s, nil
}

func (s *serverClient) String() string {
	return s.conn.RemoteAddr().String()
}

func (s *serverClient) sessionLoop() {
	for {
		err := sshclip.HandlePayload(s.storage, s.channel)
		if err != nil {
			if err == io.EOF {
				break
			}

			sshclip.Elog(err)
		}
	}
}
