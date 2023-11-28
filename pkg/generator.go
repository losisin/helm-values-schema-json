package pkg

import (
	"errors"
	"fmt"
)

// Generate JSON schema
func GenerateJsonSchema(config *Config) error {
	// Check if the input flag is set
	if len(config.input) == 0 {
		return errors.New("input flag is required. Please provide input yaml files using the -input flag")
	}

	var schemaUrl string
	switch config.draft {
	case 4:
		schemaUrl = "http://json-schema.org/draft-04/schema#"
	case 6:
		schemaUrl = "http://json-schema.org/draft-06/schema#"
	case 7:
		schemaUrl = "http://json-schema.org/draft-07/schema#"
	case 2019:
		schemaUrl = "https://json-schema.org/draft/2019-09/schema"
	case 2020:
		schemaUrl = "https://json-schema.org/draft/2020-12/schema"
	default:
		return errors.New("invalid draft version. Please use one of: 4, 6, 7, 2019, 2020")
	}

	// Declare a map to hold the merged YAML data
	mergedMap := make(map[string]interface{})

	// Iterate over the input YAML files
	for _, filePath := range config.input {
		var currentMap map[string]interface{}
		if err := readAndUnmarshalYAML(filePath, &currentMap); err != nil {
			return errors.New("error reading YAML file(s)")

		}

		// Merge the current YAML data with the mergedMap
		mergedMap = mergeMaps(mergedMap, currentMap)
	}

	// Print the merged map
	d := NewDocument(schemaUrl)
	d.Read(&mergedMap)

	err := printMap(d, config.outputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	return nil
}
