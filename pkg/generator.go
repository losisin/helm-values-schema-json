package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Generate JSON schema
func GenerateJsonSchema(ctx context.Context, config *Config) error {
	// Check if the values flag is set
	if len(config.Values) == 0 {
		return errors.New("values flag is required")
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

	// Iterate over the input YAML files
	for _, filePath := range config.Values {
		filePathAbs, err := filepath.Abs(filepath.FromSlash(filePath))
		if err != nil {
			return fmt.Errorf("%s: get absolute path: %w", filePath, err)
		}

		content, err := os.ReadFile(filepath.Clean(filePath))
		if err != nil {
			return errors.New("error reading YAML file(s)")
		}

		// Change Window's CRLF to LF line endings
		// as the YAML parser incorrectly includes them in comments otherwise
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))

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
			schema, isRequired, err := parseNode(NewPtr(keyNode.Value), keyNode, valNode, config.UseHelmDocs)
			if err != nil {
				return fmt.Errorf("parse schema: %w", err)
			}

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
		}

		tempSchema.SetReferrer(ReferrerDir(filepath.Dir(filePathAbs)))
		// Set root $ref after updating the referrer on all other $refs
		if config.SchemaRoot.Ref != "" {
			tempSchema.Ref = config.SchemaRoot.Ref
			tempSchema.RefReferrer = config.SchemaRoot.RefReferrer
		}

		// Apply "$ref: $k8s/..." transformation
		if err := updateRefK8sAlias(tempSchema, config.K8sSchemaURL, config.K8sSchemaVersion); err != nil {
			return fmt.Errorf("%s: %w", filePath, err)
		}

		// Merge with existing data
		mergedSchema = mergeSchemas(mergedSchema, tempSchema)
		mergedSchema.Required = uniqueStringAppend(mergedSchema.Required, required...)
	}

	if config.Bundle {
		if err := Bundle(ctx, mergedSchema, config.Output, config.BundleRoot, config.BundleWithoutID); err != nil {
			return err
		}
	}

	// Ensure merged Schema is JSON Schema compliant
	if err := ensureCompliant(mergedSchema, config.NoAdditionalProperties, config.Draft); err != nil {
		return err
	}
	mergedSchema.Schema = schemaURL // Include the schema draft version
	mergedSchema.Type = "object"

	if config.SchemaRoot.AdditionalProperties != nil {
		mergedSchema.AdditionalProperties = SchemaBool(*config.SchemaRoot.AdditionalProperties)
	} else if config.NoAdditionalProperties {
		mergedSchema.AdditionalProperties = &SchemaFalse
	}

	return WriteOutput(ctx, mergedSchema, filepath.FromSlash(config.Output), indentString)
}

func WriteOutput(ctx context.Context, mergedSchema *Schema, outputPath, indent string) error {
	logger := LoggerFromContext(ctx)

	// If validation is successful, marshal the schema and save to the file
	jsonBytes, err := json.MarshalIndent(mergedSchema, "", indent)
	if err != nil {
		return err
	}
	jsonBytes = append(jsonBytes, '\n')

	// Write the JSON schema to the output file
	if err := os.WriteFile(outputPath, jsonBytes, 0600); err != nil {
		return errors.New("error writing schema to file")
	}

	logger.Log("JSON schema successfully generated")
	return nil
}
