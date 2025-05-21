package pkg

import (
	"errors"
	"fmt"
	"strings"
)

// SchemaRoot struct defines root object of schema
type SchemaRoot struct {
	ID                   string   `yaml:"id"`
	Ref                  string   `yaml:"ref"`
	Title                string   `yaml:"title"`
	Description          string   `yaml:"description"`
	AdditionalProperties BoolFlag `yaml:"additionalProperties"`
}

// Save values of parsed flags in Config
type Config struct {
	Input                  multiStringFlag `yaml:"input"`
	OutputPath             string          `yaml:"output"`
	Draft                  int             `yaml:"draft"`
	Indent                 int             `yaml:"indent"`
	NoAdditionalProperties BoolFlag        `yaml:"noAdditionalProperties"`
	Bundle                 BoolFlag        `yaml:"bundle"`
	BundleRoot             string          `yaml:"bundleRoot"`

	SchemaRoot SchemaRoot `yaml:"schemaRoot"`

	Args []string `yaml:"-"`

	OutputPathSet bool
	DraftSet      bool
	IndentSet     bool
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
	switch value {
	case "true":
		b.value = true
	case "false":
		b.value = false
	default:
		return errors.New("invalid boolean value")
	}
	b.set = true
	return nil
}

func (b *BoolFlag) IsSet() bool {
	return b.set
}

func (b *BoolFlag) Value() bool {
	return b.value
}

func (b *BoolFlag) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var boolValue bool
	if err := unmarshal(&boolValue); err != nil {
		return err
	}
	b.value = boolValue
	b.set = true
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
