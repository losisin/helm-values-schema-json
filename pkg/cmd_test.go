package pkg

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFlagsPass(t *testing.T) {
	var tests = []struct {
		args []string
		conf Config
	}{
		{[]string{"-input", "values.yaml"},
			Config{
				Input:      multiStringFlag{"values.yaml"},
				OutputPath: "values.schema.json",
				Draft:      2020,
				Indent:     4,
				Args:       []string{},
			},
		},

		{[]string{"-input", "values1.yaml values2.yaml", "-indent", "2"},
			Config{
				Input:         multiStringFlag{"values1.yaml values2.yaml"},
				OutputPath:    "values.schema.json",
				Draft:         2020,
				Indent:        2,
				OutputPathSet: false,
				DraftSet:      false,
				IndentSet:     true,
				Args:          []string{},
			},
		},

		{[]string{"-input", "values.yaml", "-output", "my.schema.json", "-draft", "2019", "-indent", "2"},
			Config{
				Input:      multiStringFlag{"values.yaml"},
				OutputPath: "my.schema.json",
				Draft:      2019, Indent: 2,
				OutputPathSet: true,
				DraftSet:      true,
				IndentSet:     true,
				Args:          []string{},
			},
		},

		{[]string{"-input", "values.yaml", "-output", "my.schema.json", "-draft", "2019"},
			Config{
				Input:         multiStringFlag{"values.yaml"},
				OutputPath:    "my.schema.json",
				Draft:         2019,
				Indent:        4,
				OutputPathSet: true,
				DraftSet:      true,
				IndentSet:     false,
				Args:          []string{},
			},
		},

		{[]string{"-input", "values.yaml", "-schemaRoot.id", "http://example.com/schema", "-schemaRoot.ref", "schema/product.json", "-schemaRoot.title", "MySchema", "-schemaRoot.description", "My schema description"},
			Config{
				Input:      multiStringFlag{"values.yaml"},
				OutputPath: "values.schema.json",
				Draft:      2020,
				Indent:     4,
				SchemaRoot: SchemaRoot{
					ID:          "http://example.com/schema",
					Ref:         "schema/product.json",
					Title:       "MySchema",
					Description: "My schema description",
				},
				Args: []string{},
			},
		},

		{[]string{"-bundle=true", "-bundleRoot", "/foo/bar", "-bundleWithoutID=true"},
			Config{
				Indent:          4,
				OutputPath:      "values.schema.json",
				Draft:           2020,
				Bundle:          BoolFlag{set: true, value: true},
				BundleRoot:      "/foo/bar",
				BundleWithoutID: BoolFlag{set: true, value: true},
				Args:            []string{},
			},
		},
		{[]string{"-bundle=false", "-bundleRoot", "", "-bundleWithoutID=false"},
			Config{
				Indent:          4,
				OutputPath:      "values.schema.json",
				Draft:           2020,
				Bundle:          BoolFlag{set: true, value: false},
				BundleRoot:      "",
				BundleWithoutID: BoolFlag{set: true, value: false},
				Args:            []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := ParseFlags("schema", tt.args)
			assert.NoError(t, err)
			assert.Empty(t, output, "output")
			assert.Equal(t, &tt.conf, conf, "conf")
		})
	}
}

func TestParseFlagsUsage(t *testing.T) {
	var usageArgs = []string{"-help", "-h", "--help"}

	for _, arg := range usageArgs {
		t.Run(arg, func(t *testing.T) {
			conf, output, err := ParseFlags("schema", []string{arg})
			if err != flag.ErrHelp {
				t.Errorf("err got %v, want ErrHelp", err)
			}
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if !strings.Contains(output, "Usage of") {
				t.Errorf("output can't find \"Usage of\": %q", output)
			}
		})
	}
}

