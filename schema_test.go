package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/losisin/go-jsonschema-generator"
)

func TestMultiStringFlagString(t *testing.T) {
	// Initialize a multiStringFlag instance.
	flag := multiStringFlag{"value1", "value2", "value3"}

	// Call the String method.
	result := flag.String()

	// Define the expected result.
	expected := "value1, value2, value3"

	// Check if the result matches the expected value.
	if result != expected {
		t.Errorf("String() method returned %s, expected %s", result, expected)
	}
}

func TestMultiStringFlagSet(t *testing.T) {
	// Initialize a multiStringFlag instance.
	flag := multiStringFlag{}

	// Call the Set method with a sample value.
	value := "value1,value2,value3"
	err := flag.Set(value)

	// Check for any errors returned by the Set method.
	if err != nil {
		t.Errorf("Set() method returned an error: %v", err)
	}

	// Define the expected flag value after calling Set.
	expected := multiStringFlag{"value1", "value2", "value3"}

	// Check if the flag value matches the expected value.
	if !reflect.DeepEqual(flag, expected) {
		t.Errorf("Set() method set the flag to %v, expected %v", flag, expected)
	}
}

func TestReadAndUnmarshalYAML(t *testing.T) {
	yamlContent := []byte("key1: value1\nkey2: value2\n")
	yamlFilePath := "test.yaml"
	err := os.WriteFile(yamlFilePath, yamlContent, 0644)
	if err != nil {
		t.Fatalf("Error creating a temporary YAML file: %v", err)
	}
	defer os.Remove(yamlFilePath)
	var target map[string]interface{}
	err = readAndUnmarshalYAML(yamlFilePath, &target)
	if err != nil {
		t.Errorf("Error reading and unmarshaling YAML: %v", err)
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

	err := readAndUnmarshalYAML("values.yaml", &yamlData)
	if err != nil {
		t.Fatalf("Failed to mock YAML data: %v", err)
	}
	s := &jsonschema.Document{}
	s.Read(yamlData)

	err = printMap(s, tmpFile)
	if err != nil {
		t.Fatalf("printMap failed: %v", err)
	}

	fileData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read temporary file: %v", err)
	}

	var outputJSON interface{}
	if err := json.Unmarshal(fileData, &outputJSON); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
}
