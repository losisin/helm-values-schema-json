package pkg

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
	flags.Var(&conf.NoAdditionalProperties, "noAdditionalProperties", "Default additionalProperties to false for all objects in the schema")

	flags.Var(&conf.Bundle, "bundle", "Bundle referenced ($ref) subschemas into a single file inside $defs")
	flags.Var(&conf.BundleWithoutID, "bundleWithoutID", "Bundle without using $id to reference bundled schemas, which improves compatibility with e.g the VS Code JSON extension")
	flags.StringVar(&conf.BundleRoot, "bundleRoot", "", "Root directory to allow local referenced files to be loaded from (default current working directory)")

	flags.StringVar(&conf.K8sSchemaURL, "k8sSchemaURL", "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/", "URL template used in $ref: $k8s/... alias")
	flags.StringVar(&conf.K8sSchemaVersion, "k8sSchemaVersion", "", "Version used in the --k8sSchemaURL template for $ref: $k8s/... alias")

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

	if flags.NArg() >= 1 && flags.Arg(0) == "__complete" {
		return nil, "", ErrCompletionRequested{FlagSet: flags}
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
		case "k8sSchemaURL":
			conf.K8sSchemaURLSet = true
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

	if flagConfig.NoAdditionalProperties.IsSet() {
		mergedConfig.NoAdditionalProperties = flagConfig.NoAdditionalProperties
	}
	if flagConfig.Bundle.IsSet() {
		mergedConfig.Bundle = flagConfig.Bundle
	}
	if flagConfig.BundleWithoutID.IsSet() {
		mergedConfig.BundleWithoutID = flagConfig.BundleWithoutID
	}
	if flagConfig.BundleRoot != "" {
		mergedConfig.BundleRoot = flagConfig.BundleRoot
	}
	if flagConfig.K8sSchemaURLSet || mergedConfig.K8sSchemaURL == "" {
		mergedConfig.K8sSchemaURL = flagConfig.K8sSchemaURL
	}
	if flagConfig.K8sSchemaVersion != "" {
		mergedConfig.K8sSchemaVersion = flagConfig.K8sSchemaVersion
	}
	if flagConfig.SchemaRoot.ID != "" {
		mergedConfig.SchemaRoot.ID = flagConfig.SchemaRoot.ID
	}
	if flagConfig.SchemaRoot.Ref != "" {
		mergedConfig.SchemaRoot.Ref = flagConfig.SchemaRoot.Ref
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

type ErrCompletionRequested struct {
	FlagSet *flag.FlagSet
}

// Error implements [error].
func (ErrCompletionRequested) Error() string {
	return "completion requested"
}

func (err ErrCompletionRequested) Fprint(writer io.Writer) {
	args := err.FlagSet.Args()
	if len(args) > 1 && args[1] == "__complete" {
		args = args[2:]
	}
	if len(args) >= 2 {
		prevArg := args[len(args)-2]
		currArg := args[len(args)-1]
		if strings.HasPrefix(prevArg, "-") && !strings.Contains(prevArg, "=") &&
			!strings.HasPrefix(currArg, "-") {
			// Don't suggest any flags as the last argument was "--foo",
			// so the user must provide a flag value
			return
		}
	}
	err.FlagSet.VisitAll(func(f *flag.Flag) {
		switch f.Value.(type) {
		case *BoolFlag:
			_, _ = fmt.Fprintf(writer, "--%s=true\t%s\n", f.Name, f.Usage)
		default:
			_, _ = fmt.Fprintf(writer, "--%s\t%s\n", f.Name, f.Usage)
		}
	})
}
