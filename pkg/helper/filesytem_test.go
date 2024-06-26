package helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CreateDir_happy(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "testdir", "testfile.txt")

	file, err := CreateDir(testPath)
	defer func() {
		if file != nil {
			file.Close()
			os.Remove(testPath)
		}
	}()

	assert.NoError(t, err, "expected no error")
	assert.FileExists(t, testPath, "expected file to be created")
}

func Test_CreateDir_error(t *testing.T) {
	invalidPath := "/root/invalidpath/testfile.txt"
	file, err := CreateDir(invalidPath)

	assert.Error(t, err, "expected error when creating directory")
	assert.Nil(t, file, "expected file to be nil")
}

func Test_CreateDir_error_when_exists(t *testing.T) {
	tmpDir := t.TempDir()

	existingDir := filepath.Join(tmpDir, "existingdir")
	err := os.Mkdir(existingDir, 0700)
	assert.NoError(t, err, "expected no error when creating existing directory")

	file, err := CreateDir(existingDir)
	assert.Error(t, err, "expected error when path exists as directory")
	assert.Nil(t, file, "expected file to be nil")
}
