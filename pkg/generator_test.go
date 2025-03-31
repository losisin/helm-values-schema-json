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
	tests := []struct {
		name               string
		config             *Config
		templateSchemaFile string
	}{
		{
			name: "full json schema",
			config: &Config{
				Input: []string{
					"../testdata/full.yaml",
					"../testdata/empty.yaml",
				},
				OutputPath: "../testdata/output.json",
				Draft:      2020,
				Indent:     4,
				SchemaRoot: SchemaRoot{
					ID:                   "https://example.com/schema",
					Ref:                  "schema/product.json",
					Title:                "Helm Values Schema",
					Description:          "Schema for Helm values",
					AdditionalProperties: BoolFlag{set: true, value: true},
				},
			},
			templateSchemaFile: "../testdata/full.schema.json",
		},
		{
			name: "noAdditionalProperties",
			config: &Config{
				Draft:                  2020,
				Indent:                 4,
				NoAdditionalProperties: BoolFlag{set: true, value: true},
				Input: []string{
					"../testdata/noAdditionalProperties.yaml",
				},
				OutputPath: "../testdata/output1.json",
			},
			templateSchemaFile: "../testdata/noAdditionalProperties.schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateJsonSchema(tt.config)
			assert.NoError(t, err)

			generatedBytes, err := os.ReadFile(tt.config.OutputPath)
			assert.NoError(t, err)

			templateBytes, err := os.ReadFile(tt.templateSchemaFile)
			assert.NoError(t, err)

			var generatedSchema, templateSchema map[string]interface{}
			err = json.Unmarshal(generatedBytes, &generatedSchema)
			assert.NoError(t, err)
			err = json.Unmarshal(templateBytes, &templateSchema)
			assert.NoError(t, err)

			assert.Equal(t, templateSchema, generatedSchema, "Generated JSON schema does not match the template")

			if err := os.Remove(tt.config.OutputPath); err != nil && !os.IsNotExist(err) {
				t.Errorf("failed to remove values.schema.json: %v", err)
			}
		})
	}
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
			name: "Missing input flag",
			config: &Config{
				Input:  nil,
				Draft:  2020,
				Indent: 0,
			},
			expectedErr: errors.New("input flag is required"),
		},
		{
			name: "Invalid draft version",
			config: &Config{
				Input: []string{"../testdata/basic.yaml"},
				Draft: 5,
			},
			expectedErr: errors.New("invalid draft version"),
		},
		{
			name: "Negative indentation number",
			config: &Config{
				Input:      []string{"../testdata/basic.yaml"},
				Draft:      2020,
				OutputPath: "testdata/failure/output_readonly_schema.json",
				Indent:     0,
			},
			expectedErr: errors.New("indentation must be a positive number"),
		},
		{
			name: "Odd indentation number",
			config: &Config{
				Input:      []string{"../testdata/basic.yaml"},
				Draft:      2020,
				OutputPath: "testdata/failure/output_readonly_schema.json",
				Indent:     1,
			},
			expectedErr: errors.New("indentation must be an even number"),
		},
		{
			name: "Missing file",
			config: &Config{
				Input:  []string{"missing.yaml"},
				Draft:  2020,
				Indent: 4,
			},
			expectedErr: errors.New("error reading YAML file(s)"),
		},
		{
			name: "Fail Unmarshal",
			config: &Config{
				Input:      []string{"../testdata/fail"},
				OutputPath: "testdata/failure/output_readonly_schema.json",
				Draft:      2020,
				Indent:     4,
			},
			expectedErr: errors.New("error unmarshaling YAML"),
		},
		{
			name: "Read-only filesystem",
			config: &Config{
				Input:      []string{"../testdata/basic.yaml"},
				OutputPath: "testdata/failure/output_readonly_schema.json",
				Draft:      2020,
				Indent:     4,
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
		noAdditionalProperties  bool
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
		{
			name:                    "AdditionalProperties not set, but NoAdditionalProperties set",
			additionalPropertiesSet: false,
			noAdditionalProperties:  true,
			expected:                false,
		},
		{
			name:                    "NoAdditionalProperties set, but AdditionalProperties set to true",
			additionalPropertiesSet: true,
			additionalProperties:    true,
			noAdditionalProperties:  true,
			expected:                true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			additionalPropertiesFlag := &BoolFlag{}
			noAdditionalPropertiesFlag := &BoolFlag{}
			if tt.additionalPropertiesSet {
				if err := additionalPropertiesFlag.Set(fmt.Sprintf("%t", tt.additionalProperties)); err != nil {
					t.Fatalf("Failed to set additionalPropertiesFlag: %v", err)
				}
			}
			if tt.noAdditionalProperties {
				if err := noAdditionalPropertiesFlag.Set(fmt.Sprintf("%t", tt.noAdditionalProperties)); err != nil {
					t.Fatalf("Failed to set noAdditionalPropertiesFlag: %v", err)
				}
			}

			config := &Config{
				Input:                  []string{"../testdata/empty.yaml"},
				OutputPath:             "../testdata/empty.schema.json",
				Draft:                  2020,
				Indent:                 4,
				NoAdditionalProperties: *noAdditionalPropertiesFlag,
				SchemaRoot: SchemaRoot{
					ID:                   "",
					Title:                "",
					Description:          "",
					AdditionalProperties: *additionalPropertiesFlag,
				},
			}

			err := GenerateJsonSchema(config)
			assert.NoError(t, err)

			generatedBytes, err := os.ReadFile(config.OutputPath)
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

			if err := os.Remove(config.OutputPath); err != nil && !os.IsNotExist(err) {
				t.Errorf("failed to remove values.schema.json: %v", err)
			}
		})
	}
}
