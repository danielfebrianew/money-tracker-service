package ids

import (
	"crypto/rand"
	"encoding/hex"
)

func New(prefix string) string {
	return prefix + "_" + RandomHex(12)
}

func Token(prefix string, bytes int) string {
	return prefix + "_" + RandomHex(bytes)
}

func RandomHex(bytes int) string {
	if bytes <= 0 {
		bytes = 16
	}
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
