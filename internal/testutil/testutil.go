package testutil

import (
	"cmp"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Variable is only used to fake which GOOS is set
var goosOverrideForTests string

func MakeGetwdFail(t *testing.T) {
	t.Helper()
	switch cmp.Or(goosOverrideForTests, runtime.GOOS) {
	case "darwin", "windows":
		t.Skipf("Skipping because don't know how to make os.Getwd fail on GOOS=%s", runtime.GOOS)
	}

	// Setting up to make [os.Getwd] to fail, which on Linux can be done
	// by deleting the directory you're currently in.
	tempDir, err := os.MkdirTemp("", "schema-cwd-*")
	require.NoError(t, err)
	t.Chdir(tempDir)
	require.NoError(t, os.Remove(tempDir))
}

// ResetFile will truncate the file and write the new content to it.
func ResetFile(t *testing.T, file *os.File, content []byte) {
	t.Helper()
	_, err := file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(0))
	_, err = file.Write(content)
	require.NoError(t, err)
}

// CreateTempFile creates a temporary file and removes it at the end of the test.
func CreateTempFile(t *testing.T, pattern string) *os.File {
	t.Helper()
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
	t.Helper()
	tmpFile := CreateTempFile(t, pattern)
	_, err := tmpFile.Write(content)
	require.NoError(t, err)
	return tmpFile
}

// CreateTempDir creates a temporary directory removes it at the end of the test.
func CreateTempDir(t *testing.T, pattern string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", pattern)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll(dir)) })
	return dir
}

func ResetEnvAfterTest(t *testing.T) {
	t.Helper()
	envs := os.Environ()
	t.Setenv("_foobar", "") // calling this to indirectly call [testing.T.checkParallel]
	t.Cleanup(func() {
		os.Clearenv()
		for _, env := range envs {
			k, v, _ := strings.Cut(env, "=")
			assert.NoError(t, os.Setenv(k, v))
		}
	})
}

// PerGOOS contains various strings used depending on which OS is running the test.
type PerGOOS struct {
	Default string

	Windows string
	Darwin  string
}

func (err PerGOOS) String() string {
	switch cmp.Or(goosOverrideForTests, runtime.GOOS) {
	case "windows":
		return cmp.Or(err.Windows, err.Default)
	case "darwin":
		return cmp.Or(err.Darwin, err.Default)
	default:
		return err.Default
	}
}
