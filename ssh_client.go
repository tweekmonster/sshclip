package sshclip

import (
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

func SSHClientConnect(host string, port int) (*ssh.Client, error) {
	clientKey, err := GetClientKey()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(clientKey),
		},
	}

	return ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), config)
}