func TestParseFlagsFail(t *testing.T) {
	var tests = []struct {
		args   []string
		errStr string
	}{
		{[]string{"-input"}, "flag needs an argument"},
		{[]string{"-draft", "foo"}, "invalid value"},
		{[]string{"-foo"}, "flag provided but not defined"},
		{[]string{"-schemaRoot.additionalProperties", "null"}, "invalid boolean value"},
		{[]string{"-bundle", "null"}, "invalid boolean value"},
		{[]string{"-bundleWithoutID", "null"}, "invalid boolean value"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := ParseFlags("schema", tt.args)
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if !strings.Contains(err.Error(), tt.errStr) {
				t.Errorf("err got %q, want to find %q", err.Error(), tt.errStr)
			}
			if !strings.Contains(output, "Usage of") {
				t.Errorf("output got %q", output)
			}
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
  title: Helm Values Schema
  description: Schema for Helm values
  additionalProperties: true
`,
			expectedConf: Config{
				Input:           multiStringFlag{"testdata/empty.yaml", "testdata/meta.yaml"},
				OutputPath:      "values.schema.json",
				Draft:           2020,
				Indent:          2,
				Bundle:          BoolFlag{set: true, value: true},
				BundleRoot:      "./",
				BundleWithoutID: BoolFlag{set: true, value: true},
				SchemaRoot: SchemaRoot{
					ID:                   "https://example.com/schema",
					Title:                "Helm Values Schema",
					Description:          "Schema for Helm values",
					AdditionalProperties: BoolFlag{set: true, value: true},
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
			expectedConf: Config{},
			expectedErr:  true,
		},
		{
			name:          "InvalidYAML",
			configContent: `draft: "invalid"`,
			expectedConf:  Config{},
			expectedErr:   true,
		},
		{
			name:          "MissingFile",
			configContent: "",
			expectedConf:  Config{},
			expectedErr:   false,
		},
		{
			name:          "EmptyConfig",
			configContent: `input: []`,
			expectedConf: Config{
				Input:      multiStringFlag{},
				OutputPath: "",
				Draft:      0,
				Indent:     0,
				SchemaRoot: SchemaRoot{
					ID:                   "",
					Title:                "",
					Description:          "",
					AdditionalProperties: BoolFlag{set: false, value: false},
				},
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configFilePath string
			if tt.configContent != "" {
				tmpFile, err := os.CreateTemp("", "config-*.yaml")
				assert.NoError(t, err)
				defer func() {
					if err := os.Remove(tmpFile.Name()); err != nil && !os.IsNotExist(err) {
						t.Errorf("failed to remove temporary file %s: %v", tmpFile.Name(), err)
					}
				}()
				_, err = tmpFile.Write([]byte(tt.configContent))
				assert.NoError(t, err)
				configFilePath = tmpFile.Name()
			} else {
				configFilePath = "nonexistent.yaml"
			}

			conf, err := LoadConfig(configFilePath)

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

func TestLoadConfig_PermissionDenied(t *testing.T) {
	restrictedDir := "/restricted"
	configFilePath := restrictedDir + "/restricted.yaml"

	readFileFunc = func(filename string) ([]byte, error) {
		return nil, os.ErrPermission
	}
	defer func() { readFileFunc = os.ReadFile }()

	conf, err := LoadConfig(configFilePath)
	assert.ErrorIs(t, err, os.ErrPermission, "Expected permission denied error")
	assert.Nil(t, conf, "Expected config to be nil for permission denied error")
}

func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name           string
		fileConfig     *Config
		flagConfig     *Config
		expectedConfig *Config
	}{
		{
			name: "FlagConfigOverridesFileConfig",
			fileConfig: &Config{
				Input:                  multiStringFlag{"fileInput.yaml"},
				OutputPath:             "fileOutput.json",
				Draft:                  2020,
				Indent:                 4,
				NoAdditionalProperties: BoolFlag{set: true, value: true},
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: BoolFlag{set: true, value: false},
				},
			},
			flagConfig: &Config{
				Input:                  multiStringFlag{"flagInput.yaml"},
				OutputPath:             "flagOutput.json",
				Draft:                  2019,
				Indent:                 2,
				NoAdditionalProperties: BoolFlag{set: true, value: false},
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: BoolFlag{set: true, value: true},
				},
				OutputPathSet: true,
				DraftSet:      true,
				IndentSet:     true,
			},
			expectedConfig: &Config{
				Input:                  multiStringFlag{"flagInput.yaml"},
				OutputPath:             "flagOutput.json",
				Draft:                  2019,
				Indent:                 2,
				NoAdditionalProperties: BoolFlag{set: true, value: false},
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: BoolFlag{set: true, value: true},
				},
			},
		},
		{
			name: "FileConfigDefaultsUsed",
			fileConfig: &Config{
				Input:      multiStringFlag{"fileInput.yaml"},
				OutputPath: "fileOutput.json",
				Draft:      2020,
				Indent:     4,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: BoolFlag{set: true, value: false},
				},
			},
			flagConfig: &Config{},
			expectedConfig: &Config{
				Input:      multiStringFlag{"fileInput.yaml"},
				OutputPath: "fileOutput.json",
				Draft:      2020,
				Indent:     4,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: BoolFlag{set: true, value: false},
				},
			},
		},
		{
			name: "FlagConfigPartialOverride",
			fileConfig: &Config{
				Input:      multiStringFlag{"fileInput.yaml"},
				OutputPath: "fileOutput.json",
				Draft:      2020,
				Indent:     4,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: BoolFlag{set: true, value: false},
				},
			},
			flagConfig: &Config{
				OutputPath:    "flagOutput.json",
				OutputPathSet: true,
			},
			expectedConfig: &Config{
				Input:      multiStringFlag{"fileInput.yaml"},
				OutputPath: "flagOutput.json",
				Draft:      2020,
				Indent:     4,
				SchemaRoot: SchemaRoot{
					ID:                   "fileID",
					Title:                "fileTitle",
					Description:          "fileDescription",
					AdditionalProperties: BoolFlag{set: true, value: false},
				},
			},
		},
		{
			name: "FlagConfigWithEmptyFileConfig",
			fileConfig: &Config{
				Input: multiStringFlag{},
			},
			flagConfig: &Config{
				Input:      multiStringFlag{"flagInput.yaml"},
				OutputPath: "flagOutput.json",
				Draft:      2019,
				Indent:     2,
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: BoolFlag{set: true, value: true},
				},
				OutputPathSet: true,
				DraftSet:      true,
				IndentSet:     true,
			},
			expectedConfig: &Config{
				Input:      multiStringFlag{"flagInput.yaml"},
				OutputPath: "flagOutput.json",
				Draft:      2019,
				Indent:     2,
				SchemaRoot: SchemaRoot{
					ID:                   "flagID",
					Title:                "flagTitle",
					Description:          "flagDescription",
					AdditionalProperties: BoolFlag{set: true, value: true},
				},
			},
		},
		{
			name: "FlagConfigWithBundleOverride",
			fileConfig: &Config{
				Bundle:          BoolFlag{set: true, value: false},
				BundleRoot:      "root/from/file",
				BundleWithoutID: BoolFlag{set: true, value: false},
			},
			flagConfig: &Config{
				Bundle:          BoolFlag{set: true, value: true},
				BundleRoot:      "root/from/flags",
				BundleWithoutID: BoolFlag{set: true, value: true},
			},
			expectedConfig: &Config{
				Bundle:          BoolFlag{set: true, value: true},
				BundleRoot:      "root/from/flags",
				BundleWithoutID: BoolFlag{set: true, value: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergedConfig := MergeConfig(tt.fileConfig, tt.flagConfig)
			assert.Equal(t, tt.expectedConfig, mergedConfig)
		})
	}
}
