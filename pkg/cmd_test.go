package pkg

import (
	"io"
	"os"
	"path/filepath"
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
				"--values=../testdata/basic.yaml",
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
				"--values=/non/existing/file.yaml",
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
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		args []string
		conf Config
	}{
		{
			[]string{"--values", "values.yaml"},
			Config{
				Values:       []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},
		{
			[]string{"-f", "values.yaml"},
			Config{
				Values:       []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--values", "values1.yaml values2.yaml", "--indent", "2"},
			Config{
				Values:       []string{"values1.yaml values2.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       2,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--values", "values.yaml", "--output", "my.schema.json", "--draft", "2019", "--indent", "2"},
			Config{
				Values:       []string{"values.yaml"},
				Output:       "my.schema.json",
				Draft:        2019,
				Indent:       2,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--values", "values.yaml", "--output", "my.schema.json", "--draft", "2019", "--k8s-schema-url", "foobar"},
			Config{
				Values:       []string{"values.yaml"},
				Output:       "my.schema.json",
				Draft:        2019,
				Indent:       4,
				K8sSchemaURL: "foobar",
			},
		},

		{
			[]string{"--values", "values.yaml", "--schema-root.id", "http://example.com/schema", "--schema-root.ref", "schema/product.json", "--schema-root.title", "MySchema", "--schema-root.description", "My schema description"},
			Config{
				Values:       []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				SchemaRoot: SchemaRoot{
					ID:          "http://example.com/schema",
					Ref:         "schema/product.json",
					RefReferrer: ReferrerDir(cwd),
					Title:       "MySchema",
					Description: "My schema description",
				},
			},
		},

		{
			[]string{"--bundle", "--bundle-root", "/foo/bar", "--bundle-without-id"},
			Config{
				Values:          []string{"values.yaml"},
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
			[]string{"--bundle=true", "--bundle-root", "/foo/bar", "--bundle-without-id=true"},
			Config{
				Values:          []string{"values.yaml"},
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
			[]string{"--bundle=false", "--bundle-root", "", "--bundle-without-id=false"},
			Config{
				Values:          []string{"values.yaml"},
				Indent:          4,
				Output:          "values.schema.json",
				Draft:           2020,
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				Bundle:          false,
				BundleRoot:      "",
				BundleWithoutID: false,
			},
		},

		{
			[]string{"--use-helm-docs"},
			Config{
				Values:       []string{"values.yaml"},
				Indent:       4,
				Output:       "values.schema.json",
				Draft:        2020,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				UseHelmDocs:  true,
			},
		},
		{
			[]string{"--use-helm-docs=false"},
			Config{
				Values:       []string{"values.yaml"},
				Indent:       4,
				Output:       "values.schema.json",
				Draft:        2020,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				UseHelmDocs:  false,
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
		{[]string{"--values"}, "flag needs an argument"},
		{[]string{"--draft", "foo"}, "invalid syntax"},
		{[]string{"--foo"}, "unknown flag: --foo"},
		{[]string{"--schema-root.additional-properties=123"}, "invalid syntax"},
		{[]string{"--bundle=123"}, "invalid syntax"},
		{[]string{"--bundle-without-id=123"}, "invalid syntax"},
		{[]string{"--use-helm-docs=123"}, "invalid syntax"},
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
	tmpFile := createTempFile(t, "config-*.yaml")

	tests := []struct {
		name   string
		config string
		want   Config
	}{
		{
			name: "ValidConfig",
			config: `
values:
  - testdata/empty.yaml
  - testdata/meta.yaml
output: values.schema.json
draft: 2020
indent: 2
bundle: true
bundleRoot: ./
bundleWithoutID: true
useHelmDocs: true
schemaRoot:
  id: https://example.com/schema
  ref: schema/product.json
  title: Helm Values Schema
  description: Schema for Helm values
  additionalProperties: true
`,
			want: Config{
				Values:          []string{"testdata/empty.yaml", "testdata/meta.yaml"},
				Output:          "values.schema.json",
				Draft:           2020,
				Indent:          2,
				Bundle:          true,
				BundleRoot:      "./",
				BundleWithoutID: true,
				UseHelmDocs:     true,
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
				SchemaRoot: SchemaRoot{
					Title:                "Helm Values Schema",
					ID:                   "https://example.com/schema",
					Ref:                  "schema/product.json",
					RefReferrer:          ReferrerDir(filepath.Dir(tmpFile.Name())),
					Description:          "Schema for Helm values",
					AdditionalProperties: boolPtr(true),
				},
			},
		},
		{
			name:   "EmptyConfig",
			config: `# just a comment`,
			want: Config{
				Values:       []string{"values.yaml"},
				Output:       "values.schema.json",
				Draft:        2020,
				Indent:       4,
				K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// reuse the same file so we can have a predictable name as ref referrer
			resetFile(t, tmpFile, tt.config)

			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags([]string{"--config=" + tmpFile.Name()}))
			conf, err := LoadConfig(cmd)

			require.NoError(t, err)
			assert.NotNil(t, conf)
			assert.Equal(t, tt.want, *conf)
		})
	}
}

func TestLoadConfig_Error(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "invalid syntax",
			config: `
values: "invalid" "syntax"
values:
`,
			wantErr: "yaml: line 1: did not find expected key",
		},
		{
			name:    "invalid value for type",
			config:  `draft: "invalid"`,
			wantErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid` into int",
		},
		{
			name:    "missing file",
			config:  "",
			wantErr: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configFilePath string
			if tt.config != "" {
				configFilePath = writeTempFile(t, tt.config)
			} else {
				configFilePath = "nonexistent.yaml"
			}
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags([]string{"--config=" + configFilePath}))
			conf, err := LoadConfig(cmd)

			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, conf)
		})
	}
}

func TestLoadConfig_LoadFlagError(t *testing.T) {
	failConfigFlagLoad = true
	defer func() { failConfigFlagLoad = false }()

	cmd := NewCmd()
	_, err := LoadConfig(cmd)
	assert.ErrorContains(t, err, "load flags: ")
}

func TestLoadConfig_UnmarshalError(t *testing.T) {
	failConfigUnmarshal = true
	defer func() { failConfigUnmarshal = false }()

	cmd := NewCmd()
	_, err := LoadConfig(cmd)
	assert.ErrorContains(t, err, "parsing config: ")
}

func TestLoadConfig_SchemaRootRefReferrerConfigError(t *testing.T) {
	failConfigConfigRefReferrerAbs = true
	defer func() { failConfigConfigRefReferrerAbs = false }()

	configFile := writeTempFile(t, `
schemaRoot:
  ref: foo/bar
`)

	cmd := NewCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--config=" + configFile}))
	_, err := LoadConfig(cmd)
	assert.ErrorContains(t, err, "resolve absolute path of config file: ")
}

func TestLoadConfig_SchemaRootRefReferrerFlagError(t *testing.T) {
	// Setting up to make [os.Getwd] to fail, which on Linux can be done
	// by deleting the directory you're currently in.
	tempDir, err := os.MkdirTemp("", "schema-cwd-*")
	require.NoError(t, err)
	previousWorkingDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() { assert.NoError(t, os.Chdir(previousWorkingDir)) })
	require.NoError(t, os.Remove(tempDir))

	cmd := NewCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--schema-root.ref=foo/bar"}))
	_, err = LoadConfig(cmd)
	assert.ErrorContains(t, err, "resolve current working directory: getwd: no such file or directory")
}

func TestMergeConfig(t *testing.T) {
	tmpFile := createTempFile(t, "config-*.yaml")
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name  string
		file  string
		flags []string
		want  *Config
	}{
		{
			name: "flag overrides files",
			file: `
values: [fileInput.yaml]
output: fileOutput.json
draft: 2020
indent: 4
noAdditionalProperties: true
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
useHelmDocs: true
schemaRoot:
  id: fileID
  ref: fileRef
  title: fileTitle
  description: fileDescription
  additionalProperties: true
`,
			flags: []string{
				"--values=flagInput.yaml",
				"--output=flagOutput.json",
				"--draft=2019",
				"--indent=2",
				"--no-additional-properties=false",
				"--k8s-schema-url=flagURL",
				"--k8s-schema-version=flagVersion",
				"--use-helm-docs=false",
				"--schema-root.id=flagID",
				"--schema-root.ref=flagRef",
				"--schema-root.title=flagTitle",
				"--schema-root.description=flagDescription",
				"--schema-root.additional-properties=false",
			},
			want: &Config{
				Values:                 []string{"flagInput.yaml"},
				Output:                 "flagOutput.json",
				Draft:                  2019,
				Indent:                 2,
				NoAdditionalProperties: false,
				K8sSchemaURL:           "flagURL",
				K8sSchemaVersion:       "flagVersion",
				UseHelmDocs:            false,
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Ref:                  "flagRef",
					RefReferrer:          ReferrerDir(cwd),
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: boolPtr(false),
				},
			},
		},
		{
			name: "file overrides defaults",
			file: `
values: [fileInput.yaml]
output: fileOutput.json
draft: 2020
indent: 4
noAdditionalProperties: true
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
useHelmDocs: true
schemaRoot:
  id: fileID
  ref: fileRef
  title: fileTitle
  description: fileDescription
  additionalProperties: true
`,
			flags: []string{},
			want: &Config{
				Values:                 []string{"fileInput.yaml"},
				Output:                 "fileOutput.json",
				Draft:                  2020,
				Indent:                 4,
				K8sSchemaURL:           "fileURL",
				K8sSchemaVersion:       "fileVersion",
				NoAdditionalProperties: true,
				UseHelmDocs:            true,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Ref:                  "fileRef",
					RefReferrer:          ReferrerDir(filepath.Dir(tmpFile.Name())),
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: boolPtr(true),
				},
			},
		},
		{
			name: "flag partial overrides file",
			file: `
values: [fileInput.yaml]
output: fileOutput.json
draft: 2020
indent: 4
noAdditionalProperties: true
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
useHelmDocs: true
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
				Values:                 []string{"fileInput.yaml"},
				Output:                 "flagOutput.json",
				Draft:                  2020,
				Indent:                 4,
				K8sSchemaURL:           "fileURL",
				K8sSchemaVersion:       "fileVersion",
				NoAdditionalProperties: true,
				UseHelmDocs:            true,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Ref:                  "fileRef",
					RefReferrer:          ReferrerDir(filepath.Dir(tmpFile.Name())),
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
				"--values=flagInput.yaml",
				"--output=flagOutput.json",
				"--draft=2019",
				"--indent=2",
				"--no-additional-properties=false",
				"--k8s-schema-url=flagURL",
				"--k8s-schema-version=flagVersion",
				"--use-helm-docs=true",
				"--schema-root.id=flagID",
				"--schema-root.ref=flagRef",
				"--schema-root.title=flagTitle",
				"--schema-root.description=flagDescription",
				"--schema-root.additional-properties=false",
			},
			want: &Config{
				Values:           []string{"flagInput.yaml"},
				Output:           "flagOutput.json",
				Draft:            2019,
				Indent:           2,
				K8sSchemaURL:     "flagURL",
				K8sSchemaVersion: "flagVersion",
				UseHelmDocs:      true,
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Ref:                  "flagRef",
					RefReferrer:          ReferrerDir(cwd),
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: boolPtr(false),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFile(t, tmpFile, tt.file)

			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags(append(tt.flags, "--config="+tmpFile.Name())))

			conf, err := LoadConfig(cmd)
			require.NoError(t, err)

			assert.Equal(t, tt.want, conf)
		})
	}
}

func resetFile(t *testing.T, file *os.File, content string) {
	_, err := file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(0))
	_, err = file.WriteString(content)
	require.NoError(t, file.Sync())
}

func createTempFile(t *testing.T, pattern string) *os.File {
	tmpFile, err := os.CreateTemp("", pattern)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, tmpFile.Close())
		assert.NoError(t, os.Remove(tmpFile.Name()))
	})
	return tmpFile
}

func writeTempFile(t *testing.T, content string) string {
	tmpFile := createTempFile(t, "config-*.yaml")
	_, err := tmpFile.WriteString(content)
	require.NoError(t, err)
	return tmpFile.Name()
}
