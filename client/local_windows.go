package client

import (
	"net"
	"time"

	npipe "gopkg.in/natefinch/npipe.v2"
)

func pipeListen() (net.Listener, error) {
	return npipe.Listen(`\\.\pipe\sshclip`)
}

func pipeDial(timeout time.Duration) (net.Conn, error) {
	if timeout > 0 {
		return npipe.DialTimeout(`\\.\pipe\sshclip`, timeout)
	}
	return npipe.Dial(`\\.\pipe\sshclip`)
}
