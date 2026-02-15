package pkg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/losisin/helm-values-schema-json/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantErr   string
		wantErrIs error
		wantOut   string
	}{
		{
			name: "success",
			args: []string{"--values=../testdata/basic.yaml", "--output=" + os.DevNull},
		},
		{
			name:      "fail reading config",
			args:      []string{"--config=nonexisting.yaml"},
			wantErrIs: os.ErrNotExist,
		},
		{
			name:    "fail execution",
			args:    []string{"--values=/non/existing/file.yaml"},
			wantErr: "error reading YAML file(s)",
		},
		{
			name:    "version flag",
			args:    []string{"--version"},
			wantOut: "helm schema version test\n",
		},
		{
			name:    "version subcommand",
			args:    []string{"version"},
			wantOut: "helm schema version test\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmd()
			cmd.Version = "test"
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			switch {
			case tt.wantErrIs != nil:
				assert.ErrorIs(t, err, tt.wantErrIs)
			case tt.wantErr != "":
				assert.ErrorContains(t, err, tt.wantErr)
			default:
				assert.NoError(t, err)
			}
			testutil.Equal(t, tt.wantOut, buf.String())
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
				Values:         []string{"values.yaml"},
				Output:         "values.schema.json",
				Draft:          2020,
				Indent:         4,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},
		{
			[]string{"-f", "values.yaml"},
			Config{
				Values:         []string{"values.yaml"},
				Output:         "values.schema.json",
				Draft:          2020,
				Indent:         4,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--values", "values1.yaml values2.yaml", "--indent", "2"},
			Config{
				Values:         []string{"values1.yaml values2.yaml"},
				Output:         "values.schema.json",
				Draft:          2020,
				Indent:         2,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--values", "values.yaml", "--output", "my.schema.json", "--draft", "2019", "--indent", "2"},
			Config{
				Values:         []string{"values.yaml"},
				Output:         "my.schema.json",
				Draft:          2019,
				Indent:         2,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--values", "values.yaml", "--output", "my.schema.json", "--draft", "2019", "--k8s-schema-url", "foobar"},
			Config{
				Values:         []string{"values.yaml"},
				Output:         "my.schema.json",
				Draft:          2019,
				Indent:         4,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "foobar",
			},
		},

		{
			[]string{"--values", "values.yaml", "--schema-root.id", "http://example.com/schema", "--schema-root.ref", "schema/product.json", "--schema-root.title", "MySchema", "--schema-root.description", "My schema description"},
			Config{
				Values:         []string{"values.yaml"},
				Output:         "values.schema.json",
				Draft:          2020,
				Indent:         4,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
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
			[]string{"--recursive", "--recursive-needs", "foo-bar", "--no-recursive-needs", "--hidden", "--no-gitignore"},
			Config{
				Values:           []string{"values.yaml"},
				Indent:           4,
				Output:           "values.schema.json",
				Draft:            2020,
				Recursive:        true,
				RecursiveNeeds:   []string{"foo-bar"},
				NoRecursiveNeeds: true,
				Hidden:           true,
				NoGitIgnore:      true,
				K8sSchemaURL:     "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},

		{
			[]string{"--bundle", "--bundle-root", "/foo/bar", "--bundle-without-id"},
			Config{
				Values:          []string{"values.yaml"},
				Indent:          4,
				Output:          "values.schema.json",
				Draft:           2020,
				RecursiveNeeds:  []string{"Chart.yaml"},
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
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
				RecursiveNeeds:  []string{"Chart.yaml"},
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
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
				RecursiveNeeds:  []string{"Chart.yaml"},
				K8sSchemaURL:    "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
				Bundle:          false,
				BundleRoot:      "",
				BundleWithoutID: false,
			},
		},

		{
			[]string{"--use-helm-docs"},
			Config{
				Values:         []string{"values.yaml"},
				Indent:         4,
				Output:         "values.schema.json",
				Draft:          2020,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
				UseHelmDocs:    true,
			},
		},
		{
			[]string{"--use-helm-docs=false"},
			Config{
				Values:         []string{"values.yaml"},
				Indent:         4,
				Output:         "values.schema.json",
				Draft:          2020,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
				UseHelmDocs:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags(tt.args))
			conf, err := LoadConfig(cmd)
			assert.NoError(t, err)
			testutil.Equal(t, &tt.conf, conf, "conf")
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
		{[]string{"--recursive=yes"}, "invalid syntax"},
		{[]string{"--hidden=yes"}, "invalid syntax"},
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
	tmpFile := testutil.CreateTempFile(t, "config-*.yaml")

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
recursive: true
recursiveNeeds: [fileNeeds]
noRecursiveNeeds: true
hidden: true
noGitIgnore: true
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
				Values:           []string{"testdata/empty.yaml", "testdata/meta.yaml"},
				Output:           "values.schema.json",
				Draft:            2020,
				Indent:           2,
				Recursive:        true,
				RecursiveNeeds:   []string{"fileNeeds"},
				NoRecursiveNeeds: true,
				Hidden:           true,
				NoGitIgnore:      true,
				Bundle:           true,
				BundleRoot:       "./",
				BundleWithoutID:  true,
				UseHelmDocs:      true,
				K8sSchemaURL:     "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
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
				Values:         []string{"values.yaml"},
				Output:         "values.schema.json",
				Draft:          2020,
				Indent:         4,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// reuse the same file so we can have a predictable name as ref referrer
			testutil.ResetFile(t, tmpFile, []byte(tt.config))

			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags([]string{"--config=" + tmpFile.Name()}))
			conf, err := LoadConfig(cmd)

			require.NoError(t, err)
			require.NotNil(t, conf)
			testutil.Equal(t, tt.want, *conf, "conf")
		})
	}
}

func TestLoadConfig_Error(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantErr   string
		wantErrIs error
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
			name:      "missing file",
			config:    "",
			wantErrIs: os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configFilePath string
			if tt.config != "" {
				configFilePath = testutil.WriteTempFile(t, "config-*.yaml", []byte(tt.config)).Name()
			} else {
				configFilePath = "nonexistent.yaml"
			}
			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags([]string{"--config=" + configFilePath}))
			conf, err := LoadConfig(cmd)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
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

	configFile := testutil.WriteTempFile(t, "config-*.yaml", []byte(`
schemaRoot:
  ref: foo/bar
`))

	cmd := NewCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--config=" + configFile.Name()}))
	_, err := LoadConfig(cmd)
	assert.ErrorContains(t, err, "resolve absolute path of config file: ")
}

func TestLoadConfig_SchemaRootRefReferrerFlagError(t *testing.T) {
	testutil.MakeGetwdFail(t)

	cmd := NewCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--schema-root.ref=foo/bar"}))
	_, err := LoadConfig(cmd)
	assert.ErrorContains(t, err, "resolve current working directory: ")
}

func TestMergeConfig(t *testing.T) {
	tmpFile := testutil.CreateTempFile(t, "config-*.yaml")
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
noDefaultGlobal: true
recursive: true
recursiveNeeds: [fileNeeds]
noRecursiveNeeds: true
hidden: true
noGitIgnore: true
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
				"--no-default-global=false",
				"--recursive=false",
				"--recursive-needs=flagNeeds",
				"--no-recursive-needs=false",
				"--hidden=false",
				"--no-gitignore=false",
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
				Recursive:              false,
				RecursiveNeeds:         []string{"flagNeeds"},
				NoRecursiveNeeds:       false,
				Hidden:                 false,
				NoGitIgnore:            false,
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
noDefaultGlobal: true
recursive: true
recursiveNeeds: [fileNeeds]
noRecursiveNeeds: true
hidden: true
noGitIgnore: true
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
				NoDefaultGlobal:        true,
				Recursive:              true,
				RecursiveNeeds:         []string{"fileNeeds"},
				NoRecursiveNeeds:       true,
				Hidden:                 true,
				NoGitIgnore:            true,
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
noDefaultGlobal: true
recursive: true
recursiveNeeds: [fileNeeds]
noRecursiveNeeds: true
hidden: true
noGitIgnore: true
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
				NoDefaultGlobal:        true,
				Recursive:              true,
				RecursiveNeeds:         []string{"fileNeeds"},
				NoRecursiveNeeds:       true,
				Hidden:                 true,
				NoGitIgnore:            true,
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
				"--no-default-global=false",
				"--recursive=true",
				"--recursive-needs=flagNeeds",
				"--no-recursive-needs=true",
				"--hidden=true",
				"--no-gitignore=true",
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
				Recursive:        true,
				RecursiveNeeds:   []string{"flagNeeds"},
				NoRecursiveNeeds: true,
				Hidden:           true,
				NoGitIgnore:      true,
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
			testutil.ResetFile(t, tmpFile, []byte(tt.file))

			cmd := NewCmd()
			require.NoError(t, cmd.ParseFlags(append(tt.flags, "--config="+tmpFile.Name())))

			conf, err := LoadConfig(cmd)
			require.NoError(t, err)

			testutil.Equal(t, tt.want, conf, "conf")
		})
	}
}

