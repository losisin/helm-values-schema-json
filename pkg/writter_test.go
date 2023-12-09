package pkg

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteMap(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		outputPath  string
		expectedErr error
	}{
		{
			name:        "non-nil data",
			data:        map[string]interface{}{"key": "value"},
			outputPath:  "test_output.json",
			expectedErr: nil,
		},
		{
			name:        "nil data",
			data:        nil,
			outputPath:  "test_output.json",
			expectedErr: errors.New("data is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file
			tmpfile, err := os.CreateTemp("", tt.outputPath)
			if err != nil {
				t.Fatal(err)
			}
			tt.outputPath = tmpfile.Name()

			// Clean up after the test
			defer os.Remove(tt.outputPath)

			// Call printMap and check for errors
			err = writeMap(tt.data, tt.outputPath)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				// Confirm the file content is as expected
				contents, err := os.ReadFile(tt.outputPath)
				assert.NoError(t, err)
				var jsonData map[string]interface{}
				err = json.Unmarshal(contents, &jsonData)
				assert.NoError(t, err)
				assert.Equal(t, tt.data, jsonData)
			}

			// Cleanup
			tmpfile.Close()
		})
	}
}
