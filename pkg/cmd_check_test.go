package pkg

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.schema.json")
	invoke := func(output string, extra ...string) error {
		cmd := NewCmd()
		cmd.SetArgs(append([]string{"--values", "../testdata/lint/values.yaml", "--output", output}, extra...))
		return cmd.Execute()
	}

	require.ErrorContains(t, invoke(path, "--check"), "missing")
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorContains(t, invoke("-", "--check"), "requires a file")

	require.NoError(t, invoke(path))
	stamp := time.Unix(1234567890, 0)
	require.NoError(t, os.Chtimes(path, stamp, stamp))
	require.NoError(t, invoke(path, "--check"))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, stamp.Unix(), info.ModTime().Unix())

	stale := []byte("{}\n")
	require.NoError(t, os.WriteFile(path, stale, 0600))
	require.ErrorContains(t, invoke(path, "--check"), "out of date")
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, stale, actual)
}
