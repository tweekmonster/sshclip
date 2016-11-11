package clipboard

import (
	"bytes"
	"errors"

	"golang.org/x/crypto/blake2b"

	"github.com/tweekmonster/sshclip"
)

var enabled = false
var errEmpty = errors.New("empty")
var ErrUnavailable = errors.New("clipboard unavailable")
var stop = sshclip.CreateUniqueEvent("ClipboardMonitorStop")

func init() {
	enabled = setup()
}

func Enabled() bool {
	return enabled
}

func MonitorStop() {
	sshclip.DispatchEvent(stop)
}

func Monitor(storage sshclip.Register, reg uint8) error {
	if !enabled {
		return ErrUnavailable
	}

	events := sshclip.CreateListener(sshclip.Terminate, stop)
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
		case <-events:
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
