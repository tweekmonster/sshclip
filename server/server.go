package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"sync"

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
	clients     []*clientConnection
	storage     *sshclip.MemoryRegister
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

	return sshclip.ListenLoop(s.conn, func(c net.Conn) {
		cli, err := newClientConnection(c, s)
		if err != nil {
			sshclip.Elog(err)
		} else {
			s.addClient(cli)
		}
	})
}

func (s *server) addClient(c *clientConnection) {
	s.Lock()
	defer s.Unlock()

	s.clients = append(s.clients, c)
}

func (s *server) removeClient(c *clientConnection) {
	s.Lock()
	defer s.Unlock()

	clients := s.clients
	s.clients = s.clients[:0]

	for _, cli := range clients {
		if cli != c {
			s.clients = append(s.clients, cli)
		}
	}
}

// Send a request to all clients on a specific channel.
func (s *server) broadcast(channel, name string, data []byte) {
	sshclip.Dlog("Broadcast")
	s.Lock()
	defer s.Unlock()
	sshclip.Dlog("Broadcast start")

	clients := s.clients
	s.clients = s.clients[:0]

	for _, cli := range clients {
		sshclip.Dlog("Broadcasting to %s", cli.conn.RemoteAddr())
		cli.Lock()
		if ch, ok := cli.channels[channel]; ok {
			if _, err := ch.SendRequest(name, false, data); err == nil {
				s.clients = append(s.clients, cli)
			}
		}
		cli.Unlock()
	}
}

// Services the MemoryRegister's notification channel and notifies of register
// changes.
func (s *server) notificationRoutine() {
	msgBytes := make([]byte, 2)

	for msg := range s.storage.Notify {
		binary.BigEndian.PutUint16(msgBytes, msg)
		sshclip.Dlog("Notification: %#v", msgBytes)
		op := msgBytes[0]
		reg := msgBytes[1]

		switch op {
		case sshclip.OpPut:
			item, err := s.storage.GetItem(reg)
			if err == nil {
				var b bytes.Buffer
				binary.Write(&b, binary.BigEndian, item.Hash())
				s.broadcast("sshclip", "sync", b.Bytes())
			}
		}
	}
}

// Listen starts the SSH server for new connections.
func Listen(host string, port int) error {
	conn, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}

	storage := sshclip.NewMemoryRegister()

	s := &server{
		conn:    conn,
		storage: storage,
	}

	go s.notificationRoutine()

	return s.run()
}
