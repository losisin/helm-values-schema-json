package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Print the merged map to a file as JSON schema
func writeMap(data map[string]interface{}, outputPath string) error {
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
