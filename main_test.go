package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalArgs := os.Args
			originalStdout := os.Stdout

			defer os.Remove("values.schema.json")

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

			assert.Contains(t, out, tt.expectedOut)
			if tt.expectedError != "" {
				assert.Contains(t, out, tt.expectedError)
			}
		})
	}
}
