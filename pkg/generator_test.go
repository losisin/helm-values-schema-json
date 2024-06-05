package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
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
		indent:     4,
		SchemaRoot: SchemaRoot{
			ID:          "",
			Title:       "",
			Description: "",
		},
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
			name: "Negative indentation number",
			config: &Config{
				input:      []string{"../testdata/basic.yaml"},
				draft:      2020,
				outputPath: "testdata/failure/output_readonly_schema.json",
				indent:     0,
			},
			expectedErr: errors.New("indentation must be a positive number"),
		},
		{
			name: "Odd indentation number",
			config: &Config{
				input:      []string{"../testdata/basic.yaml"},
				draft:      2020,
				outputPath: "testdata/failure/output_readonly_schema.json",
				indent:     1,
			},
			expectedErr: errors.New("indentation must be an even number"),
		},
		{
			name: "Missing file",
			config: &Config{
				input:  []string{"missing.yaml"},
				draft:  2020,
				indent: 4,
			},
			expectedErr: errors.New("error reading YAML file(s)"),
		},
		{
			name: "Fail Unmarshal",
			config: &Config{
				input:      []string{"../testdata/fail"},
				outputPath: "testdata/failure/output_readonly_schema.json",
				draft:      2020,
				indent:     4,
			},
			expectedErr: errors.New("error unmarshaling YAML"),
		},
		{
			name: "Read-only filesystem",
			config: &Config{
				input:      []string{"../testdata/basic.yaml"},
				outputPath: "testdata/failure/output_readonly_schema.json",
				draft:      2020,
				indent:     4,
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
func TestGenerateJsonSchema_AdditionalProperties(t *testing.T) {
	tests := []struct {
		name                    string
		additionalPropertiesSet bool
		additionalProperties    bool
		expected                interface{}
	}{
		{
			name:                    "AdditionalProperties set to true",
			additionalPropertiesSet: true,
			additionalProperties:    true,
			expected:                true,
		},
		{
			name:                    "AdditionalProperties set to false",
			additionalPropertiesSet: true,
			additionalProperties:    false,
			expected:                false,
		},
		{
			name:                    "AdditionalProperties not set",
			additionalPropertiesSet: false,
			expected:                nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			additionalPropertiesFlag := &BoolFlag{}
			if tt.additionalPropertiesSet {
				if err := additionalPropertiesFlag.Set(fmt.Sprintf("%t", tt.additionalProperties)); err != nil {
					t.Fatalf("Failed to set additionalPropertiesFlag: %v", err)
				}
			}

			config := &Config{
				input:      []string{"../testdata/empty.yaml"},
				outputPath: "../testdata/empty.schema.json",
				draft:      2020,
				indent:     4,
				SchemaRoot: SchemaRoot{
					ID:                   "",
					Title:                "",
					Description:          "",
					AdditionalProperties: *additionalPropertiesFlag,
				},
			}

			err := GenerateJsonSchema(config)
			assert.NoError(t, err)

			generatedBytes, err := os.ReadFile(config.outputPath)
			assert.NoError(t, err)

			var generatedSchema map[string]interface{}
			err = json.Unmarshal(generatedBytes, &generatedSchema)
			assert.NoError(t, err)

			if tt.expected == nil {
				_, exists := generatedSchema["additionalProperties"]
				assert.False(t, exists, "additionalProperties should not be present in the generated schema")
			} else {
				assert.Equal(t, tt.expected, generatedSchema["additionalProperties"], "additionalProperties value mismatch")
			}

			os.Remove(config.outputPath)
		})
	}
}
