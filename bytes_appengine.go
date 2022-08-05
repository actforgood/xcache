//go:build appengine
// +build appengine

package xcache

// bytesToString converts a byte slice to string.
func bytesToString(b []byte) string {
	return string(b)
}
