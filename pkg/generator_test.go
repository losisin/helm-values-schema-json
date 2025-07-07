package pkg

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/losisin/helm-values-schema-json/v2/internal/testutil"
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
				Values: []string{
					"../testdata/full.yaml",
					"../testdata/empty.yaml",
				},
				Output: "../testdata/output.json",
				Draft:  2020,
				Indent: 4,
				SchemaRoot: SchemaRoot{
					ID:                   "https://example.com/schema",
					Ref:                  "schema/product.json",
					Title:                "Helm Values Schema",
					Description:          "Schema for Helm values",
					AdditionalProperties: boolPtr(true),
				},
			},
			templateSchemaFile: "../testdata/full.schema.json",
		},
		{
			name: "noAdditionalProperties",
			config: &Config{
				Draft:                  2020,
				Indent:                 4,
				NoAdditionalProperties: true,
				Values: []string{
					"../testdata/noAdditionalProperties.yaml",
				},
				Output: "../testdata/output1.json",
			},
			templateSchemaFile: "../testdata/noAdditionalProperties.schema.json",
		},

		{
			name: "k8s ref alias",
			config: &Config{
				Draft:            2020,
				Indent:           4,
				K8sSchemaURL:     "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
				K8sSchemaVersion: "v1.33.1",
				Values: []string{
					"../testdata/k8sRef.yaml",
				},
				Output: "../testdata/k8sRef_output.json",
			},
			templateSchemaFile: "../testdata/k8sRef.schema.json",
		},

		{
			name: "ref draft 7",
			config: &Config{
				Draft:  7,
				Indent: 4,
				Values: []string{
					"../testdata/ref.yaml",
				},
				Output: "../testdata/ref-draft7_output.json",
			},
			templateSchemaFile: "../testdata/ref-draft7.schema.json",
		},
		{
			name: "ref draft 2020",
			config: &Config{
				Draft:  2020,
				Indent: 4,
				Values: []string{
					"../testdata/ref.yaml",
				},
				Output: "../testdata/ref-draft2020_output.json",
			},
			templateSchemaFile: "../testdata/ref-draft2020.schema.json",
		},

		{
			name: "bundle/simple",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "../",
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle/simple_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple.schema.json",
		},
		{
			name: "bundle/simple-disabled",
			config: &Config{
				Draft:  2020,
				Indent: 4,
				Bundle: false,
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle/simple-disabled_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple-disabled.schema.json",
		},
		{
			name: "bundle/without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          true,
				BundleWithoutID: true,
				BundleRoot:      "../",
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple-without-id.schema.json",
		},
		{
			name: "bundle/nested",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "..",
				Values: []string{
					"../testdata/bundle/nested.yaml",
				},
				Output: "../testdata/bundle/nested_output.json",
			},
			templateSchemaFile: "../testdata/bundle/nested.schema.json",
		},
		{
			name: "bundle/nested-without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          true,
				BundleWithoutID: true,
				BundleRoot:      "..",
				Values: []string{
					"../testdata/bundle/nested.yaml",
				},
				Output: "../testdata/bundle/nested-without-id_output.json",
			},
			templateSchemaFile: "../testdata/bundle/nested-without-id.schema.json",
		},
		{
			name: "bundle/fragment",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "..",
				Values: []string{
					"../testdata/bundle/fragment.yaml",
				},
				Output: "../testdata/bundle/fragment_output.json",
			},
			templateSchemaFile: "../testdata/bundle/fragment.schema.json",
		},
		{
			name: "bundle/fragment-without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          true,
				BundleWithoutID: true,
				BundleRoot:      "..",
				Values: []string{
					"../testdata/bundle/fragment.yaml",
				},
				Output: "../testdata/bundle/fragment-without-id_output.json",
			},
			templateSchemaFile: "../testdata/bundle/fragment-without-id.schema.json",
		},
		{
			name: "bundle/namecollision",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "..",
				Values: []string{
					"../testdata/bundle/namecollision.yaml",
				},
				Output: "../testdata/bundle/namecollision_output.json",
			},
			templateSchemaFile: "../testdata/bundle/namecollision.schema.json",
		},
		{
			name: "bundle/yaml",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "..",
				Values: []string{
					"../testdata/bundle/yaml.yaml",
				},
				Output: "../testdata/bundle/yaml_output.json",
			},
			templateSchemaFile: "../testdata/bundle/yaml.schema.json",
		},
		{
			// https://github.com/losisin/helm-values-schema-json/issues/159
			name: "bundle/root-ref",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "..",
				SchemaRoot: SchemaRoot{
					// Should be relative to CWD, which is this ./pkg dir
					Ref: "../testdata/bundle/simple-subschema.schema.json",
				},
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle/simple-root-ref_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple-root-ref.schema.json",
		},
		{
			name: "bundle/absolute path",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: filepath.Clean("/"),
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle/simple-abs_output.json",
			},
			templateSchemaFile: "../testdata/bundle/simple-absolute-root.schema.json",
		},
		{
			// https://github.com/losisin/helm-values-schema-json/issues/176
			name: "bundle/multiple-values-with-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          true,
				BundleRoot:      "..",
				BundleWithoutID: false,
				SchemaRoot: SchemaRoot{
					// Should be relative to CWD, which is this ./pkg dir
					Ref: "../testdata/bundle/simple-subschema.schema.json",
				},
				Values: []string{
					"../testdata/bundle/multiple-values-1.yaml",
					"../testdata/bundle/multiple-values-2.yaml",
				},
				Output: "../testdata/bundle/multiple-values_output.json",
			},
			templateSchemaFile: "../testdata/bundle/multiple-values.schema.json",
		},
		{
			// https://github.com/losisin/helm-values-schema-json/issues/176
			name: "bundle/multiple-values-without-id",
			config: &Config{
				Draft:           2020,
				Indent:          4,
				Bundle:          true,
				BundleRoot:      "..",
				BundleWithoutID: true,
				SchemaRoot: SchemaRoot{
					// Should be relative to CWD, which is this ./pkg dir
					Ref: "../testdata/bundle/simple-subschema.schema.json",
				},
				Values: []string{
					"../testdata/bundle/multiple-values-1.yaml",
					"../testdata/bundle/multiple-values-2.yaml",
				},
				Output: "../testdata/bundle/multiple-values-without-id_output.json",
			},
			templateSchemaFile: "../testdata/bundle/multiple-values-without-id.schema.json",
		},

		{
			name: "helm-docs",
			config: &Config{
				Draft:       2020,
				Indent:      4,
				UseHelmDocs: true,
				Values: []string{
					"../testdata/helm-docs/values.yaml",
				},
				Output: "../testdata/helm-docs/values_output.json",
			},
			templateSchemaFile: "../testdata/helm-docs/values.schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := os.Remove(tt.config.Output); err != nil && !os.IsNotExist(err) {
					t.Errorf("failed to remove values.schema.json: %v", err)
				}
			})

			cwd, err := os.Getwd()
			require.NoError(t, err)
			tt.config.SchemaRoot.RefReferrer = ReferrerDir(cwd)

			ctx := ContextWithLogger(t.Context(), t)
			err = GenerateJsonSchema(ctx, tt.config)
			require.NoError(t, err, "Error from pkg.GenerateJsonSchema")

			generatedBytes, err := os.ReadFile(tt.config.Output)
			require.NoError(t, err, "Error from os.ReadFile on config.Output")

			templateBytes, err := os.ReadFile(tt.templateSchemaFile)
			require.NoError(t, err, "Error from os.ReadFile on templateSchemaFile")

			// t.Logf("Generated output:\n%s\n", generatedBytes)

			assert.JSONEqf(t, string(templateBytes), string(generatedBytes), "Generated JSON schema %q does not match the template", tt.templateSchemaFile)
		})
	}
}

