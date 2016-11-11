package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit

#include "clipboard_darwin.h"
*/
import "C"
import (
	"errors"
	"time"
	"unsafe"
)

func setup() bool {
	return true
}

func getClipboardData() []byte {
	var length C.int
	contents := C.getClipboard(&length)
	if length > 0 && contents != nil {
		out := C.GoBytes(unsafe.Pointer(contents), length)
		C.free(unsafe.Pointer(contents))
		return []byte(out)
	}
	return nil
}

func putClipboardData(data []byte) error {
	var err error
	res := C.setClipboard((*C.char)(unsafe.Pointer(&data[0])), C.int(len(data)))
	if res != nil {
		err = errors.New(C.GoString(res))
		C.free(unsafe.Pointer(res))
	}
	return err
}

func watch(out chan []byte) {
	t := time.NewTicker(time.Millisecond * 500)
	for _ = range t.C {
		data := getClipboardData()
		if data != nil {
			out <- data
		}
	}
}
