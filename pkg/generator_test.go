package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		{
			name: "k8s ref alias",
			config: &Config{
				Draft:            2020,
				Indent:           4,
				K8sSchemaURL:     "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				K8sSchemaVersion: "v1.33.1",
				Input: []string{
					"../testdata/k8sRef.yaml",
				},
				OutputPath: "../testdata/k8sRef_output.json",
			},
			templateSchemaFile: "../testdata/k8sRef.schema.json",
		},

		{
			name: "ref draft 7",
			config: &Config{
				Draft:  7,
				Indent: 4,
				Input: []string{
					"../testdata/ref.yaml",
				},
				OutputPath: "../testdata/ref-draft7_output.json",
			},
			templateSchemaFile: "../testdata/ref-draft7.schema.json",
		},
		{
			name: "ref draft 2020",
			config: &Config{
				Draft:  2020,
				Indent: 4,
				Input: []string{
					"../testdata/ref.yaml",
				},
				OutputPath: "../testdata/ref-draft2020_output.json",
			},
			templateSchemaFile: "../testdata/ref-draft2020.schema.json",
		},

		{
			name: "bundle/simple",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: "../",
				Input: []string{
					"../testdata/bundle/simple.yaml",
				},
				OutputPath: "../testdata/bundle/simple_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple.schema.json",
		},
		{
			name: "bundle/simple-disabled",
			config: &Config{
				Draft:  2020,
				Indent: 4,
				Bundle: BoolFlag{set: true, value: false},
				Input: []string{
					"../testdata/bundle/simple.yaml",
				},
				OutputPath: "../testdata/bundle/simple-disabled_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple-disabled.schema.json",
		},
		{
			name: "bundle/without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          BoolFlag{set: true, value: true},
				BundleWithoutID: BoolFlag{set: true, value: true},
				BundleRoot:      "../",
				Input: []string{
					"../testdata/bundle/simple.yaml",
				},
				OutputPath: "../testdata/bundle_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple-without-id.schema.json",
		},
		{
			name: "bundle/nested",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: "..",
				Input: []string{
					"../testdata/bundle/nested.yaml",
				},
				OutputPath: "../testdata/bundle/nested_output.json",
			},
			templateSchemaFile: "../testdata/bundle/nested.schema.json",
		},
		{
			name: "bundle/nested-without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          BoolFlag{set: true, value: true},
				BundleWithoutID: BoolFlag{set: true, value: true},
				BundleRoot:      "..",
				Input: []string{
					"../testdata/bundle/nested.yaml",
				},
				OutputPath: "../testdata/bundle/nested-without-id_output.json",
			},
			templateSchemaFile: "../testdata/bundle/nested-without-id.schema.json",
		},
		{
			name: "bundle/fragment",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: "..",
				Input: []string{
					"../testdata/bundle/fragment.yaml",
				},
				OutputPath: "../testdata/bundle/fragment_output.json",
			},
			templateSchemaFile: "../testdata/bundle/fragment.schema.json",
		},
		{
			name: "bundle/fragment-without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          BoolFlag{set: true, value: true},
				BundleWithoutID: BoolFlag{set: true, value: true},
				BundleRoot:      "..",
				Input: []string{
					"../testdata/bundle/fragment.yaml",
				},
				OutputPath: "../testdata/bundle/fragment-without-id_output.json",
			},
			templateSchemaFile: "../testdata/bundle/fragment-without-id.schema.json",
		},
		{
			name: "bundle/namecollision",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: "..",
				Input: []string{
					"../testdata/bundle/namecollision.yaml",
				},
				OutputPath: "../testdata/bundle/namecollision_output.json",
			},
			templateSchemaFile: "../testdata/bundle/namecollision.schema.json",
		},
		{
			name: "bundle/yaml",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: "..",
				Input: []string{
					"../testdata/bundle/yaml.yaml",
				},
				OutputPath: "../testdata/bundle/yaml_output.json",
			},
			templateSchemaFile: "../testdata/bundle/yaml.schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateJsonSchema(tt.config)
			require.NoError(t, err)

			generatedBytes, err := os.ReadFile(tt.config.OutputPath)
			require.NoError(t, err)

			templateBytes, err := os.ReadFile(tt.templateSchemaFile)
			require.NoError(t, err)

			t.Logf("Generated output:\n%s\n", generatedBytes)

			assert.JSONEqf(t, string(templateBytes), string(generatedBytes), "Generated JSON schema %q does not match the template", tt.templateSchemaFile)

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
		{
			name: "bundle invalid root path",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: "\000", // null byte is invalid in both linux & windows
				Input: []string{
					"../testdata/bundle/simple.yaml",
				},
				OutputPath: "../testdata/bundle_output.json",
			},
			expectedErr: errors.New("open bundle root: open \x00: invalid argument"),
		},
		{
			name: "bundle wrong root path",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: ".",
				Input: []string{
					"../testdata/bundle/simple.yaml",
				},
				OutputPath: "../testdata/bundle_output.json",
			},
			expectedErr: errors.New("path escapes from parent"),
		},
		{
			name: "bundle fail to get relative path",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     BoolFlag{set: true, value: true},
				BundleRoot: filepath.Clean("/"),
				Input: []string{
					"../testdata/bundle/simple.yaml",
				},
				OutputPath: "../testdata/bundle_output.json",
			},
			expectedErr: errors.New("get relative path from bundle root to file"),
		},
		{
			name: "invalid k8s ref alias",
			config: &Config{
				Draft:            2020,
				Indent:           4,
				K8sSchemaURL:     "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				K8sSchemaVersion: "",
				Input: []string{
					"../testdata/k8sRef.yaml",
				},
				OutputPath: "../testdata/k8sRef_output.json",
			},
			expectedErr: errors.New("/properties/memory: must set k8sSchemaVersion config when using \"$ref: $k8s/...\""),
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
