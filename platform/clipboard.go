package platform

import (
	"bytes"
	"errors"

	"golang.org/x/crypto/blake2b"

	"github.com/tweekmonster/sshclip"
)

var clipboardEnabled = false
var errEmpty = errors.New("empty")
var ErrClipboardUnavailable = errors.New("clipboard unavailable")
var stop = sshclip.CreateUniqueEvent("ClipboardMonitorStop")

func init() {
	clipboardEnabled = setupClipboard()
}

func ClipboardEnabled() bool {
	return clipboardEnabled
}

func ClipboardMonitorStop() {
	sshclip.DispatchEvent(stop)
}

func ClipboardMonitor(storage sshclip.Register, reg uint8) error {
	if !clipboardEnabled {
		return ErrClipboardUnavailable
	}

	events := sshclip.CreateListener(sshclip.Terminate, stop)
	watchChan := make(chan []byte)
	go watchClipboard(watchChan)

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

func ClipboardGet() []byte {
	return getClipboardData()
}

func ClipboardPut(data []byte) error {
	return putClipboardData(data)
}
