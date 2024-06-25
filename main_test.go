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
				defer file.Close()
				if _, err := file.WriteString("draft: invalid\n"); err != nil {
					log.Fatalf("Error writing to file: %v", err)
				}
			},
			cleanup: func() {
				if _, err := os.Stat(".schema.yaml.bak"); err == nil {
					os.Remove(".schema.yaml")
					if err := os.Rename(".schema.yaml.bak", ".schema.yaml"); err != nil {
						log.Fatalf("Error renaming file: %v", err)
					}
				} else {
					os.Remove(".schema.yaml")
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

			go func() {
				main()
				w.Close()
			}()

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
		})
	}
}
