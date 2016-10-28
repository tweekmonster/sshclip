package server

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/tweekmonster/sshclip"

	"golang.org/x/crypto/ssh"
)

var errPubKeyPending = errors.New("PublicKey is pending approval")

type server struct {
	sync.RWMutex
	conn        net.Listener
	config      ssh.ServerConfig
	keysFile    string
	pendingKeys []ssh.PublicKey
	clients     []*ssh.ServerConn
	storage     sshclip.Register
}

func (s *server) authenticate(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// Unknown keys aren't allowed to do anything until an existing client
	// approves its membership.
	keyBytes := ssh.MarshalAuthorizedKey(key)

	// Pre-fail if the key is pending approval from an existing client.
	s.RLock()
	for _, k := range s.pendingKeys {
		if bytes.Equal(keyBytes, ssh.MarshalAuthorizedKey(k)) {
			sshclip.Dlog("Key is pending:", sshclip.FingerPrint(key))
			s.RUnlock()
			return nil, errPubKeyPending
		}
	}
	s.RUnlock()

	if !sshclip.IsAuthorizedKey(key) {
		s.Lock()
		s.pendingKeys = append(s.pendingKeys, key)
		s.Unlock()
		return nil, errPubKeyPending
	}

	perm := &ssh.Permissions{
		Extensions: map[string]string{
			"pubkey": string(keyBytes),
		},
	}

	sshclip.Dlog("Authenticatd client:", conn.RemoteAddr(), "Fingerprint:", sshclip.FingerPrint(key))

	return perm, nil
}

func (s *server) handleConnection(client net.Conn) error {
	sshclip.Log("Connection from:", client.RemoteAddr())
	conn, channels, requests, err := ssh.NewServerConn(client, &s.config)
	if err != nil {
		return err
	}

	s.Lock()
	s.clients = append(s.clients, conn)
	s.Unlock()

	var pubkey ssh.PublicKey

	if keyStr, ok := conn.Permissions.Extensions["pubkey"]; ok {
		pubkey, _, _, _, err = ssh.ParseAuthorizedKey([]byte(keyStr))
		if err != nil {
			return err
		}
	}

	go ssh.DiscardRequests(requests)

	var cli *serverClient

	for ch := range channels {
		// 'sshclip' is the standard get/put channel.  'monitor' will be a
		// broadcast channel.
		if chtype := ch.ChannelType(); chtype != "sshclip" {
			ch.Reject(ssh.UnknownChannelType, "Unknown channel type: "+chtype)
			continue
		}

		cli, err = newClient(s.storage, conn, pubkey, ch)
		if err != nil {
			return err
		}

		sshclip.Log("New sshclip channel from", conn.RemoteAddr())

		if cli != nil {
			go func() {
				for ch := range channels {
					ch.Reject(ssh.Prohibited, "No")
				}
			}()
		}
	}

	return nil
}

func (s *server) run() error {
	hostKeySigner, err := sshclip.GetHostKey()
	if err != nil {
		return err
	}

	sshclip.Dlog("Host fingerprint:", sshclip.FingerPrint(hostKeySigner.PublicKey()))

	s.config = ssh.ServerConfig{
		PublicKeyCallback: s.authenticate,
	}
	s.config.AddHostKey(hostKeySigner)

	for {
		conn, err := s.conn.Accept()
		if err != nil {
			sshclip.Elog(err)
			continue
		}

		go func() {
			if err := s.handleConnection(conn); err != nil {
				if err != io.EOF {
					sshclip.Elog(err)
				}
			}

			sshclip.Dlog("Session ended for", conn.RemoteAddr())
			s.cleanupConnections()
		}()
	}
}

func (s *server) cleanupConnections() {
	s.Lock()
	clients := s.clients
	s.clients = s.clients[:0]

	for _, conn := range clients {
		_, _, err := conn.SendRequest("keepalive", true, nil)
		if err != nil {
			// TODO: Find out if Close() does anything useful if the client
			// connection is actually closed.
			conn.Close()
			sshclip.Dlog("Dead Client", conn.RemoteAddr(), err)
		} else {
			s.clients = append(s.clients, conn)
		}
	}
	s.Unlock()
}

func Listen(host string, port int) error {
	conn, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}

	s := &server{
		conn:    conn,
		storage: sshclip.NewMemoryRegister(),
	}

	go func() {
		t := time.Tick(time.Minute * 2)
		for _ = range t {
			s.cleanupConnections()
		}
	}()

	return s.run()
}
