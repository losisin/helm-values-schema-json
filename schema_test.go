package main

import (
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/losisin/go-jsonschema-generator"
)

func TestMultiStringFlagString(t *testing.T) {
	tests := []struct {
		input    multiStringFlag
		expected string
	}{
		{
			input:    multiStringFlag{},
			expected: "",
		},
		{
			input:    multiStringFlag{"value1"},
			expected: "value1",
		},
		{
			input:    multiStringFlag{"value1", "value2", "value3"},
			expected: "value1, value2, value3",
		},
	}

	for i, test := range tests {
		result := test.input.String()
		if result != test.expected {
			t.Errorf("Test case %d: Expected %q, but got %q", i+1, test.expected, result)
		}
	}
}

func TestMultiStringFlagSet(t *testing.T) {
	tests := []struct {
		input     string
		initial   multiStringFlag
		expected  multiStringFlag
		errorFlag bool
	}{
		{
			input:     "value1,value2,value3",
			initial:   multiStringFlag{},
			expected:  multiStringFlag{"value1", "value2", "value3"},
			errorFlag: false,
		},
		{
			input:     "",
			initial:   multiStringFlag{"existingValue"},
			expected:  multiStringFlag{"existingValue"},
			errorFlag: false,
		},
		{
			input:     "value1, value2, value3",
			initial:   multiStringFlag{},
			expected:  multiStringFlag{"value1", "value2", "value3"},
			errorFlag: false,
		},
	}

	for i, test := range tests {
		err := test.initial.Set(test.input)
		if err != nil && !test.errorFlag {
			t.Errorf("Test case %d: Expected no error, but got: %v", i+1, err)
		} else if err == nil && test.errorFlag {
			t.Errorf("Test case %d: Expected an error, but got none", i+1)
		}
	}
}

func TestReadAndUnmarshalYAML(t *testing.T) {
	t.Run("Valid YAML", func(t *testing.T) {
		yamlContent := []byte("key1: value1\nkey2: value2\n")
		yamlFilePath := "valid.yaml"

		err := os.WriteFile(yamlFilePath, yamlContent, 0644)
		if err != nil {
			t.Fatalf("Error creating a temporary YAML file: %v", err)
		}
		defer os.Remove(yamlFilePath)

		var target map[string]interface{}
		err = readAndUnmarshalYAML(yamlFilePath, &target)

		if err != nil {
			t.Errorf("Error reading and unmarshaling valid YAML: %v", err)
			return
		}

		if len(target) != 2 {
			t.Errorf("Expected target map length to be 2, but got %d", len(target))
		}

		if target["key1"] != "value1" {
			t.Errorf("target map should contain key1 with value 'value1'")
		}

		if target["key2"] != "value2" {
			t.Errorf("target map should contain key2 with value 'value2'")
		}
	})

	t.Run("File Missing", func(t *testing.T) {
		// YAML file is assumed to be missing
		missingFilePath := "missing.yaml"

		var target map[string]interface{}
		err := readAndUnmarshalYAML(missingFilePath, &target)

		if err == nil {
			t.Errorf("Expected an error when the file is missing, but got nil")
		}
	})
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		a, b, expected map[string]interface{}
	}{
		{
			a:        map[string]interface{}{"key1": "value1"},
			b:        map[string]interface{}{"key2": "value2"},
			expected: map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			a:        map[string]interface{}{"key1": "value1"},
			b:        map[string]interface{}{"key1": "value2"},
			expected: map[string]interface{}{"key1": "value2"},
		},
		{
			a: map[string]interface{}{
				"key1": map[string]interface{}{"subkey1": "value1"},
			},
			b: map[string]interface{}{
				"key1": map[string]interface{}{"subkey2": "value2"},
			},
			expected: map[string]interface{}{
				"key1": map[string]interface{}{"subkey1": "value1", "subkey2": "value2"},
			},
		},
		{
			a: map[string]interface{}{
				"key1": map[string]interface{}{
					"subkey1": "value1",
				},
			},
			b: map[string]interface{}{
				"key1": map[string]interface{}{
					"subkey2": "value2",
				},
			},
			expected: map[string]interface{}{
				"key1": map[string]interface{}{
					"subkey1": "value1",
					"subkey2": "value2",
				},
			},
		},
	}

	for i, test := range tests {
		result := mergeMaps(test.a, test.b)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("Test case %d failed. Expected: %v, Got: %v", i+1, test.expected, result)
		}
	}
}

func TestPrintMap(t *testing.T) {
	tmpFile := "test_output.json"
	defer os.Remove(tmpFile)

	var yamlData map[string]interface{}

	// Test successful data read and schema creation
	err := readAndUnmarshalYAML("testdata/values.yaml", &yamlData)
	if err != nil {
		t.Fatalf("Failed to mock YAML data: %v", err)
	}
	data := jsonschema.NewDocument("")
	data.ReadDeep(&yamlData)

	cases := []struct {
		data        *jsonschema.Document
		tmpFile     string
		expectError bool
	}{
		{data, tmpFile, false},
		{data, "", true},
		{nil, tmpFile, true},
	}

	for _, c := range cases {
		t.Run("PrintMap", func(t *testing.T) {
			err := printMap(c.data, c.tmpFile)
			switch {
			case err == nil && c.expectError:
				t.Fatalf("Expected an error, but printMap succeeded")
			case err != nil && !c.expectError:
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParseFlagsCorrect(t *testing.T) {
	var tests = []struct {
		args []string
		conf Config
	}{
		{[]string{"-input", "testdata/values.yaml"},
			Config{input: multiStringFlag{"testdata/values.yaml"}, outputPath: "values.schema.json", draft: 2020, args: []string{}}},

		{[]string{"-input", "values1.yaml testdata/values.yaml"},
			Config{input: multiStringFlag{"values1.yaml testdata/values.yaml"}, outputPath: "values.schema.json", draft: 2020, args: []string{}}},

		{[]string{"-input", "testdata/values.yaml", "-output", "my.schema.json", "-draft", "2019"},
			Config{input: multiStringFlag{"testdata/values.yaml"}, outputPath: "my.schema.json", draft: 2019, args: []string{}}},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := parseFlags("prog", tt.args)
			if err != nil {
				t.Errorf("err got %v, want nil", err)
			}
			if output != "" {
				t.Errorf("output got %q, want empty", output)
			}
			if !reflect.DeepEqual(*conf, tt.conf) {
				t.Errorf("conf got %+v, want %+v", *conf, tt.conf)
			}
		})
	}
}

func TestParseFlagsUsage(t *testing.T) {
	var usageArgs = []string{"-help", "-h", "--help"}

	for _, arg := range usageArgs {
		t.Run(arg, func(t *testing.T) {
			conf, output, err := parseFlags("prog", []string{arg})
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

func TestParseFlagsError(t *testing.T) {
	var tests = []struct {
		args   []string
		errstr string
	}{
		{[]string{"-input"}, "flag needs an argument"},
		{[]string{"-draft", "foo"}, "invalid value"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := parseFlags("prog", tt.args)
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if !strings.Contains(err.Error(), tt.errstr) {
				t.Errorf("err got %q, want to find %q", err.Error(), tt.errstr)
			}
			if !strings.Contains(output, "Usage of") {
				t.Errorf("output got %q", output)
			}
		})
	}
}
