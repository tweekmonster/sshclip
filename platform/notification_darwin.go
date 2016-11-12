package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit

#include "notification_darwin.h"
*/
import "C"
import "unsafe"

func setupNotifications() bool {
	C.setup()
	return true
}

func postNotification(text string) bool {
	data := []byte(text)
	res := C.postNotification((*C.char)(unsafe.Pointer(&data[0])))
	return res == 0
}
