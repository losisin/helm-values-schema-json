package pkg

import (
	"errors"
	"fmt"
	"strings"
)

// SchemaRoot struct defines root object of schema
type SchemaRoot struct {
	ID                   string
	Title                string
	Description          string
	AdditionalProperties BoolFlag
}

// Save values of parsed flags in Config
type Config struct {
	input      multiStringFlag
	outputPath string
	draft      int
	indent     int

	SchemaRoot SchemaRoot

	args []string
}

// Define a custom flag type to accept multiple yaml files
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

// Custom BoolFlag type that tracks if it was explicitly set
type BoolFlag struct {
	set   bool
	value bool
}

func (b *BoolFlag) String() string {
	if b.set {
		return fmt.Sprintf("%t", b.value)
	}
	return "not set"
}

func (b *BoolFlag) Set(value string) error {
	if value == "true" {
		b.value = true
	} else if value == "false" {
		b.value = false
	} else {
		return errors.New("invalid boolean value")
	}
	b.set = true
	return nil
}

// Accessor method to check if the flag was explicitly set
func (b *BoolFlag) IsSet() bool {
	return b.set
}

// Accessor method to get the value of the flag
func (b *BoolFlag) Value() bool {
	return b.value
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
