package testutil

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"

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

// List of structs to allow unexported values in when diffing with [github.com/google/go-cmp/cmp.Diff]
var ExtraDiffAllowUnexported []any

func diff(want, got any) string {
	s := gocmp.Diff(want, got, gocmp.AllowUnexported(ExtraDiffAllowUnexported...))
	if s != "" {
		s = "(-want, +got):\n" + s
	}
	return colorizeDiff(s)
}

var diffCommentRegex = regexp.MustCompile(`\t*\.\.\. // .*`)

func colorizeDiff(diff string) string {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return diff
	}
	var buf bytes.Buffer
	for line := range strings.Lines(diff) {
		if buf.Len() == 0 && line == "(-want, +got):\n" {
			buf.WriteString("(\033[31m-want\033[0m, \033[32m+got\033[0m):\n")
			continue
		}
		switch line[0] {
		case '-':
			buf.WriteString("\033[31m") // red
		case '+':
			buf.WriteString("\033[32m") // green
		case '~':
			buf.WriteString("\033[33m") // yellow
		default:
			if !diffCommentRegex.MatchString(line) {
				buf.WriteString(line)
				continue
			}
			buf.WriteString("\033[90m") // bright black
		}

		withoutLF, hasLF := strings.CutSuffix(line, "\n")
		buf.WriteString(withoutLF)
		buf.WriteString("\033[0m")
		if hasLF {
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

func Equal(t T, want, got any, arg ...any) bool {
	t.Helper()
	if diff := diff(want, got); diff != "" {
		t.Errorf("%s%s", addSpace(fmt.Sprint(arg...)), diff)
		return false
	}
	return true
}

func Equalf(t T, want, got any, format string, arg ...any) bool {
	t.Helper()
	if diff := diff(want, got); diff != "" {
		t.Errorf("%s%s", addSpace(fmt.Sprintf(format, arg...)), diff)
		return false
	}
	return true
}

func MustEqual(t T, want, got any, arg ...any) {
	t.Helper()
	if diff := diff(want, got); diff != "" {
		t.Fatalf("%s%s", addSpace(fmt.Sprint(arg...)), diff)
	}
}

func MustEqualf(t T, want, got any, format string, arg ...any) {
	t.Helper()
	if diff := diff(want, got); diff != "" {
		t.Fatalf("%s%s", addSpace(fmt.Sprintf(format, arg...)), diff)
	}
}

func addSpace(s string) string {
	if s == "" || strings.HasSuffix(s, " ") {
		return s
	}
	return s + " "
}
