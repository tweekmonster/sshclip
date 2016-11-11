package clipboard

import (
	"bytes"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/blake2b"

	"github.com/tweekmonster/sshclip"
)

var enabled = false
var errEmpty = errors.New("empty")
var ErrUnavailable = errors.New("clipboard unavailable")

func init() {
	enabled = setup()
}

func Enabled() bool {
	return enabled
}

func Monitor(storage sshclip.Register, reg uint8) error {
	if !enabled {
		return ErrUnavailable
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	watchChan := make(chan []byte)
	go watch(watchChan)

	var prevHash [32]byte

	for {
		select {
		case data := <-watchChan:
			curHash := blake2b.Sum256(data)

			if !bytes.Equal(curHash[:], prevHash[:]) {
				prevHash = curHash

				var attrs uint8
				if data[len(data)-1] == '\n' {
					attrs = sshclip.VimLineAttr
				} else {
					attrs = sshclip.VimCharAttr
				}

				storage.Put(reg, attrs, data)
			}
		case <-signals:
			return nil
		}
	}
}

func Get() []byte {
	return getClipboardData()
}

func Put(data []byte) error {
	return putClipboardData(data)
}
