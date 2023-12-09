package pkg

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateJsonSchema(t *testing.T) {
	config := &Config{
		input: []string{
			"../testdata/full.yaml",
			"../testdata/empty.yaml",
		},
		outputPath: "../testdata/output.json",
		draft:      2020,
	}

	err := GenerateJsonSchema(config)
	assert.NoError(t, err)

	generatedBytes, err := os.ReadFile(config.outputPath)
	assert.NoError(t, err)

	templateBytes, err := os.ReadFile("../testdata/full.schema.json")
	assert.NoError(t, err)

	var generatedSchema, templateSchema map[string]interface{}
	err = json.Unmarshal(generatedBytes, &generatedSchema)
	assert.NoError(t, err)
	err = json.Unmarshal(templateBytes, &templateSchema)
	assert.NoError(t, err)

	assert.Equal(t, templateSchema, generatedSchema, "Generated JSON schema does not match the template")

	os.Remove(config.outputPath)
}
