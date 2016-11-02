package sshclip

import (
	"net"
	"os"
	"os/signal"
	"syscall"
)

var loopStop = make(chan os.Signal)

type manualStopSignal int

func (manualStopSignal) Signal()        {}
func (manualStopSignal) String() string { return "Manual Stop" }

func init() {
	signal.Notify(loopStop, os.Interrupt, syscall.SIGTERM)
}

func ListenLoopStop() {
	loopStop <- manualStopSignal(0)
}

// ListenLoop waits for an interrupt signal while accepting connections from a
// listener in a goroutine.
func ListenLoop(listener net.Listener, handler func(net.Conn)) error {
	conn := make(chan net.Conn)
	errs := make(chan error)

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				errs <- err
				break
			}
			conn <- c
		}
	}()

	for {
		select {
		case err := <-errs:
			return err
		case sig := <-loopStop:
			Dlog("Got signal:", sig)
			return listener.Close()
		case c := <-conn:
			Dlog("Connection from:", c.RemoteAddr())
			go handler(c)
		}
	}
}
