package testutil

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeGetwdFail(t *testing.T) {
	_, err := os.Getwd()
	require.NoError(t, err)

	MakeGetwdFail(t)

	_, err = os.Getwd()
	require.ErrorContains(t, err, "getwd: no such file or directory")
}

func TestCreateTempFile(t *testing.T) {
	var file *os.File
	t.Run("sub-test", func(t *testing.T) {
		file = CreateTempFile(t, "test-*.txt")
		require.NotNil(t, file)
		require.FileExists(t, file.Name())
	})
	require.NoFileExists(t, file.Name())
}

func TestWriteTempFile(t *testing.T) {
	var file *os.File
	t.Run("sub-test", func(t *testing.T) {
		file = WriteTempFile(t, "test-*.txt", []byte("lorem ipsum"))
		require.FileExists(t, file.Name())
		content, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		require.Equal(t, "lorem ipsum", string(content))
	})
	require.NoFileExists(t, file.Name())
}

func TestResetFile(t *testing.T) {
	// Create a long file
	file := WriteTempFile(t, "test-*.txt", bytes.Repeat([]byte("lorem ipsum "), 100))
	require.FileExists(t, file.Name())

	ResetFile(t, file, []byte("much shorter content"))

	content, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	require.Equal(t, "much shorter content", string(content))
}
