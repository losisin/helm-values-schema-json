package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

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

func TestMainWithDynamicFlags(t *testing.T) {
	testCases := []struct {
		draftVersion   int
		expectedString string
	}{
		{4, `"$schema": "http://json-schema.org/draft-04/schema#",`},
		{6, `"$schema": "http://json-schema.org/draft-06/schema#",`},
		{7, `"$schema": "http://json-schema.org/draft-07/schema#",`},
		{2019, `"$schema": "https://json-schema.org/draft/2019-09/schema",`},
		{2020, `"$schema": "https://json-schema.org/draft/2020-12/schema",`},
	}

	for _, testCase := range testCases {
		// Run schema.go with the specified draft version flag.
		cmd := exec.Command("go", "run", "schema.go", "--input=values.yaml", fmt.Sprintf("--draft=%d", testCase.draftVersion))

		// Capture the command's output (stdout and stderr).
		cmdOutput, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command execution failed: %v\nOutput:\n%s", err, cmdOutput)
		}

		// Check the exit status to ensure it ran successfully.
		if cmd.ProcessState.ExitCode() != 0 {
			t.Fatalf("Command execution returned a non-zero exit code: %d\nOutput:\n%s", cmd.ProcessState.ExitCode(), cmdOutput)
		}

		// Now, read and inspect the contents of values.schema.json.
		file, err := os.Open("values.schema.json")
		if err != nil {
			t.Fatalf("Failed to open values.schema.json: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNumber := 1

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if lineNumber == 2 {
				if line != testCase.expectedString {
					t.Errorf("Expected line 2 to be:\n%s\nGot:\n%s", testCase.expectedString, line)
				}
				break
			}
			lineNumber++
		}

		if err := scanner.Err(); err != nil {
			t.Fatalf("Error reading values.schema.json: %v", err)
		}
	}
}
