package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setup         func()
		cleanup       func()
		expectedError string
		expectedOut   string
	}{
		{
			name:          "HelpFlag",
			args:          []string{"schema", "-h"},
			expectedOut:   "Usage of schema",
			expectedError: "",
		},
		{
			name:          "InvalidFlags",
			args:          []string{"schema", "-fail"},
			expectedOut:   "",
			expectedError: "flag provided but not defined",
		},
		{
			name:          "SuccessfulRun",
			args:          []string{"schema", "-input", "testdata/basic.yaml"},
			expectedOut:   "JSON schema successfully generated",
			expectedError: "",
		},
		{
			name:          "GenerateError",
			args:          []string{"schema", "-input", "fail.yaml", "-draft", "2020"},
			expectedOut:   "error reading YAML file(s)",
			expectedError: "",
		},
		{
			name: "ErrorLoadingConfigFile",
			args: []string{"schema", "-input", "testdata/basic.yaml"},
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
			expectedOut:   "",
			expectedError: "Error loading config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalArgs := os.Args
			originalStdout := os.Stdout

			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			r, w, _ := os.Pipe()
			os.Stdout = w

			os.Args = tt.args

			errCh := make(chan error, 1)

			go func() {
				main()
				if err := w.Close(); err != nil {
					errCh <- err
				}
				close(errCh)
			}()

			if err := <-errCh; err != nil {
				t.Errorf("Error closing pipe: %v", err)
			}

			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			if err != nil {
				t.Errorf("Error reading stdout: %v", err)
			}

			os.Args = originalArgs
			os.Stdout = originalStdout

			out := buf.String()

			assert.Contains(t, out, tt.expectedError)
			if tt.expectedOut != "" {
				assert.Contains(t, out, tt.expectedOut)
			}
			if err := os.Remove("values.schema.json"); err != nil && !os.IsNotExist(err) {
				t.Errorf("failed to remove values.schema.json: %v", err)
			}
		})
	}
}
