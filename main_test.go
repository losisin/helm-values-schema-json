package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/losisin/helm-values-schema-json/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		setup            func()
		cleanup          func()
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
			name:          "SuccessfulRun",
			args:          []string{"schema", "--values", "testdata/basic.yaml"},
			expectedOut:   "JSON schema successfully generated",
			expectedError: "",
		},
		{
			name:             "GenerateError",
			args:             []string{"schema", "--values", "fail.yaml", "--draft", "2020"},
			expectedOut:      "",
			expectedError:    "error reading YAML file(s)",
			expectedExitCode: 1,
		},
		{
			name: "ErrorLoadingConfigFile",
			args: []string{"schema", "--values", "testdata/basic.yaml"},
			setup: func() {
				if _, err := os.Stat(".schema.yaml"); err == nil {
					if err := os.Rename(".schema.yaml", ".schema.yaml.bak"); err != nil {
						log.Fatalf("Error renaming file: %v", err)
					}
				}

				file, _ := os.Create(".schema.yaml")
				defer func() {
					if err := file.Close(); err != nil {
						log.Fatalf("Error closing file: %v", err)
					}
				}()
				if _, err := file.WriteString("draft: invalid\n"); err != nil {
					log.Fatalf("Error writing to file: %v", err)
				}
			},
			cleanup: func() {
				if _, err := os.Stat(".schema.yaml.bak"); err == nil {
					if err := os.Remove(".schema.yaml"); err != nil && !os.IsNotExist(err) {
						log.Fatalf("Error removing file: %v", err)
					}
					if err := os.Rename(".schema.yaml.bak", ".schema.yaml"); err != nil {
						log.Fatalf("Error renaming file: %v", err)
					}
				} else {
					if err := os.Remove(".schema.yaml"); err != nil && !os.IsNotExist(err) {
						log.Fatalf("Error removing file: %v", err)
					}
				}
			},
			expectedOut:      "",
			expectedError:    "load config file .schema.yaml:",
			expectedExitCode: 1,
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

			t.Cleanup(func() {
				if err := os.Remove("values.schema.json"); err != nil && !os.IsNotExist(err) {
					t.Fatalf("failed to remove values.schema.json: %v", err)
				}
			})

			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

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
			testutil.Equal(t, tt.expectedExitCode, exitCode, "Expected exit code")
		})
	}
}

func TestGetVersion(t *testing.T) {
	Version = "v1.2.3"
	testutil.Equal(t, "v1.2.3", getVersion())

	Version = "1.2.3"
	testutil.Equal(t, "v1.2.3", getVersion())

	Version = ""
	testutil.Equal(t, "(devel)", getVersion())
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
