package pkg

import (
	"os"
	"testing"
)

func TestPrintMap(t *testing.T) {
	tmpFile := "test_output.json"
	defer os.Remove(tmpFile)

	var yamlData map[string]interface{}

	err := readAndUnmarshalYAML("../testdata/values_1.yaml", &yamlData)
	if err != nil {
		t.Fatalf("Failed to mock YAML data: %v", err)
	}
	data := NewDocument("")
	data.ReadDeep(&yamlData)

	tests := []struct {
		data        *Document
		tmpFile     string
		expectError bool
	}{
		{data, tmpFile, false},
		{data, "", true},
		{nil, tmpFile, true},
	}

	for _, tt := range tests {
		t.Run("PrintMap", func(t *testing.T) {
			err := printMap(tt.data, tt.tmpFile)
			switch {
			case err == nil && tt.expectError:
				t.Fatalf("Expected an error, but printMap succeeded")
			case err != nil && !tt.expectError:
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}
