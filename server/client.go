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
		op, attrs, index, err := sshclip.ReadPayloadHeader(s.channel)
		if err != nil {
			if err != io.EOF {
				sshclip.Elog(s, "- (Header)", err)
			}
			break
		}

		sshclip.Dlog("Op: %02d, Register: '%c', Attrs: %08b", op, index, attrs)

		switch op {
		case sshclip.OpGet:
			item, err := s.storage.Get(index)
			if err != nil {
				sshclip.Elog(s, "- (Get)", err)
				continue
			}

			n, err := sshclip.SendRegister(s.channel, item)
			if err != nil {
				break
			}

			sshclip.Dlog("Sent %d bytes from register '%c'", n, index)

		case sshclip.OpPut:
			data, err := sshclip.ReadPayload(s.channel)
			if err != nil {
				if err != io.EOF {
					sshclip.Elog(s, "- (Payload)", err)
				}
				break
			}

			s.storage.Put(index, data, attrs)
		}
	}
}
