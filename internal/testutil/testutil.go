package testutil

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func MakeGetwdFail(t *testing.T) {
	// Setting up to make [os.Getwd] to fail, which on Linux can be done
	// by deleting the directory you're currently in.
	tempDir, err := os.MkdirTemp("", "schema-cwd-*")
	require.NoError(t, err)
	t.Chdir(tempDir)
	require.NoError(t, os.Remove(tempDir))
}

// ResetFile will truncate the file and write the new content to it.
func ResetFile(t *testing.T, file *os.File, content []byte) {
	_, err := file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(0))
	_, err = file.Write(content)
	require.NoError(t, err)
}

// CreateTempFile creates a temporary file and removes it at the end of the test.
func CreateTempFile(t *testing.T, pattern string) *os.File {
	tmpFile, err := os.CreateTemp("", pattern)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, tmpFile.Close())
		assert.NoError(t, os.Remove(tmpFile.Name()))
	})
	return tmpFile
}

// WriteTempFile creates a temporary file with a given content and removes it at the end of the test.
func WriteTempFile(t *testing.T, pattern string, content []byte) *os.File {
	tmpFile := CreateTempFile(t, pattern)
	_, err := tmpFile.Write(content)
	require.NoError(t, err)
	return tmpFile
}
