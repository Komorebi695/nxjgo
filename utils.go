package nxjgo

import (
	"strings"
	"unicode"
	"unsafe"
)

func SubStringLast(s string, subs string) string {
	index := strings.Index(s, subs)
	if index < 0 {
		return ""
	}
	return s[index+len(subs):]
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
