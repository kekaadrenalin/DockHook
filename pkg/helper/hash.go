package helper

import (
	"crypto/sha512"
	"encoding/hex"
)

func Sha512sum(s string) string {
	sum512bytes := sha512.Sum512([]byte(s))

	return hex.EncodeToString(sum512bytes[:])
}
