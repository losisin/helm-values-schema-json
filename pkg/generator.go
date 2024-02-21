package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Generate JSON schema
func GenerateJsonSchema(config *Config) error {
	// Determine the schema URL based on the draft version
	schemaURL, err := getSchemaURL(config.draft)
	if err != nil {
		return err
	}

	// Initialize a Schema to hold the merged YAML data
	mergedSchema := &Schema{}

	// Iterate over the input YAML files
	for _, filePath := range config.input {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return errors.New("error reading YAML file(s)")
		}

		var node yaml.Node
		if err := yaml.Unmarshal(content, &node); err != nil {
			return errors.New("error unmarshaling YAML")
		}

		if len(node.Content) == 0 {
			continue // Skip empty files
		}

		rootNode := node.Content[0]
		properties := make(map[string]*Schema)
		required := []string{}

		for i := 0; i < len(rootNode.Content); i += 2 {
			keyNode := rootNode.Content[i]
			valNode := rootNode.Content[i+1]
			schema, isRequired := parseNode(keyNode, valNode)
			properties[keyNode.Value] = schema
			if isRequired {
				required = append(required, keyNode.Value)
			}
		}

		// Create a temporary Schema to merge from the nodes
		tempSchema := &Schema{
			Type:                 "object",
			Properties:           properties,
			Required:             required,
			AdditionalProperties: &config.noAdditionalProperties,
		}

		// Merge with existing data
		mergedSchema = mergeSchemas(mergedSchema, tempSchema)
		mergedSchema.Required = uniqueStringAppend(mergedSchema.Required, required...)
	}

	// Convert merged Schema into a JSON Schema compliant map
	jsonSchemaMap, err := convertSchemaToMap(mergedSchema, config.noAdditionalProperties)
	if err != nil {
		return err
	}
	jsonSchemaMap["$schema"] = schemaURL // Include the schema draft version

	// If validation is successful, marshal the schema and save to the file
	jsonBytes, err := json.MarshalIndent(jsonSchemaMap, "", "    ")
	if err != nil {
		return err
	}

	// Write the JSON schema to the output file
	outputPath := config.outputPath
	if err := os.WriteFile(outputPath, jsonBytes, 0644); err != nil {
		return errors.New("error writing schema to file")
	}

	fmt.Println("JSON schema successfully generated")

	return nil
}
