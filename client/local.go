package client

import (
	"net"
	"os"
	"strconv"
	"time"

	"github.com/kardianos/osext"
	"github.com/tweekmonster/sshclip"
)

// Spawn starts a local monitoring process.
func Spawn(host string, port int) error {
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

	proc, err := os.StartProcess(exe, []string{exe, "monitor", "--listen", host, "--port", strconv.Itoa(port)}, attr)
	if err != nil {
		return err
	}

	return proc.Release()
}

// LocalListen listens on a unix socket or named pipe to receive local queries.
// The purpose of this is to maintain a fast local cache to keep the clients
// responsive.
func LocalListen(sshHost string, sshPort int) error {
	sshClient, err := sshclip.NewSSHRegister(sshHost, sshPort)
	if err != nil {
		sshclip.Elog(err)
		return err
	}

	register := sshclip.NewCacheStorage(sshClient)

	conn, err := pipeListen()
	if err != nil {
		sshclip.Elog(err)
		return err
	}

	for {
		localClient, err := conn.Accept()
		if err != nil {
			sshclip.Elog(err)
			continue
		}

		go func() {
			sshclip.HandlePayload(register, localClient)
			localClient.Close()
		}()
	}
}

// LocalConnect connects to the local monitoring server.
func LocalConnect(timeout time.Duration) (pipe net.Conn, err error) {
	return pipeDial(timeout)
}