func TestGenerateJsonSchema_Errors(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name        string
		config      *Config
		setupFunc   func() error
		cleanupFunc func() error
		expectedErr error
	}{
		{
			name: "Missing values flag",
			config: &Config{
				Values: nil,
				Draft:  2020,
				Indent: 0,
			},
			expectedErr: errors.New("values flag is required"),
		},
		{
			name: "Invalid draft version",
			config: &Config{
				Values: []string{"../testdata/basic.yaml"},
				Draft:  5,
			},
			expectedErr: errors.New("invalid draft version"),
		},
		{
			name: "Negative indentation number",
			config: &Config{
				Values: []string{"../testdata/basic.yaml"},
				Draft:  2020,
				Output: "testdata/failure/output_readonly_schema.json",
				Indent: 0,
			},
			expectedErr: errors.New("indentation must be a positive number"),
		},
		{
			name: "Odd indentation number",
			config: &Config{
				Values: []string{"../testdata/basic.yaml"},
				Draft:  2020,
				Output: "testdata/failure/output_readonly_schema.json",
				Indent: 1,
			},
			expectedErr: errors.New("indentation must be an even number"),
		},
		{
			name: "Missing file",
			config: &Config{
				Values: []string{"missing.yaml"},
				Draft:  2020,
				Indent: 4,
			},
			expectedErr: errors.New("error reading YAML file(s)"),
		},
		{
			name: "Fail Unmarshal",
			config: &Config{
				Values: []string{"../testdata/fail"},
				Output: "testdata/failure/output_readonly_schema.json",
				Draft:  2020,
				Indent: 4,
			},
			expectedErr: errors.New("error unmarshaling YAML"),
		},
		{
			name: "Read-only filesystem",
			config: &Config{
				Values: []string{"../testdata/basic.yaml"},
				Output: "testdata/failure/output_readonly_schema.json",
				Draft:  2020,
				Indent: 4,
			},
			expectedErr: errors.New("write output schema:"),
		},
		{
			name: "bundle invalid root path",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "\000", // null byte is invalid in both linux & windows
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle_output.json",
			},
			expectedErr: errors.New(testutil.PerGOOS{
				Default: "bundle root \x00: open " + cwd + "/\x00: invalid argument",
				Windows: "bundle root \x00: get absolute path: invalid argument",
			}.String()),
		},
		{
			name: "bundle wrong root path",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: ".",
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle_output.json",
			},
			expectedErr: errors.New("path escapes from parent"),
		},
		{
			name: "bundle invalid root ref file",
			config: &Config{
				Draft:      2020,
				Indent:     4,
				Bundle:     true,
				BundleRoot: "..",
				SchemaRoot: SchemaRoot{
					Ref: "::",
				},
				Values: []string{
					"../testdata/bundle/simple.yaml",
				},
				Output: "../testdata/bundle_output.json",
			},
			expectedErr: errors.New("bundle schemas: /$ref: parse \"::\": missing protocol scheme"),
		},
		{
			name: "invalid k8s ref alias",
			config: &Config{
				Draft:            2020,
				Indent:           4,
				K8sSchemaURL:     "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				K8sSchemaVersion: "",
				Values: []string{
					"../testdata/k8sRef.yaml",
				},
				Output: "../testdata/k8sRef_output.json",
			},
			expectedErr: errors.New("/properties/memory: must set k8sSchemaVersion config when using \"$ref: $k8s/...\""),
		},
		{
			name: "invalid subschema type",
			config: &Config{
				Draft:  2020,
				Indent: 4,
				Values: []string{
					"../testdata/fail-type.yaml",
				},
				Output: "../testdata/fail-type_output.json",
			},
			expectedErr: errors.New("/properties/nameOverride/type: invalid type \"foobar\", must be one of: array, boolean, integer, null, number, object, string"),
		},
		{
			name: "invalid helm-docs comment",
			config: &Config{
				Draft:       2020,
				Indent:      4,
				UseHelmDocs: true,
				Values: []string{
					"../testdata/helm-docs/values-fail.yaml",
				},
				Output: "../testdata/helm-docs/values-fail_output.json",
			},
			expectedErr: errors.New("parse schema: /foo: parse helm-docs comment: '# @schema' comments are not supported in helm-docs comments."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := os.Remove(tt.config.Output); err != nil && !os.IsNotExist(err) {
					t.Errorf("failed to remove values.schema.json: %v", err)
				}
			})

			if tt.setupFunc != nil {
				err := tt.setupFunc()
				assert.NoError(t, err)
			}

			ctx := ContextWithLogger(t.Context(), t)
			err := GenerateJsonSchema(ctx, tt.config)
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

