package sshclip

import (
	"errors"
	"net"
)

var manualStop = CreateEvent("ManualLoopStop")

func ListenLoopStop() {
	DispatchEvent(manualStop)
}

// ListenLoop waits for an stop event while accepting connections from a
// listener in a goroutine.
func ListenLoop(listener net.Listener, handler func(net.Conn)) error {
	loopStop := CreateListener(Interrupt, Terminate, manualStop)
	defer RemoveListener(loopStop)

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
		case e := <-loopStop:
			Dlog("Got event:", e)
			return listener.Close()
		case c, ok := <-conn:
			if !ok {
				return errors.New("Connection channel dead")
			}
			Dlog("Connection from:", c.RemoteAddr())
			go handler(c)
		}
	}
}
