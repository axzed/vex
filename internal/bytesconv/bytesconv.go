package bytesconv

import "unsafe"

// StringToBytes use unsafe pointer it will not copy the memory
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
