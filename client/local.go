package client

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/kardianos/osext"
	"github.com/tweekmonster/sshclip"
	"github.com/tweekmonster/sshclip/clipboard"
)

// Spawn starts a local monitoring process.
func Spawn(host string, port int) error {
	sshclip.Dlog("Spawning monitor")
	exe, err := osext.Executable()
	if err != nil {
		return err
	}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	stdout, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	stderr, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	var attr = &os.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []*os.File{
			stdin,
			stdout,
			stderr,
		},
	}

	proc, err := os.StartProcess(exe, []string{exe, "monitor", "--host", host, "--port", strconv.Itoa(port)}, attr)
	if err != nil {
		return err
	}

	return proc.Release()
}

// LocalListen listens on a unix socket or named pipe to receive local queries.
// The purpose of this is to maintain a fast local cache to keep the clients
// responsive.
func LocalListen(sshHost string, sshPort int) error {
	storage := sshclip.NewMemoryRegister()
	sshClient, err := sshclip.NewSSHRegister(sshHost, sshPort, storage)
	if err != nil {
		sshclip.Elog(err)
		return err
	}

	conn, err := pipeListen()
	if err != nil {
		sshclip.Elog(err)
		return err
	}

	go func() {
		if clipboard.Enabled() {
			sshclip.Dlog("Starting system clipboard monitor")
			go clipboard.Monitor(sshClient, '+')
		}

		sshClient.Run()
		clipboard.MonitorStop()
		sshclip.ListenLoopStop()
	}()

	go func() {
		msgBytes := make([]byte, 2)

		for msg := range storage.Notify {
			binary.BigEndian.PutUint16(msgBytes, msg)
			op := msgBytes[0]
			reg := msgBytes[1]

			if op == sshclip.OpPut && reg == '+' {
				if item, err := storage.GetItem(reg); err == nil {
					data := make([]byte, item.Size())
					if _, err := io.ReadAtLeast(item, data, item.Size()); err == nil {
						if err := clipboard.Put(data); err != nil {
							sshclip.Elog("Error setting clipboard:", err)
						}
					}
				}
			}
		}
	}()

	return sshclip.ListenLoop(conn, func(c net.Conn) {
		sshclip.HandlePayload(sshClient, c)
		c.Close()
	})
}

// LocalConnect connects to the local monitoring server.
func LocalConnect(timeout time.Duration) (pipe net.Conn, err error) {
	return pipeDial(timeout)
}
