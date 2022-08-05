//go:build !appengine
// +build !appengine

package xcache

import (
	"unsafe"
)

// bytesToString converts unsafely a slice of bytes to a string.
func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
