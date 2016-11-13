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
	"github.com/tweekmonster/sshclip/platform"
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

	cmd := "monitor"
	if same, err := platform.LocalIsServer(host); err == nil && same {
		cmd = "server"
	}

	proc, err := os.StartProcess(exe, []string{exe, cmd, "--host", host, "--port", strconv.Itoa(port)}, attr)
	if err != nil {
		return err
	}

	return proc.Release()
}

// MonitorListen listens on a unix socket or named pipe to receive local queries.
// The purpose of this is to maintain a fast local cache to keep the clients
// responsive.
func MonitorListen(sshHost string, sshPort int) error {
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
		if platform.ClipboardEnabled() {
			sshclip.Dlog("Starting system clipboard monitor")
			go platform.ClipboardMonitor(sshClient, '+')
		}

		sshClient.Run()
		platform.ClipboardMonitorStop()
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
						if err := platform.ClipboardPut(data); err != nil {
							sshclip.Elog("Error setting clipboard:", err)
						} else if platform.NotificationsEnabled() {
							platform.PostNotification("Clipboard updated from remote")
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

// MonitorConnect connects to the local monitoring server.
func MonitorConnect(timeout time.Duration) (pipe net.Conn, err error) {
	return pipeDial(timeout)
}
