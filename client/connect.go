package client

import (
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/tweekmonster/sshclip"
	"golang.org/x/crypto/ssh"
)

func RemoteConnect(host string, port int) error {
	clientKey, err := sshclip.GetClientKey()
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(clientKey),
		},
	}

	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), config)
	if err != nil {
		return err
	}

	session, err := New(conn)
	if err != nil {
		return err
	}
	defer session.Close()

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	session.Put('*', sshclip.VimBlockAttr, input)

	time.Sleep(time.Second * 5)

	back, err := session.Get('*')
	if err != nil {
		sshclip.Elog(err)
	} else {
		sshclip.Dlog("Response: %#v", back)
	}

	time.Sleep(time.Second * 10)

	return nil
}
