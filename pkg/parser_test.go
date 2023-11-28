package pkg

import (
	"os"
	"reflect"
	"testing"
)

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
