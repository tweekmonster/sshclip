// +build !windows

package client

import (
	"net"
	"os"
	"time"
)

func pipeListen() (net.Listener, error) {
	if err := os.Remove("/tmp/sshclip.sock"); err != nil {
		return nil, err
	}
	return net.Listen("unix", "/tmp/sshclip.sock")
}

func pipeDial(timeout time.Duration) (net.Conn, error) {
	if timeout > 0 {
		return net.DialTimeout("unix", "/tmp/sshclip.sock", timeout)
	}

	return net.Dial("unix", "/tmp/sshclip.sock")
}
