package pkg

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/losisin/helm-values-schema-json/v2/internal/yamlfile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "helm",
		Args: cobra.NoArgs,
		Example: `  # Reads values.yaml and outputs to values.schema.json
  helm schema

  # Reads from other-values.yaml (only) and outputs to values.schema.json
  helm schema -f other-values.yaml

  # Reads from multiple files, either comma-separated or use flag multiple times
  helm schema -f values_1.yaml,values_2.yaml
  helm schema -f values_1.yaml -f values_2.yaml

  # Bundle schemas mentioned by one of these comment formats:
  #   myField: {} # @schema $ref: file://some/file/relative/to/values/file
  #   myField: {} # @schema $ref: some/file/relative/to/values/file
  #   myField: {} # @schema $ref: https://example.com/schema.json
  helm schema --bundle

  # Use descriptions from helm-docs
  # https://github.com/norwoodj/helm-docs
  helm schema --use-helm-docs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := LoadConfig(cmd)
			if err != nil {
				return err
			}
			return GenerateJsonSchema(cmd.Context(), config)
		},
		SilenceErrors: true,
		SilenceUsage:  true,

		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "helm schema",
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "version for helm schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			version := cmp.Or(cmd.Root().Version, "(unset)")
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", cmd.Root().DisplayName(), version)
			return err
		},
	}
	cmd.AddCommand(versionCmd)

	cmd.PersistentFlags().String("config", ".schema.yaml", "Config file for setting defaults.")

	cmd.Flags().StringSliceP("values", "f", DefaultConfig.Values, "One or more YAML files as inputs. Use comma-separated list or supply flag multiple times")
	cmd.Flags().StringP("output", "o", DefaultConfig.Output, "Output file path")
	cmd.Flags().Int("draft", DefaultConfig.Draft, "Draft version (4, 6, 7, 2019, or 2020)")
	cmd.Flags().Int("indent", DefaultConfig.Indent, "Indentation spaces (even number)")
	cmd.Flags().Bool("no-additional-properties", false, "Default additionalProperties to false for all objects in the schema")
	cmd.Flags().Bool("no-default-global", false, "Disable automatic injection of 'global' property when schema root does not allow it")

	cmd.Flags().Bool("bundle", false, "Bundle referenced ($ref) subschemas into a single file inside $defs")
	cmd.Flags().Bool("bundle-without-id", false, "Bundle without using $id to reference bundled schemas, which improves compatibility with e.g the VS Code JSON extension")
	cmd.Flags().String("bundle-root", "", "Root directory to allow local referenced files to be loaded from (default current working directory)")

	cmd.Flags().String("k8s-schema-url", DefaultConfig.K8sSchemaURL, "URL template used in $ref: $k8s/... alias")
	cmd.Flags().String("k8s-schema-version", "", "Version used in the --k8s-schema-url template for $ref: $k8s/... alias")

	cmd.Flags().Bool("use-helm-docs", false, "Read description from https://github.com/norwoodj/helm-docs comments")

	// Nested SchemaRoot flags
	cmd.Flags().String("schema-root.id", "", "JSON schema ID")
	cmd.Flags().String("schema-root.ref", "", "JSON schema URI reference. Relative to current working directory when using \"-bundle true\".")
	cmd.Flags().String("schema-root.title", "", "JSON schema title")
	cmd.Flags().String("schema-root.description", "", "JSON schema description")
	cmd.Flags().Bool("schema-root.additional-properties", false, "Allow additional properties")

	return cmd
}

// Flags are only used in testing to achieve better test coverage
var (
	failConfigFlagLoad             bool
	failConfigUnmarshal            bool
	failConfigConfigRefReferrerAbs bool
)

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	k := koanf.New(".")
	var refReferrer Referrer

	configFlag := cmd.Flag("config")
	configPath := configFlag.Value.String()
	if err := k.Load(yamlfile.Provider(DefaultConfig, configPath, "koanf"), nil); err != nil {
		// ignore "not exists" errors, unless user specified the "--config" flag
		if !os.IsNotExist(err) || configFlag.Changed {
			return nil, fmt.Errorf("load config file %s: %w", configPath, err)
		}
	}

	if k.String(schemaRootRefKey) != "" {
		configAbsPath, err := filepath.Abs(configPath)
		if err != nil || failConfigConfigRefReferrerAbs {
			// [filepath.Abs] can't fail here because we already loaded the config file,
			// so resolving its absolute position is guaranteed to also work
			// (except for a race condition, but that's super tricky to test for)
			return nil, fmt.Errorf("resolve absolute path of config file: %w", err)
		}
		refReferrer = ReferrerDir(filepath.Dir(configAbsPath))
	}

	if err := k.Load(posflag.ProviderWithFlag(cmd.Flags(), ".", k, func(f *pflag.Flag) (string, any) {
		if !f.Changed && f.Value.Type() == "bool" {
			// ignore boolean flags that are not explicitly set
			// this allows "schemaRoot.additionalProperties" to stay as null when unset
			return "", nil
		}

		return f.Name, posflag.FlagVal(cmd.Flags(), f)
	}), nil); err != nil || failConfigFlagLoad {
		// The [posflag] provider can't fail, so we have to induce a fake failure via [failConfigFlagLoad]
		return nil, fmt.Errorf("load flags: %w", err)
	}

	if cmd.Flag(schemaRootRefKey).Changed {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve current working directory: %w", err)
		}
		refReferrer = ReferrerDir(cwd)
	}

	var config Config
	if err := k.Unmarshal("", &config); err != nil || failConfigUnmarshal {
		// Now that we use our internal [yamlfile] package, then the parsing of field types are done
		// in that "k.Load" step.
		// Meaning, this "k.Unmarshal" will never fail, so we have to induce a fake failure via [failConfigUnmarshal]
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	config.SchemaRoot.RefReferrer = refReferrer

	return &config, nil
}

var DefaultConfig = Config{
	Values: []string{"values.yaml"},
	Output: "values.schema.json",
	Draft:  2020,
	Indent: 4,

	K8sSchemaURL: "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{ .K8sSchemaVersion }}/",
}

// Save values of parsed flags in Config
type Config struct {
	Values                 []string `yaml:"values" koanf:"values"`
	Output                 string   `yaml:"output" koanf:"output"`
	Draft                  int      `yaml:"draft" koanf:"draft"`
	Indent                 int      `yaml:"indent" koanf:"indent"`
	NoAdditionalProperties bool     `yaml:"noAdditionalProperties" koanf:"no-additional-properties"`
	NoDefaultGlobal        bool     `yaml:"noDefaultGlobal" koanf:"no-default-global"`
	Bundle                 bool     `yaml:"bundle" koanf:"bundle"`
	BundleRoot             string   `yaml:"bundleRoot" koanf:"bundle-root"`
	BundleWithoutID        bool     `yaml:"bundleWithoutID" koanf:"bundle-without-id"`

	K8sSchemaURL     string `yaml:"k8sSchemaURL" koanf:"k8s-schema-url"`
	K8sSchemaVersion string `yaml:"k8sSchemaVersion" koanf:"k8s-schema-version"`

	UseHelmDocs bool `yaml:"useHelmDocs" koanf:"use-helm-docs"`

	SchemaRoot SchemaRoot `yaml:"schemaRoot" koanf:"schema-root"`
}

const schemaRootRefKey = "schema-root.ref"

// SchemaRoot struct defines root object of schema
type SchemaRoot struct {
	ID                   string   `yaml:"id" koanf:"id"`
	Ref                  string   `yaml:"ref" koanf:"ref"`
	RefReferrer          Referrer `yaml:"-" koanf:"-"`
	Title                string   `yaml:"title" koanf:"title"`
	Description          string   `yaml:"description" koanf:"description"`
	AdditionalProperties *bool    `yaml:"additionalProperties" koanf:"additional-properties"`
}
