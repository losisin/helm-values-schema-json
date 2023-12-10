package pkg

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateJsonSchema(t *testing.T) {
	config := &Config{
		input: []string{
			"../testdata/full.yaml",
			"../testdata/empty.yaml",
		},
		outputPath: "../testdata/output.json",
		draft:      2020,
	}

	err := GenerateJsonSchema(config)
	assert.NoError(t, err)

	generatedBytes, err := os.ReadFile(config.outputPath)
	assert.NoError(t, err)

	templateBytes, err := os.ReadFile("../testdata/full.schema.json")
	assert.NoError(t, err)

	var generatedSchema, templateSchema map[string]interface{}
	err = json.Unmarshal(generatedBytes, &generatedSchema)
	assert.NoError(t, err)
	err = json.Unmarshal(templateBytes, &templateSchema)
	assert.NoError(t, err)

	assert.Equal(t, templateSchema, generatedSchema, "Generated JSON schema does not match the template")

	os.Remove(config.outputPath)
}

func TestGenerateJsonSchema_Errors(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		setupFunc   func() error
		cleanupFunc func() error
		expectedErr error
	}{
		{
			name: "Invalid draft version",
			config: &Config{
				input: []string{"../testdata/basic.yaml"},
				draft: 5,
			},
			expectedErr: errors.New("invalid draft version"),
		},
		{
			name: "Missing file",
			config: &Config{
				input: []string{"missing.yaml"},
				draft: 2020,
			},
			expectedErr: errors.New("error reading YAML file(s)"),
		},
		{
			name: "Fail Unmarshal",
			config: &Config{
				input:      []string{"../testdata/fail"},
				outputPath: "testdata/failure/output_readonly_schema.json",
				draft:      2020,
			},
			expectedErr: errors.New("error unmarshaling YAML"),
		},
		{
			name: "Read-only filesystem",
			config: &Config{
				input:      []string{"../testdata/basic.yaml"},
				outputPath: "testdata/failure/output_readonly_schema.json",
				draft:      2020,
			},
			expectedErr: errors.New("error writing schema to file"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				err := tt.setupFunc()
				assert.NoError(t, err)
			}

			err := GenerateJsonSchema(tt.config)
			assert.Error(t, err)
			if err != nil {
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			}

			if tt.cleanupFunc != nil {
				err := tt.cleanupFunc()
				assert.NoError(t, err)
			}
		})
	}
}
