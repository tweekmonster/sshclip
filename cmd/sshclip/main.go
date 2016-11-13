package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	termutil "github.com/andrew-d/go-termutil"
	"github.com/nightlyone/lockfile"
	"github.com/tweekmonster/sshclip"
	"github.com/tweekmonster/sshclip/client"
	"github.com/tweekmonster/sshclip/platform"
	"github.com/tweekmonster/sshclip/server"
	cli "gopkg.in/urfave/cli.v2"
)

var errLocalIsServer = errors.New("refusing to run monitor along side the server")

func lockPath(name string) string {
	return filepath.Join(os.TempDir(), "sshclip_"+name+".lock")
}

func lockFile(name string, timeout time.Duration) (lockfile.Lockfile, error) {
	lockfilePath := lockPath(name)
	lock, err := lockfile.New(lockfilePath)
	if err != nil {
		return "", err
	}

	stop := time.Now().Add(timeout)
	t := time.NewTicker(time.Millisecond * 100)
	for n := range t.C {
		if err := lock.TryLock(); err != nil {
			if _, ok := err.(lockfile.TemporaryError); ok {
				if n.After(stop) {
					return "", err
				}
				continue
			}
		}
		break
	}

	sshclip.Dlog("Lock acquired: %s", lock)

	return lock, nil
}

func runServer(c *cli.Context) error {
	sshclip.LogPrefix = "server"
	lock, err := lockFile("server", time.Second*5)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	return server.Listen(c.String("host"), c.Int("port"))
}

func runMonitor(c *cli.Context) error {
	if same, _ := platform.LocalIsServer(c.String("host")); same {
		return errLocalIsServer
	}

	sshclip.LogPrefix = "monitor"
	lock, err := lockFile("monitor", time.Second*5)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	if err := client.MonitorListen(c.String("host"), c.Int("port")); err != nil {
		return err
	}

	return nil
}

func runClient(c *cli.Context) (err error) {
	sshclip.LogPrefix = "client"

	if c.Bool("keys") {
		return manageKeys(c.String("host"), c.Int("port"))
	}

	remoteType := "monitor"
	restart := c.Bool("restart")

	if same, _ := platform.LocalIsServer(c.String("host")); same {
		remoteType = "server"
	}

	connect := func(timeout time.Duration) (io.ReadWriter, error) {
		if remoteType == "server" {
			sshClient, err := sshclip.SSHClientConnect(c.String("host"), c.Int("port"))
			if err != nil {
				return nil, err
			}

			ch, reqs, err := sshClient.OpenChannel("sshclip", nil)
			if err != nil {
				return nil, err
			}

			go ssh.DiscardRequests(reqs)
			return ch, nil
		}

		return client.MonitorConnect(timeout)
	}

	conn, err := connect(0)

	if err != nil || restart {
		lock := lockfile.Lockfile(lockPath(remoteType))
		if proc, _ := lock.GetOwner(); proc != nil {
			if err := proc.Signal(syscall.SIGTERM); err == nil {
				t := time.NewTicker(time.Millisecond * 10)
				attempts := 100
				for _ = range t.C {
					if attempts <= 0 {
						return fmt.Errorf("Failed to restart %s", remoteType)
					}
					if _, err := lock.GetOwner(); err != nil {
						break
					}
					attempts--
				}
			}
		}

		err = io.EOF
	}

	if err != nil {
		if err := client.Spawn(c.String("host"), c.Int("port")); err != nil {
			return err
		}

		attempts := 100
		t := time.NewTicker(time.Millisecond * 10)
		for _ = range t.C {
			conn, err = connect(time.Second * 5)
			if err == nil {
				break
			}
			if err != nil && attempts <= 0 {
				return err
			}
			attempts--
		}
	}

	reg := c.String("reg")
	if len(reg) != 1 {
		return errors.New("Invalid register")
	}

	flags := uint8(c.Int("flags"))

	if !termutil.Isatty(os.Stdin.Fd()) {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		return sshclip.PutRegister(conn, reg[0], flags, data)
	}

	item, err := sshclip.GetRegister(conn, reg[0])
	if err != nil {
		return err
	}

	if item.Size() > 0 {
		if _, err := io.CopyN(os.Stdout, item, int64(item.Size())); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	stdFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "host",
			Value: "127.0.0.1",
			Usage: "Listen address",
		},
		&cli.IntFlag{
			Name:  "port",
			Value: 2222,
			Usage: "Port",
		},
	}

	clientFlags := append(stdFlags, []cli.Flag{
		&cli.StringFlag{
			Name:  "reg",
			Usage: "Register",
		},
		&cli.IntFlag{
			Name:  "flags",
			Usage: "Flags",
		},
		&cli.BoolFlag{
			Name:  "restart",
			Usage: "Restart monitor",
		},
		&cli.BoolFlag{
			Name:  "keys",
			Usage: "Key management",
		},
	}...)

	commands := []*cli.Command{
		&cli.Command{
			Name:   "server",
			Usage:  "Starts the server",
			Flags:  stdFlags,
			Action: runServer,
		},
		&cli.Command{
			Name:   "monitor",
			Usage:  "Starts the local monitor",
			Flags:  stdFlags,
			Action: runMonitor,
		},
	}

	app := &cli.App{
		Name:   "sshclip",
		Usage:  "Clipboard service over SSH",
		Flags:  clientFlags,
		Action: runClient,
	}

	app.Commands = commands

	if err := app.Run(os.Args); err != nil {
		sshclip.Elog(err)
	}
}
