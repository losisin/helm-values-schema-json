package pkg

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse flags
func ParseFlags(progname string, args []string) (*Config, string, error) {
	flags := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	conf := &Config{}
	flags.Var(&conf.Input, "input", "Multiple yaml files as inputs (comma-separated)")
	flags.StringVar(&conf.OutputPath, "output", "values.schema.json", "Output file path")
	flags.IntVar(&conf.Draft, "draft", 2020, "Draft version (4, 6, 7, 2019, or 2020)")
	flags.IntVar(&conf.Indent, "indent", 4, "Indentation spaces (even number)")

	// Nested SchemaRoot flags
	flags.StringVar(&conf.SchemaRoot.ID, "schemaRoot.id", "", "JSON schema ID")
	flags.StringVar(&conf.SchemaRoot.Ref, "schemaRoot.ref", "", "JSON schema URI reference")
	flags.StringVar(&conf.SchemaRoot.Title, "schemaRoot.title", "", "JSON schema title")
	flags.StringVar(&conf.SchemaRoot.Description, "schemaRoot.description", "", "JSON schema description")
	flags.Var(&conf.SchemaRoot.AdditionalProperties, "schemaRoot.additionalProperties", "Allow additional properties")

	err := flags.Parse(args)
	if err != nil {
		fmt.Println("Usage: helm schema [options...] <arguments>")
		return nil, buf.String(), err
	}

	// Mark fields as set if they were provided as flags
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "output":
			conf.OutputPathSet = true
		case "draft":
			conf.DraftSet = true
		case "indent":
			conf.IndentSet = true
		}
	})

	conf.Args = flags.Args()
	return conf, buf.String(), nil
}

// LoadConfig loads configuration from a YAML file
var readFileFunc = os.ReadFile

func LoadConfig(configPath string) (*Config, error) {
	data, err := readFileFunc(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return an empty config if the file does not exist
			return &Config{}, nil
		}
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// MergeConfig merges CLI flags into the configuration file values, giving precedence to CLI flags
func MergeConfig(fileConfig, flagConfig *Config) *Config {
	mergedConfig := *fileConfig

	if len(flagConfig.Input) > 0 {
		mergedConfig.Input = flagConfig.Input
	}
	if flagConfig.OutputPathSet || mergedConfig.OutputPath == "" {
		mergedConfig.OutputPath = flagConfig.OutputPath
	}
	if flagConfig.DraftSet || mergedConfig.Draft == 0 {
		mergedConfig.Draft = flagConfig.Draft
	}
	if flagConfig.IndentSet || mergedConfig.Indent == 0 {
		mergedConfig.Indent = flagConfig.Indent
	}
	if flagConfig.SchemaRoot.ID != "" {
		mergedConfig.SchemaRoot.ID = flagConfig.SchemaRoot.ID
	}
	if flagConfig.SchemaRoot.Title != "" {
		mergedConfig.SchemaRoot.Title = flagConfig.SchemaRoot.Title
	}
	if flagConfig.SchemaRoot.Description != "" {
		mergedConfig.SchemaRoot.Description = flagConfig.SchemaRoot.Description
	}
	if flagConfig.SchemaRoot.AdditionalProperties.IsSet() {
		mergedConfig.SchemaRoot.AdditionalProperties = flagConfig.SchemaRoot.AdditionalProperties
	}
	mergedConfig.Args = flagConfig.Args

	return &mergedConfig
}
