package pkg

import "strings"

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
