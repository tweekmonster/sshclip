package sshclip

import "net"

var manualStop = CreateEvent("ManualLoopStop")
var loopStop = CreateListener(Interrupt, Terminate, manualStop)

func ListenLoopStop() {
	DispatchEvent(manualStop)
}

// ListenLoop waits for an stop event while accepting connections from a
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
		case e := <-loopStop:
			Dlog("Got event:", e)
			return listener.Close()
		case c := <-conn:
			Dlog("Connection from:", c.RemoteAddr())
			go handler(c)
		}
	}
}