func TestGenerateJsonSchema_AbsInputError(t *testing.T) {
	testutil.MakeGetwdFail(t)

	ctx := ContextWithLogger(t.Context(), t)
	err := GenerateJsonSchema(ctx, &Config{
		Values: []string{"foo/bar.yaml"},
		Draft:  2020,
		Indent: 4,
	})
	require.ErrorContains(t, err, "read --values=\"foo/bar.yaml\": get absolute path:")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenerateJsonSchema_AbsOutputError(t *testing.T) {
	input := testutil.WriteTempFile(t, "values-*.yaml", []byte(""))
	testutil.MakeGetwdFail(t)

	ctx := ContextWithLogger(t.Context(), t)
	err := GenerateJsonSchema(ctx, &Config{
		Values: []string{input.Name()}, // unaffected by failing [os.Getwd] because it is an absolute path
		Output: "foo.json",
		Draft:  2020,
		Indent: 4,
		Bundle: true,
	})
	require.ErrorContains(t, err, "output foo.json: get absolute path: ")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenerateJsonSchema_AbsBundleRootError(t *testing.T) {
	input := testutil.WriteTempFile(t, "values-*.yaml", []byte(""))
	testutil.MakeGetwdFail(t)

	ctx := ContextWithLogger(t.Context(), t)
	err := GenerateJsonSchema(ctx, &Config{
		Values:     []string{input.Name()}, // unaffected by failing [os.Getwd] because it is an absolute path
		Output:     "/tmp/foo.json",        // unaffected by failing [os.Getwd] because it is an absolute path
		Draft:      2020,
		Indent:     4,
		Bundle:     true,
		BundleRoot: "foo",
	})
	require.ErrorContains(t, err, "bundle root foo: get absolute path: ")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenerateJsonSchema_AdditionalProperties(t *testing.T) {
	tests := []struct {
		name                   string
		noAdditionalProperties bool
		additionalProperties   *bool
		expected               any
	}{
		{
			name:                 "AdditionalProperties set to true",
			additionalProperties: boolPtr(true),
			expected:             true,
		},
		{
			name:                 "AdditionalProperties set to false",
			additionalProperties: boolPtr(false),
			expected:             false,
		},
		{
			name:                 "AdditionalProperties not set",
			additionalProperties: nil,
			expected:             nil,
		},
		{
			name:                   "AdditionalProperties not set, but NoAdditionalProperties set",
			additionalProperties:   nil,
			noAdditionalProperties: true,
			expected:               false,
		},
		{
			name:                   "NoAdditionalProperties set, but AdditionalProperties set to true",
			additionalProperties:   boolPtr(true),
			noAdditionalProperties: true,
			expected:               true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Values:                 []string{"../testdata/empty.yaml"},
				Output:                 "../testdata/empty.schema.json",
				Draft:                  2020,
				Indent:                 4,
				NoAdditionalProperties: tt.noAdditionalProperties,
				SchemaRoot: SchemaRoot{
					ID:                   "",
					Title:                "",
					Description:          "",
					AdditionalProperties: tt.additionalProperties,
				},
			}

			ctx := ContextWithLogger(t.Context(), t)
			err := GenerateJsonSchema(ctx, config)
			assert.NoError(t, err)

			generatedBytes, err := os.ReadFile(config.Output)
			assert.NoError(t, err)

			var generatedSchema map[string]any
			err = json.Unmarshal(generatedBytes, &generatedSchema)
			assert.NoError(t, err)

			if tt.expected == nil {
				_, exists := generatedSchema["additionalProperties"]
				assert.False(t, exists, "additionalProperties should not be present in the generated schema")
			} else {
				assert.Equal(t, tt.expected, generatedSchema["additionalProperties"], "additionalProperties value mismatch")
			}

			if err := os.Remove(config.Output); err != nil && !os.IsNotExist(err) {
				t.Errorf("failed to remove values.schema.json: %v", err)
			}
		})
	}
}

func TestWriteOutput_JSONError(t *testing.T) {
	schema := &Schema{Type: func() {}}

	ctx := ContextWithLogger(t.Context(), t)
	err := WriteOutput(ctx, schema, os.DevNull, "  ")
	require.ErrorContains(t, err, "unsupported type: func()")
}
