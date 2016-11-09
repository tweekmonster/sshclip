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
var errDenied = errors.New("denied")

type server struct {
	sync.RWMutex
	conn       net.Listener
	config     ssh.ServerConfig
	keysFile   string
	clients    []*clientConnection
	storage    *sshclip.MemoryRegister
	seenClient bool
}

func (s *server) authenticate(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// Unknown keys aren't allowed to do anything until an existing client
	// approves its membership.

	keyRecord := sshclip.NewPublicKeyRecord(key, conn.RemoteAddr())

	if sshclip.KeyExists("rejected", keyRecord) {
		return nil, errDenied
	}

	if !sshclip.KeyExists("authorized", keyRecord) {
		if !sshclip.DataFileExists("keys/authorized") {
			// Special case where the first key is added.  Only works if an
			// authorized connection has never occurred and the authorized key file
			// doesn't exist.
			if !s.seenClient {
				keyRecord.State = "authorized"
				sshclip.AddKey("authorized", keyRecord)
			} else {
				// Don't write the key to the rejected file if the authorized file
				// doesn't exist.  Otherwise, deleting the authorized file while the
				// server is running could cause the server to indiscriminately deny
				// all connections.
				return nil, errDenied
			}
		} else {
			sshclip.AddKey("rejected", keyRecord)
			return nil, errPubKeyPending
		}
	}

	s.seenClient = true
	perm := &ssh.Permissions{
		Extensions: map[string]string{
			"pubkey": string(key.Marshal()),
		},
	}

	sshclip.Dlog("Authenticated client:", conn.RemoteAddr(), "Fingerprint:", sshclip.FingerPrint(key))

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
