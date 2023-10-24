package main

import (
	"fmt"
	"os"
	"testing"
)

func TestReadAndUnmarshalYAML(t *testing.T) {
	// Create a temporary YAML file for testing
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
	fmt.Println(err)
}

func TestMergeMaps(t *testing.T) {
	mapA := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	mapB := map[string]interface{}{
		"key2": "newvalue2",
		"key3": "value3",
	}

	merged := mergeMaps(mapA, mapB)

	if len(merged) != 3 {
		t.Errorf("Expected merged map length to be 3, but got %d", len(merged))
	}

	if merged["key1"] != "value1" {
		t.Errorf("Merged map should contain key1 from mapA")
	}

	if merged["key2"] != "newvalue2" {
		t.Errorf("Merged map should contain key2 from mapB")
	}

	if merged["key3"] != "value3" {
		t.Errorf("Merged map should contain key3 from mapB")
	}
}
