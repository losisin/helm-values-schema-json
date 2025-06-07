package pkg

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name: "success",
			args: []string{
				"--input=../testdata/basic.yaml",
				"--output=/dev/null",
			},
		},
		{
			name: "fail reading config",
			args: []string{
				"--config=nonexisting.yaml",
			},
			wantErr: "open nonexisting.yaml: no such file or directory",
		},
		{
			name: "fail execution",
			args: []string{
				"--input=/non/existing/file.yaml",
			},
			wantErr: "error reading YAML file(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags(tt.args))
			err := cmd.Execute()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseFlagsPass(t *testing.T) {
	tests := []struct {
		args []string
		conf Config
	}{
		{
			[]string{"--input", "values.yaml"},
			Config{
				Input:        []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},
		{
			[]string{"-i", "values.yaml"},
			Config{
				Input:        []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--input", "values1.yaml values2.yaml", "--indent", "2"},
			Config{
				Input:        []string{"values1.yaml values2.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       2,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--input", "values.yaml", "--output", "my.schema.json", "--draft", "2019", "--indent", "2"},
			Config{
				Input:        []string{"values.yaml"},
				Output:       "my.schema.json",
				Draft:        2019,
				Indent:       2,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--input", "values.yaml", "--output", "my.schema.json", "--draft", "2019", "--k8sSchemaURL", "foobar"},
			Config{
				Input:        []string{"values.yaml"},
				Output:       "my.schema.json",
				Draft:        2019,
				Indent:       4,
				K8sSchemaURL: "foobar",
			},
		},

		{
			[]string{"--input", "values.yaml", "--schemaRoot.id", "http://example.com/schema", "--schemaRoot.ref", "schema/product.json", "--schemaRoot.title", "MySchema", "--schemaRoot.description", "My schema description"},
			Config{
				Input:        []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				SchemaRoot: SchemaRoot{
					ID:          "http://example.com/schema",
					Ref:         "schema/product.json",
					Title:       "MySchema",
					Description: "My schema description",
				},
			},
		},

		{
			[]string{"--bundle", "--bundleRoot", "/foo/bar", "--bundleWithoutID"},
			Config{
				Input:           []string{"values.yaml"},
				Indent:          4,
				Output:          "values.schema.json",
				Draft:           2020,
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				Bundle:          true,
				BundleRoot:      "/foo/bar",
				BundleWithoutID: true,
			},
		},
		{
			[]string{"--bundle=true", "--bundleRoot", "/foo/bar", "--bundleWithoutID=true"},
			Config{
				Input:           []string{"values.yaml"},
				Indent:          4,
				Output:          "values.schema.json",
				Draft:           2020,
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				Bundle:          true,
				BundleRoot:      "/foo/bar",
				BundleWithoutID: true,
			},
		},
		{
			[]string{"--bundle=false", "--bundleRoot", "", "--bundleWithoutID=false"},
			Config{
				Input:           []string{"values.yaml"},
				Indent:          4,
				Output:          "values.schema.json",
				Draft:           2020,
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				Bundle:          false,
				BundleRoot:      "",
				BundleWithoutID: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags(tt.args))
			conf, err := LoadConfig(cmd)
			assert.NoError(t, err)
			assert.Equal(t, &tt.conf, conf, "conf")
		})
	}
}

func TestParseFlagsFail(t *testing.T) {
	tests := []struct {
		args   []string
		errStr string
	}{
		{[]string{"--input"}, "flag needs an argument"},
		{[]string{"--draft", "foo"}, "invalid syntax"},
		{[]string{"--foo"}, "unknown flag: --foo"},
		{[]string{"--schemaRoot.additionalProperties=123"}, "invalid syntax"},
		{[]string{"--bundle=123"}, "invalid syntax"},
		{[]string{"--bundleWithoutID=123"}, "invalid syntax"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := NewCmd()
			err := cmd.ParseFlags(tt.args)
			assert.ErrorContains(t, err, tt.errStr)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectedConf  Config
		expectedErr   bool
	}{
		{
			name: "ValidConfig",
			configContent: `
input:
  - testdata/empty.yaml
  - testdata/meta.yaml
output: values.schema.json
draft: 2020
indent: 2
bundle: true
bundleRoot: ./
bundleWithoutID: true
schemaRoot:
  id: https://example.com/schema
  ref: schema/product.json
  title: Helm Values Schema
  description: Schema for Helm values
  additionalProperties: true
`,
			expectedConf: Config{
				Input:           []string{"testdata/empty.yaml", "testdata/meta.yaml"},
				Output:          "values.schema.json",
				Draft:           2020,
				Indent:          2,
				Bundle:          true,
				BundleRoot:      "./",
				BundleWithoutID: true,
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				SchemaRoot: SchemaRoot{
					Title:                "Helm Values Schema",
					ID:                   "https://example.com/schema",
					Ref:                  "schema/product.json",
					Description:          "Schema for Helm values",
					AdditionalProperties: boolPtr(true),
				},
			},
			expectedErr: false,
		},
		{
			name: "InvalidConfig",
			configContent: `
input: "invalid" "input"
input:
`,
			expectedErr: true,
		},
		{
			name:          "InvalidYAML",
			configContent: `draft: "invalid"`,
			expectedErr:   true,
		},
		{
			name:          "MissingFile",
			configContent: "",
			expectedErr:   true,
		},
		{
			name:          "EmptyConfig",
			configContent: `# just a comment`,
			expectedConf: Config{
				Input:        []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configFilePath string
			if tt.configContent != "" {
				configFilePath = writeTempFile(t, tt.configContent)
			} else {
				configFilePath = "nonexistent.yaml"
			}
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags([]string{"--config=" + configFilePath}))
			conf, err := LoadConfig(cmd)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, conf)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conf)
				assert.Equal(t, tt.expectedConf, *conf)
			}
		})
	}
}

func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		flags []string
		want  *Config
	}{
		{
			name: "flag overrides files",
			file: `
input: [fileInput.yaml]
output: fileOutput.json
draft: 2020
indent: 4
noAdditionalProperties: true
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
schemaRoot:
  id: fileID
  ref: fileRef
  title: fileTitle
  description: fileDescription
  additionalProperties: true
`,
			flags: []string{
				"--input=flagInput.yaml",
				"--output=flagOutput.json",
				"--draft=2019",
				"--indent=2",
				"--noAdditionalProperties=false",
				"--k8sSchemaURL=flagURL",
				"--k8sSchemaVersion=flagVersion",
				"--schemaRoot.id=flagID",
				"--schemaRoot.ref=flagRef",
				"--schemaRoot.title=flagTitle",
				"--schemaRoot.description=flagDescription",
				"--schemaRoot.additionalProperties=false",
			},
			want: &Config{
				Input:                  []string{"flagInput.yaml"},
				Output:                 "flagOutput.json",
				Draft:                  2019,
				Indent:                 2,
				NoAdditionalProperties: false,
				K8sSchemaURL:           "flagURL",
				K8sSchemaVersion:       "flagVersion",
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Ref:                  "flagRef",
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: boolPtr(false),
				},
			},
		},
		{
			name: "file overrides defaults",
			file: `
input: [fileInput.yaml]
output: fileOutput.json
draft: 2020
indent: 4
noAdditionalProperties: true
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
schemaRoot:
  id: fileID
  ref: fileRef
  title: fileTitle
  description: fileDescription
  additionalProperties: true
`,
			flags: []string{},
			want: &Config{
				Input:                  []string{"fileInput.yaml"},
				Output:                 "fileOutput.json",
				Draft:                  2020,
				Indent:                 4,
				K8sSchemaURL:           "fileURL",
				K8sSchemaVersion:       "fileVersion",
				NoAdditionalProperties: true,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Ref:                  "fileRef",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: boolPtr(true),
				},
			},
		},
		{
			name: "flag partial overrides file",
			file: `
input: [fileInput.yaml]
output: fileOutput.json
draft: 2020
indent: 4
noAdditionalProperties: true
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
schemaRoot:
  id: fileID
  ref: fileRef
  title: fileTitle
  description: fileDescription
  additionalProperties: true
`,
			flags: []string{
				"--output=flagOutput.json",
			},
			want: &Config{
				Input:                  []string{"fileInput.yaml"},
				Output:                 "flagOutput.json",
				Draft:                  2020,
				Indent:                 4,
				K8sSchemaURL:           "fileURL",
				K8sSchemaVersion:       "fileVersion",
				NoAdditionalProperties: true,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Ref:                  "fileRef",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: boolPtr(true),
				},
			},
		},
		{
			name: "flag overrides empty file",
			file: "",
			flags: []string{
				"--input=flagInput.yaml",
				"--output=flagOutput.json",
				"--draft=2019",
				"--indent=2",
				"--noAdditionalProperties=false",
				"--k8sSchemaURL=flagURL",
				"--k8sSchemaVersion=flagVersion",
				"--schemaRoot.id=flagID",
				"--schemaRoot.ref=flagRef",
				"--schemaRoot.title=flagTitle",
				"--schemaRoot.description=flagDescription",
				"--schemaRoot.additionalProperties=false",
			},
			want: &Config{
				Input:            []string{"flagInput.yaml"},
				Output:           "flagOutput.json",
				Draft:            2019,
				Indent:           2,
				K8sSchemaURL:     "flagURL",
				K8sSchemaVersion: "flagVersion",
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Ref:                  "flagRef",
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: boolPtr(false),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags(append(tt.flags, "--config="+writeTempFile(t, tt.file))))

			conf, err := LoadConfig(cmd)
			require.NoError(t, err)

			assert.Equal(t, tt.want, conf)
		})
	}
}

func writeTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile.Name()))
	})
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	return tmpFile.Name()
}
