package testutil

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeGetwdFail(t *testing.T) {
	_, err := os.Getwd()
	require.NoError(t, err)

	MakeGetwdFail(t)

	_, err = os.Getwd()
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestMakeGetwdFail_skipped(t *testing.T) {
	goosOverrideForTests = "darwin"
	defer func() { goosOverrideForTests = "" }()

	var skipped bool
	t.Run("sub-test", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		MakeGetwdFail(t)
	})

	require.True(t, skipped)
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

func TestCreateTempDir(t *testing.T) {
	var dir string
	t.Run("sub-test", func(t *testing.T) {
		dir = CreateTempDir(t, "test-*.txt")
		require.NotNil(t, dir)
		require.DirExists(t, dir)
	})
	require.NoDirExists(t, dir)
}

func TestResetEnvAfterTest(t *testing.T) {
	t.Setenv("foo", "bar")
	t.Run("sub-test", func(t *testing.T) {
		require.Equal(t, "bar", os.Getenv("foo"))
		ResetEnvAfterTest(t)
		require.NoError(t, os.Setenv("foo", "inner"))
		require.Equal(t, "inner", os.Getenv("foo"))
	})
	require.Equal(t, "bar", os.Getenv("foo"))
}

func TestPerGOOS(t *testing.T) {
	tests := []struct {
		name string
		goos string
		per  PerGOOS
		want string
	}{
		{name: "empty", goos: "", per: PerGOOS{}, want: ""},

		{name: "linux uses default", goos: "linux", per: PerGOOS{Default: "default value"}, want: "default value"},
		{name: "linux ignores windows", goos: "linux", per: PerGOOS{Default: "default value", Windows: "windows value"}, want: "default value"},
		{name: "linux ignores darwin", goos: "linux", per: PerGOOS{Default: "default value", Darwin: "darwin value"}, want: "default value"},

		{name: "darwin uses default", goos: "darwin", per: PerGOOS{Default: "default value"}, want: "default value"},
		{name: "darwin ignores windows", goos: "darwin", per: PerGOOS{Default: "default value", Windows: "windows value"}, want: "default value"},
		{name: "darwin uses darwin", goos: "darwin", per: PerGOOS{Default: "default value", Darwin: "darwin value"}, want: "darwin value"},

		{name: "windows uses default", goos: "windows", per: PerGOOS{Default: "default value"}, want: "default value"},
		{name: "windows uses windows", goos: "windows", per: PerGOOS{Default: "default value", Windows: "windows value"}, want: "windows value"},
		{name: "windows ignores darwin", goos: "windows", per: PerGOOS{Default: "default value", Darwin: "darwin value"}, want: "default value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goosOverrideForTests = tt.goos
			defer func() { goosOverrideForTests = "" }()
			assert.Equal(t, tt.want, tt.per.String())
		})
	}
}
