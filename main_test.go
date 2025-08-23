package main

import (
	"bytes"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedError    string
		expectedOut      string
		expectedExitCode int
	}{
		{
			name:          "HelpFlag",
			args:          []string{"schema", "-h"},
			expectedOut:   "Usage:\n  helm schema",
			expectedError: "",
		},
		{
			name:          "CompleteFlag",
			args:          []string{"schema", "__complete", "--d"},
			expectedOut:   "--draft\tDraft version",
			expectedError: "",
		},
		{
			name:             "InvalidFlags",
			args:             []string{"schema", "--fail"},
			expectedOut:      "",
			expectedError:    "unknown flag: --fail",
			expectedExitCode: 1,
		},
		{
			name: "SuccessfulRun",
			args: []string{
				"schema",
				"--config", "testdata/main/config.yaml",
				"--output", "testdata/main/values.schema.json",
				"--values", "testdata/basic.yaml",
			},
			expectedOut:   "JSON schema successfully generated",
			expectedError: "",
		},
		{
			name: "GenerateError",
			args: []string{
				"schema",
				"--config", "testdata/main/config.yaml",
				"--values", "fail.yaml",
				"--draft", "2020",
			},
			expectedOut:      "",
			expectedError:    "error reading YAML file(s)",
			expectedExitCode: 1,
		},
		{
			name: "ErrorLoadingConfigFile",
			args: []string{
				"schema",
				"--config", "testdata/main/config-invalid.yaml",
				"--values", "testdata/basic.yaml",
			},
			expectedOut:      "",
			expectedError:    "load config file testdata/main/config-invalid.yaml:",
			expectedExitCode: 1,
		},
		{
			name: "GenerateConfigSchema",
			args: []string{
				"schema",
			},
			expectedOut: "JSON schema successfully generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func(args []string, stdout, stderr *os.File) {
				// Reset to original args/stdout/stderr at end of test
				os.Args = args
				os.Stdout = stdout
				os.Stderr = stderr
			}(os.Args, os.Stdout, os.Stderr)

			var exitCode int
			defer func(oldFunc func(code int)) { osExit = oldFunc }(osExit)
			osExit = func(code int) { exitCode = code }

			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			os.Args = tt.args

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				main()
				require.NoError(t, w.Close())
			}()
			wg.Wait()

			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			require.NoError(t, err, "Error reading stdout")

			out := buf.String()

			assert.Contains(t, out, tt.expectedError)
			if tt.expectedOut != "" {
				assert.Contains(t, out, tt.expectedOut)
			}
			assert.Equal(t, tt.expectedExitCode, exitCode, "Expected exit code")
		})
	}
}

func TestGetVersion(t *testing.T) {
	Version = "v1.2.3"
	assert.Equal(t, "v1.2.3", getVersion())

	Version = "1.2.3"
	assert.Equal(t, "v1.2.3", getVersion())

	Version = ""
	assert.Equal(t, "(devel)", getVersion())
}

func TestGetVersionFromBuildInfo(t *testing.T) {
	Version = ""
	tests := []struct {
		name    string
		version string
		info    *debug.BuildInfo
		want    string
	}{
		{
			name: "nil",
			info: nil,
			want: "(devel)",
		},
		{
			name: "main version",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v4.5.6"},
			},
			want: "v4.5.6",
		},
		{
			name: "vcs revision",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "some-sha-value"},
				},
			},
			want: "some-sha-value",
		},
		{
			name: "vcs dirty revision",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "some-sha-value"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			want: "some-sha-value-dirty",
		},
		{
			name: "no vcs",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: ""},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			want: "(devel)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getVersionFromBuildInfo(tt.info, tt.info != nil)
			if got != tt.want {
				t.Errorf("wrong result\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}
