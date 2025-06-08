// Package yamlfile implements a [koanf.Parser] that parses YAML as a specific
// type with a custom output field names.
package yamlfile

import (
	"errors"

	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

// YAML implements a YAML [koanf.Provider].
type YAML[T any] struct {
	Defaults T
	File     *file.File
	Tag      string
}

// ensure it implements the interface
var _ koanf.Provider = &YAML[int]{}

// Provider returns a YAML [koanf.Provider].
func Provider[T any](defaults T, path, destTag string) *YAML[T] {
	return &YAML[T]{
		Defaults: defaults,
		File:     file.Provider(path),
		Tag:      destTag,
	}
}

// ReadBytes is not supported by the YAML provider.
func (*YAML[T]) ReadBytes() ([]byte, error) {
	return nil, errors.New("yamlfile provider does not support this method")
}

// Read implements [koanf.Provider].
func (p *YAML[T]) Read() (map[string]any, error) {
	b, err := p.File.ReadBytes()
	if err != nil {
		return nil, err
	}
	out := p.Defaults
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return structs.Provider(out, p.Tag).Read()
}
