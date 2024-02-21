package pkg

import "strings"

// Save values of parsed flags in Config
type Config struct {
	input                  multiStringFlag
	outputPath             string
	draft                  int
	noAdditionalProperties bool

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

func uniqueStringAppend(dest []string, src ...string) []string {
	existingItems := make(map[string]bool)
	for _, item := range dest {
		existingItems[item] = true
	}

	for _, item := range src {
		if _, exists := existingItems[item]; !exists {
			dest = append(dest, item)
			existingItems[item] = true
		}
	}

	return dest
}
