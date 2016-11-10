package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit

#include "clipboard_darwin.h"
*/
import "C"
import "unsafe"

func init() {
	C.setup()
}

func clipboardContents() string {
	contents := C.getClipboard()
	if contents != nil {
		out := C.GoString(contents)
		C.free(unsafe.Pointer(contents))
		return out
	}
	return ""
}

func watch(out <-chan []byte) {

}
