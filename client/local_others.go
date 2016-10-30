// +build !windows

package client

import (
	"net"
	"os"
	"time"
)

func pipeListen() (net.Listener, error) {
	os.Remove("/tmp/sshclip.sock")
	return net.Listen("unix", "/tmp/sshclip.sock")
}

func pipeDial(timeout time.Duration) (net.Conn, error) {
	if timeout > 0 {
		return net.DialTimeout("unix", "/tmp/sshclip.sock", timeout)
	}

	return net.Dial("unix", "/tmp/sshclip.sock")
}
