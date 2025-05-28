package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Generate JSON schema
func GenerateJsonSchema(config *Config) error {
	// Check if the input flag is set
	if len(config.Input) == 0 {
		return errors.New("input flag is required")
	}

	// Determine the schema URL based on the draft version
	schemaURL, err := getSchemaURL(config.Draft)
	if err != nil {
		return err
	}

	// Determine the indentation string based on the number of spaces
	if config.Indent <= 0 {
		return errors.New("indentation must be a positive number")
	}
	if config.Indent%2 != 0 {
		return errors.New("indentation must be an even number")
	}
	indentString := strings.Repeat(" ", config.Indent)

	// Initialize a Schema to hold the merged YAML data
	mergedSchema := &Schema{}

	bundleRoot := config.BundleRoot
	if bundleRoot == "" {
		bundleRoot = "."
	}
	root, err := os.OpenRoot(bundleRoot)
	if err != nil {
		return fmt.Errorf("open bundle root: %w", err)
	}
	defer closeIgnoreError(root)

	// Iterate over the input YAML files
	for _, filePath := range config.Input {
		content, err := os.ReadFile(filepath.Clean(filePath))
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

			// Exclude hidden nodes
			if schema != nil && !schema.Hidden {
				if schema.SkipProperties && schema.Type == "object" {
					schema.Properties = nil
				}
				properties[keyNode.Value] = schema
				if isRequired {
					required = append(required, keyNode.Value)
				}
			}
		}

		// Create a temporary Schema to merge from the nodes
		tempSchema := &Schema{
			Type:        "object",
			Properties:  properties,
			Required:    required,
			Title:       config.SchemaRoot.Title,
			Description: config.SchemaRoot.Description,
			ID:          config.SchemaRoot.ID,
			Ref:         config.SchemaRoot.Ref,
		}

		if config.Bundle.Value() {
			ctx := context.Background()
			basePath, err := filepath.Rel(bundleRoot, filepath.Dir(filePath))
			if err != nil {
				return fmt.Errorf("get relative path from bundle root to file %q: %w", filePath, err)
			}
			loader := NewDefaultLoader(http.DefaultClient, root, basePath)
			if err := BundleSchema(ctx, loader, tempSchema); err != nil {
				return fmt.Errorf("bundle schemas on %q: %w", filePath, err)
			}
		}

		// Merge with existing data
		mergedSchema = mergeSchemas(mergedSchema, tempSchema)
		mergedSchema.Required = uniqueStringAppend(mergedSchema.Required, required...)
	}

	if config.Bundle.Value() && config.BundleWithoutID.Value() {
		if err := BundleRemoveIDs(mergedSchema); err != nil {
			return fmt.Errorf("remove bundled $id: %w", err)
		}
	}

	// Convert merged Schema into a JSON Schema compliant map
	jsonSchemaMap, err := convertSchemaToMap(mergedSchema, config.NoAdditionalProperties.value)
	if err != nil {
		return err
	}
	jsonSchemaMap["$schema"] = schemaURL // Include the schema draft version
	jsonSchemaMap["type"] = "object"

	if config.SchemaRoot.AdditionalProperties.IsSet() {
		jsonSchemaMap["additionalProperties"] = config.SchemaRoot.AdditionalProperties.Value()
	} else if config.NoAdditionalProperties.value {
		jsonSchemaMap["additionalProperties"] = false
	}

	// If validation is successful, marshal the schema and save to the file
	jsonBytes, err := json.MarshalIndent(jsonSchemaMap, "", indentString)
	if err != nil {
		return err
	}
	jsonBytes = append(jsonBytes, '\n')

	// Write the JSON schema to the output file
	outputPath := config.OutputPath
	if err := os.WriteFile(outputPath, jsonBytes, 0600); err != nil {
		return errors.New("error writing schema to file")
	}

	fmt.Println("JSON schema successfully generated")

	return nil
}
