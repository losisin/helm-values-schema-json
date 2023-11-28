package pkg

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGenerateJsonSchemaPass(t *testing.T) {
	var tests = []struct {
		conf        Config
		expectedUrl string
	}{
		{Config{input: multiStringFlag{"../testdata/values_1.yaml", "../testdata/values_2.yaml"}, draft: 2020, outputPath: "2020.schema.json", args: []string{}}, "https://json-schema.org/draft/2020-12/schema"},
		{Config{input: multiStringFlag{"../testdata/values_1.yaml"}, draft: 2020, outputPath: "2020.schema.json", args: []string{}}, "https://json-schema.org/draft/2020-12/schema"},
		{Config{input: multiStringFlag{"../testdata/values_1.yaml"}, draft: 2019, outputPath: "2019.schema.json", args: []string{}}, "https://json-schema.org/draft/2019-09/schema"},
		{Config{input: multiStringFlag{"../testdata/values_1.yaml"}, draft: 7, outputPath: "7.schema.json", args: []string{}}, "http://json-schema.org/draft-07/schema#"},
		{Config{input: multiStringFlag{"../testdata/values_1.yaml"}, draft: 6, outputPath: "6.schema.json", args: []string{}}, "http://json-schema.org/draft-06/schema#"},
		{Config{input: multiStringFlag{"../testdata/values_1.yaml"}, draft: 4, outputPath: "4.schema.json", args: []string{}}, "http://json-schema.org/draft-04/schema#"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.conf), func(t *testing.T) {
			conf := &tt.conf
			err := GenerateJsonSchema(conf)
			if err != nil {
				t.Fatalf("generateJsonSchema() failed: %v", err)
			}

			_, err = os.Stat(conf.outputPath)
			if os.IsNotExist(err) {
				t.Errorf("Expected file '%q' to be created, but it doesn't exist", conf.outputPath)
			}

			outputJson, err := os.ReadFile(conf.outputPath)
			if err != nil {
				t.Errorf("Error reading file '%q': %v", conf.outputPath, err)
			}

			actualURL := string(outputJson)
			if !strings.Contains(actualURL, tt.expectedUrl) {
				t.Errorf("Schema URL does not match. Got: %s, Expected: %s", actualURL, tt.expectedUrl)
			}

			os.Remove(conf.outputPath)
		})
		t.Run(fmt.Sprintf("%v", tt.conf), func(t *testing.T) {
			conf := &tt.conf
			err := GenerateJsonSchema(conf)
			if err != nil {
				t.Fatalf("generateJsonSchema() failed: %v", err)
			}

			outputJson, err := os.ReadFile(conf.outputPath)
			if err != nil {
				t.Errorf("Error reading file '%q': %v", conf.outputPath, err)
			}

			actualURL := string(outputJson)
			if !strings.Contains(actualURL, tt.expectedUrl) {
				t.Errorf("Schema URL does not match. Got: %s, Expected: %s", actualURL, tt.expectedUrl)
			}
			os.Remove(conf.outputPath)
		})
	}
}

func TestGenerateJsonSchemaFail(t *testing.T) {
	testCases := []struct {
		config      *Config
		expectedErr string
	}{
		{
			config:      &Config{},
			expectedErr: "input flag is required. Please provide input yaml files using the -input flag",
		},
		{
			config: &Config{
				input: multiStringFlag{"values.yaml"},
				draft: 5,
			},
			expectedErr: "invalid draft version. Please use one of: 4, 6, 7, 2019, 2020",
		},
		{
			config: &Config{
				input: multiStringFlag{"fake.yaml"},
				draft: 2019,
			},
			expectedErr: "error reading YAML file(s)",
		},
	}

	for _, testCase := range testCases {
		err := GenerateJsonSchema(testCase.config)
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if err.Error() != testCase.expectedErr {
			t.Errorf("Expected error: %s, got: %s", testCase.expectedErr, err.Error())
		}
	}
}
