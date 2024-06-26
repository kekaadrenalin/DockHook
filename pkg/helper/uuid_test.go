package helper

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/google/uuid"
)

func Test_GenerateUUIDv7_happy(t *testing.T) {
	hashData := "testdata"
	expectedUUID, err := GenerateUUIDv7(hashData)
	assert.NoError(t, err, "expected no error")

	_, err = uuid.Parse(expectedUUID.String())
	assert.NoError(t, err, "expected generated UUID to be valid")
}
func Test_GenerateUUIDv7_happy_with_empty_hash(t *testing.T) {
	hashData := ""
	expectedUUID, err := GenerateUUIDv7(hashData)
	assert.NoError(t, err, "expected no error")

	_, err = uuid.Parse(expectedUUID.String())
	assert.NoError(t, err, "expected generated UUID to be valid")
}
