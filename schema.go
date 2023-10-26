package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/losisin/go-jsonschema-generator"
	"gopkg.in/yaml.v3"
)

// Define a custom flag type to accept multiple yamlFiles
type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiStringFlag) Set(value string) error {
	values := strings.Split(value, ",")
	for _, v := range values {
		*m = append(*m, v)
	}
	return nil
}

func readAndUnmarshalYAML(filePath string, target interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func printMap(data *jsonschema.Document, outputPath string) error {
	// Use YAML marshaling to format the map
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	// If outputPath is provided, create or overwrite the specified file
	if outputPath != "" {
		// Create or overwrite the file
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Write the new data to the output file
		_, err = file.Write(jsonData)
		if err != nil {
			return err
		}
		fmt.Printf("Merged data saved to %s\n", outputPath)
	}

	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: helm schema [-input STR] [-draft INT] [-output STR]")
	flag.PrintDefaults()
}

func main() {
	// Define the custom flag for yamlFiles and set its default value
	var yamlFiles multiStringFlag
	flag.Var(&yamlFiles, "input", "Multiple yamlFiles as inputs (comma-separated)")

	// Define the flag to specify the schema url
	draft := flag.Int("draft", 2020, "Draft version (4, 6, 7, 2019, or 2020)")

	// Define the flag to specify the output file
	var outputPath string
	flag.StringVar(&outputPath, "output", "values.schema.json", "Output file path")

	flag.Usage = usage
	flag.Parse()

	// Check if the input flag is set
	if len(yamlFiles) == 0 {
		fmt.Println("Input flag is required. Please provide input yaml files using the -i flag.")
		usage()
		return
	}

	var schemaUrl string
	switch *draft {
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
		fmt.Fprintln(os.Stderr, "Invalid draft version. Please use one of: 4, 6, 7, 2019, 2020.")
		os.Exit(1)
	}

	// Declare a map to hold the merged YAML data
	mergedMap := make(map[string]interface{})

	// Iterate over the input YAML files
	for _, filePath := range yamlFiles {
		// Read and unmarshal each YAML file
		var currentMap map[string]interface{}
		if err := readAndUnmarshalYAML(filePath, &currentMap); err != nil {
			fmt.Printf("Error reading %s: %v\n", filePath, err)
			continue
		}

		// Merge the current YAML data with the mergedMap
		mergedMap = mergeMaps(mergedMap, currentMap)
		// fmt.Println(mergedMap)
	}

	// Print or save the merged map
	d := jsonschema.NewDocument(schemaUrl)
	d.ReadDeep(&mergedMap)

	err := printMap(d, outputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
