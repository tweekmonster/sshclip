package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	termutil "github.com/andrew-d/go-termutil"
	"github.com/nightlyone/lockfile"
	"github.com/tweekmonster/sshclip"
	"github.com/tweekmonster/sshclip/client"
	"github.com/tweekmonster/sshclip/server"
	cli "gopkg.in/urfave/cli.v2"
)

func lockPath(name string) string {
	return filepath.Join(os.TempDir(), "sshclip_"+name+".lock")
}

func lockFile(name string, lockRetry int) (lockfile.Lockfile, error) {
	lockfilePath := lockPath(name)
	lock, err := lockfile.New(lockfilePath)
	if err != nil {
		return "", err
	}

	for tryCount := 0; tryCount < lockRetry; tryCount++ {
		if err := lock.TryLock(); err != nil {
			if _, ok := err.(lockfile.TemporaryError); ok {
				if tryCount >= lockRetry-1 {
					return "", err
				}
				time.Sleep(time.Second)
				sshclip.Dlog("Retrying " + name + " lock.")
				continue
			} else {
				return "", err
			}
		}
		break
	}

	sshclip.Dlog("Lock acquired: %s", lock)

	return lock, nil
}

func runServer(c *cli.Context) error {
	sshclip.LogPrefix = "server"
	lock, err := lockFile("server", 10)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	return server.Listen(c.String("host"), c.Int("port"))
}

func runMonitor(c *cli.Context) error {
	sshclip.LogPrefix = "monitor"
	lock, err := lockFile("monitor", 10)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	if err := client.LocalListen(c.String("host"), c.Int("port")); err != nil {
		return err
	}

	return nil
}

func runClient(c *cli.Context) (err error) {
	sshclip.LogPrefix = "client"
	conn, err := client.LocalConnect(0)
	if err != nil {
		if err := client.Spawn(c.String("host"), c.Int("port")); err != nil {
			return err
		}

		attempts := 100
		t := time.NewTicker(time.Millisecond * 10)
		for _ = range t.C {
			conn, err = client.LocalConnect(time.Second * 5)
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
