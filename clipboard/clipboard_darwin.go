package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit

#include "clipboard_darwin.h"
*/
import "C"
import (
	"time"
	"unsafe"
)

func setup() bool {
	return true
}

func clipboardContents() []byte {
	var length C.int
	contents := C.getClipboard(&length)
	if length > 0 && contents != nil {
		out := C.GoBytes(unsafe.Pointer(contents), length)
		C.free(unsafe.Pointer(contents))
		return []byte(out)
	}
	return nil
}

func watch(out chan []byte) {
	t := time.NewTicker(time.Millisecond * 500)
	for _ = range t.C {
		data := clipboardContents()
		if data != nil {
			out <- data
		}
	}
}
