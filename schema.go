package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/losisin/go-jsonschema-generator"
	"gopkg.in/yaml.v3"
)

// Save values of parsed flags in Config
type Config struct {
	input      multiStringFlag
	outputPath string
	draft      int

	args []string
}

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

// Read and unmarshal YAML file
func readAndUnmarshalYAML(filePath string, target interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

// Merge all YAML files into a single map
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

// Print the merged map to a file as JSON schema
func printMap(data *jsonschema.Document, outputPath string) error {
	if data == nil {
		return errors.New("data is nil")
	}
	// Use YAML marshaling to format the map
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

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

	return nil
}

// Parse flags
func parseFlags(progname string, args []string) (config *Config, output string, err error) {
	flags := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	var conf Config
	flags.Var(&conf.input, "input", "Multiple yaml files as inputs (comma-separated)")
	flags.StringVar(&conf.outputPath, "output", "values.schema.json", "Output file path")
	flags.IntVar(&conf.draft, "draft", 2020, "Draft version (4, 6, 7, 2019, or 2020)")

	err = flags.Parse(args)
	if err != nil {
		fmt.Println("Usage: helm schema [-input STR] [-draft INT] [-output STR]")
		return nil, buf.String(), err
	}

	conf.args = flags.Args()
	return &conf, buf.String(), nil
}

// Generate JSON schema
func generateJsonSchema(config *Config) error {
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
	d := jsonschema.NewDocument(schemaUrl)
	d.ReadDeep(&mergedMap)

	err := printMap(d, config.outputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	return nil
}

func main() {
	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Println(output)
		return
	} else if err != nil {
		fmt.Println("Error:", output)
		return
	}

	err = generateJsonSchema(conf)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
