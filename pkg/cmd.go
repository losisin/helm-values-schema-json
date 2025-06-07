package pkg

import (
	"fmt"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "helm schema",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := LoadConfig(cmd)
			if err != nil {
				return err
			}
			return GenerateJsonSchema(config)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().String("config", ".schema.yaml", "Config file for setting defaults.")

	cmd.Flags().StringSliceP("input", "i", []string{"values.yaml"}, "Multiple yaml files as inputs. Use comma-separated list or supply flag multiple times")
	cmd.Flags().String("output", "values.schema.json", "Output file path")
	cmd.Flags().Int("draft", 2020, "Draft version (4, 6, 7, 2019, or 2020)")
	cmd.Flags().Int("indent", 4, "Indentation spaces (even number)")
	cmd.Flags().Bool("noAdditionalProperties", false, "Default additionalProperties to false for all objects in the schema")

	cmd.Flags().Bool("bundle", false, "Bundle referenced ($ref) subschemas into a single file inside $defs")
	cmd.Flags().Bool("bundleWithoutID", false, "Bundle without using $id to reference bundled schemas, which improves compatibility with e.g the VS Code JSON extension")
	cmd.Flags().String("bundleRoot", "", "Root directory to allow local referenced files to be loaded from (default current working directory)")

	cmd.Flags().String("k8sSchemaURL", "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/", "URL template used in $ref: $k8s/... alias")
	cmd.Flags().String("k8sSchemaVersion", "", "Version used in the --k8sSchemaURL template for $ref: $k8s/... alias")

	cmd.Flags().Bool("useHelmDocs", false, "Read description from https://github.com/norwoodj/helm-docs comments")

	// Nested SchemaRoot flags
	cmd.Flags().String("schemaRoot.id", "", "JSON schema ID")
	cmd.Flags().String("schemaRoot.ref", "", "JSON schema URI reference. Relative to current working directory when using \"-bundle true\".")
	cmd.Flags().String("schemaRoot.title", "", "JSON schema title")
	cmd.Flags().String("schemaRoot.description", "", "JSON schema description")
	cmd.Flags().Bool("schemaRoot.additionalProperties", false, "Allow additional properties")

	return cmd
}

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	k := koanf.New(".")

	configFlag := cmd.Flag("config")
	configPath := configFlag.Value.String()
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		// ignore "not exists" errors, unless user specified the "--config" flag
		if !os.IsNotExist(err) || configFlag.Changed {
			return nil, fmt.Errorf("load file %s: %w", configPath, err)
		}
	}

	if err := k.Load(posflag.ProviderWithFlag(cmd.Flags(), ".", k, func(f *pflag.Flag) (string, any) {
		if !f.Changed && f.Value.Type() == "bool" {
			// ignore boolean flags that are not explicitly set
			// this allows "schemaRoot.additionalProperties" to stay as null when unset
			return "", nil
		}
		return f.Name, posflag.FlagVal(cmd.Flags(), f)
	}), nil); err != nil {
		return nil, fmt.Errorf("load flags: %w", err)
	}

	var config Config
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{
		Tag: "yaml",
	}); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &config, nil
}

// SchemaRoot struct defines root object of schema
type SchemaRoot struct {
	ID                   string `yaml:"id"`
	Ref                  string `yaml:"ref"`
	Title                string `yaml:"title"`
	Description          string `yaml:"description"`
	AdditionalProperties *bool  `yaml:"additionalProperties"`
}

// Save values of parsed flags in Config
type Config struct {
	Input                  []string `yaml:"input"`
	Output                 string   `yaml:"output"`
	Draft                  int      `yaml:"draft"`
	Indent                 int      `yaml:"indent"`
	NoAdditionalProperties bool     `yaml:"noAdditionalProperties"`
	Bundle                 bool     `yaml:"bundle"`
	BundleRoot             string   `yaml:"bundleRoot"`
	BundleWithoutID        bool     `yaml:"bundleWithoutID"`
	UseHelmDocs            bool     `yaml:"useHelmDocs"`

	K8sSchemaURL     string `yaml:"k8sSchemaURL"`
	K8sSchemaVersion string `yaml:"k8sSchemaVersion"`

	SchemaRoot SchemaRoot `yaml:"schemaRoot"`
}