func TestLoadFileConfigOverwrite(t *testing.T) {
	tmpFile := testutil.CreateTempFile(t, "config-*.yaml")

	tests := []struct {
		name string
		base *Config
		file string
		want *Config
	}{
		{
			name: "override empty with empty",
			base: &Config{},
			file: ``,
			want: &Config{},
		},
		{
			name: "override defaults with empty",
			base: &DefaultConfig,
			file: ``,
			want: &Config{
				Values:         []string{"values.yaml"},
				Output:         "values.schema.json",
				Draft:          2020,
				Indent:         4,
				RecursiveNeeds: []string{"Chart.yaml"},
				K8sSchemaURL:   "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
			},
		},
		{
			name: "override every field",
			base: &Config{
				Values:                 []string{"baseInput.yaml"},
				Output:                 "baseOutput.json",
				Draft:                  2020,
				Indent:                 4,
				NoAdditionalProperties: true,
				NoDefaultGlobal:        true,
				Recursive:              true,
				RecursiveNeeds:         []string{"baseNeeds"},
				NoRecursiveNeeds:       true,
				Hidden:                 true,
				NoGitIgnore:            true,
				K8sSchemaURL:           "baseURL",
				K8sSchemaVersion:       "baseVersion",
				UseHelmDocs:            true,
				SchemaRoot: SchemaRoot{
					ID:                   "baseID",
					Ref:                  "baseRef",
					RefReferrer:          ReferrerDir("/tmp"),
					Title:                "baseTitle",
					Description:          "baseDescription",
					AdditionalProperties: boolPtr(true),
				},
			},
			file: `
values: [fileInput.yaml]
output: fileOutput.json
draft: 7
indent: 2
noAdditionalProperties: false
noDefaultGlobal: false
recursive: false
recursiveNeeds: [fileNeeds]
noRecursiveNeeds: false
hidden: false
noGitIgnore: false
k8sSchemaURL: fileURL
k8sSchemaVersion: fileVersion
useHelmDocs: false
schemaRoot:
  id: fileID
  ref: fileRef
  title: fileTitle
  description: fileDescription
  additionalProperties: false
`,
			want: &Config{
				Values:                 []string{"fileInput.yaml"},
				Output:                 "fileOutput.json",
				Draft:                  7,
				Indent:                 2,
				NoAdditionalProperties: false,
				NoDefaultGlobal:        false,
				Recursive:              false,
				RecursiveNeeds:         []string{"fileNeeds"},
				NoRecursiveNeeds:       false,
				Hidden:                 false,
				NoGitIgnore:            false,
				K8sSchemaURL:           "fileURL",
				K8sSchemaVersion:       "fileVersion",
				UseHelmDocs:            false,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Ref:                  "fileRef",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: boolPtr(false),
					RefReferrer:          ReferrerDir("/tmp"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ResetFile(t, tmpFile, []byte(tt.file))

			conf, err := LoadFileConfigOverwrite(tt.base, tmpFile.Name())
			require.NoError(t, err)

			testutil.Equal(t, tt.want, conf, "conf")
		})
	}
}

func TestLoadFileConfigOverwrite_NoFileExists(t *testing.T) {
	conf, err := LoadFileConfigOverwrite(&Config{}, "/file-that-does-not-exist")
	require.NoError(t, err)

	want := &Config{}
	testutil.Equal(t, want, conf, "conf")
}

func TestLoadFileConfigOverwrite_SchemaRootRefReferrerConfigError(t *testing.T) {
	failConfigConfigRefReferrerAbs = true
	defer func() { failConfigConfigRefReferrerAbs = false }()

	configFile := testutil.WriteTempFile(t, "config-*.yaml", []byte(`
schemaRoot:
  ref: foo/bar
`))

	_, err := LoadFileConfigOverwrite(&Config{}, configFile.Name())
	assert.ErrorContains(t, err, "resolve absolute path of config file: ")
}
