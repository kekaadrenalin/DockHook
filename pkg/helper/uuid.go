package helper

import (
	"crypto/sha256"

	"github.com/google/uuid"
)

func GenerateUUIDv7(hashData string) (uuid.UUID, error) {
	hash := sha256.Sum256([]byte(hashData))

	var uuidBytes [16]byte
	copy(uuidBytes[:], hash[:16])

	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x70
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80

	return uuid.UUID(uuidBytes), nil
}
