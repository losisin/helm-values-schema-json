package pkg

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "helm [directory]",
		Args: cobra.ArbitraryArgs,
		Example: `  # Reads values.yaml and outputs to values.schema.json
  helm schema

  # Reads ./my-chart/values.yaml and outputs to ./my-chart/values.schema.json
  helm schema ./my-chart

  # Run on multiple chart directories
  helm schema ./my-first-chart ./my-second-chart

  # Reads from other-values.yaml (only) and outputs to values.schema.json
  helm schema -f other-values.yaml

  # Reads from ./my-chart/other-values.yaml (only) and outputs to ./my-chart/values.schema.json
  helm schema ./my-chart -f other-values.yaml

  # Reads from multiple files, either comma-separated or use flag multiple times
  helm schema -f values_1.yaml,values_2.yaml
  helm schema -f values_1.yaml -f values_2.yaml

  # Run in all subdirectories containing a Chart.yaml file
  helm schema --recursive ./my-charts
  helm schema -r ./my-charts

  # Glob patterns are supported when using '--recursive'
  helm schema --recursive "charts/prod-*/*"
  helm schema -r "charts/prod-*/*"

  # Bundle schemas mentioned by one of these comment formats:
  #   myField: {} # @schema $ref: file://some/file/relative/to/values/file
  #   myField: {} # @schema $ref: some/file/relative/to/values/file
  #   myField: {} # @schema $ref: https://example.com/schema.json
  helm schema --bundle

  # Use descriptions from helm-docs
  # https://github.com/norwoodj/helm-docs
  helm schema --use-helm-docs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cb, err := NewInitializedConfigBuilder(cmd)
			if err != nil {
				return err
			}
			config, err := cb.Build()
			if err != nil {
				return err
			}
			dirs, err := ParseArgs(cmd.Context(), args, config)
			if err != nil {
				return err
			}
			httpLoader := NewCacheLoader(NewHTTPLoader(http.DefaultClient, NewHTTPCache()))
			return GenerateForCharts(cmd.Context(), cmd, httpLoader, dirs, cb)
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

	cmd.Flags().BoolP("recursive", "r", false, "Look for chart directories recursively.\nArguments are then glob patterns to find directories that contain one of the '--recursive-needs' files.\nUsing '--recursive' with no arguments implies the glob pattern '**/'")
	cmd.Flags().StringSlice("recursive-needs", DefaultConfig.RecursiveNeeds, "List of files used to filter the directories found with the glob patterns.")
	cmd.Flags().Bool("no-recursive-needs", false, "Disables the '--recursive-needs' filter, meaning all directories that match the glob patterns are used.\nOnly applies if '--recursive' is enabled.")
	cmd.Flags().BoolP("hidden", "H", false, "Include hidden directories (dirs that start with a dot, e.g '.my-hidden-dir/') when using --recursive.")
	cmd.Flags().Bool("no-gitignore", false, "Disable Git integration when using '--recursive', meaning '.gitignore' files will not be respected.")

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
	failConfigStructsLoad          bool
	failConfigMerge                bool
)

type ConfigBuilder struct {
	fileRefReferrer  Referrer
	fileConfig       Config
	flags            *koanf.Koanf
	flagsRefReferrer Referrer
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		flags:      koanf.New("."),
		fileConfig: DefaultConfig.Clone(),
	}
}

func NewInitializedConfigBuilder(cmd *cobra.Command) (*ConfigBuilder, error) {
	cb := NewConfigBuilder()

	configFlag := cmd.Flag("config")
	if err := cb.LoadFile(configFlag.Value.String(), configFlag.Changed); err != nil {
		return nil, err
	}
	if err := cb.LoadFlags(cmd); err != nil {
		return nil, err
	}

	return cb, nil
}

func (b *ConfigBuilder) Clone() *ConfigBuilder {
	clone := *b
	clone.fileConfig = b.fileConfig.Clone()
	clone.flags = b.flags.Copy()
	return &clone
}

func (b *ConfigBuilder) LoadFile(configPath string, isConfigRequired bool) error {
	if err := decodeYAMLFile(configPath, &b.fileConfig); err != nil {
		if (!os.IsNotExist(err) || isConfigRequired) && !errors.Is(err, io.EOF) {
			return fmt.Errorf("load config file %s: parsing config: %w", configPath, err)
		}
	}

	// TODO: add test for
	// 1. ConfigBuilder.Config already has some data
	// 2. load file that doesnt exist
	// 3. refReferrer should still be set
	refReferrer, err := getConfigRefReferrer(&b.fileConfig, configPath)
	if err != nil {
		return fmt.Errorf("load config file %s: %w", configPath, err)
	}
	b.fileRefReferrer = refReferrer
	return nil
}

func decodeYAMLFile(path string, v any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	return decoder.Decode(v)
}

func (b *ConfigBuilder) LoadFlags(cmd *cobra.Command) error {
	if err := b.flags.Load(posflag.ProviderWithFlag(cmd.Flags(), ".", b.flags, func(f *pflag.Flag) (string, any) {
		if !f.Changed {
			// ignore flags that are not explicitly set
			// this allows fields to override the file configs properly
			return "", nil
		}

		return f.Name, posflag.FlagVal(cmd.Flags(), f)
	}), nil); err != nil || failConfigFlagLoad {
		// The [posflag] provider can't fail, so we have to induce a fake failure via [failConfigFlagLoad]
		return fmt.Errorf("load flags: %w", err)
	}

	if cmd.Flag(schemaRootRefKey).Changed {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve current working directory: %w", err)
		}
		b.flagsRefReferrer = ReferrerDir(cwd)
	}

	return nil
}

func (b *ConfigBuilder) Build() (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(structs.Provider(b.fileConfig, "koanf"), nil); err != nil || failConfigStructsLoad {
		// This "structs.Provider" will never fail, so we have to induce a fake failure via [failConfigStructsLoad]
		return nil, fmt.Errorf("apply config file: %w", err)
	}
	if err := k.Merge(b.flags); err != nil || failConfigMerge {
		// This "k.Merge" will never fail, so we have to induce a fake failure via [failConfigMerge]
		return nil, fmt.Errorf("apply config from flags: %w", err)
	}

	refReferrer := b.fileRefReferrer
	if b.flagsRefReferrer != (Referrer{}) {
		refReferrer = b.flagsRefReferrer
	}
	return unmarshalKoanfConfig(k, refReferrer)
}

func getConfigRefReferrer(config *Config, configPath string) (Referrer, error) {
	// Only set referrer if the ref was also set.
	// No need to specify the referrer otherwise
	if config.SchemaRoot.Ref != "" {
		configAbsPath, err := filepath.Abs(configPath)
		if err != nil || failConfigConfigRefReferrerAbs {
			// [filepath.Abs] can't fail here because we already loaded the config file,
			// so resolving its absolute position is guaranteed to also work
			// (except for a race condition, but that's super tricky to test for)
			return Referrer{}, fmt.Errorf("resolve absolute path of config file: %w", err)
		}
		return ReferrerDir(filepath.Dir(configAbsPath)), nil
	}
	return Referrer{}, nil
}

func unmarshalKoanfConfig(k *koanf.Koanf, referrer Referrer) (*Config, error) {
	var config Config
	if err := k.Unmarshal("", &config); err != nil || failConfigUnmarshal {
		// This "k.Unmarshal" will never fail, so we have to induce a fake failure via [failConfigUnmarshal]
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	config.SchemaRoot.RefReferrer = referrer
	return &config, nil
}

var DefaultConfig = Config{
	Values: []string{"values.yaml"},
	Output: "values.schema.json",
	Draft:  2020,
	Indent: 4,

	RecursiveNeeds: []string{"Chart.yaml"},

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

	Recursive        bool     `yaml:"recursive" koanf:"recursive"`
	RecursiveNeeds   []string `yaml:"recursiveNeeds" koanf:"recursive-needs"`
	NoRecursiveNeeds bool     `yaml:"noRecursiveNeeds" koanf:"no-recursive-needs"`
	Hidden           bool     `yaml:"hidden" koanf:"hidden"`
	NoGitIgnore      bool     `yaml:"noGitIgnore" koanf:"no-gitignore"`

	Bundle          bool   `yaml:"bundle" koanf:"bundle"`
	BundleRoot      string `yaml:"bundleRoot" koanf:"bundle-root"`
	BundleWithoutID bool   `yaml:"bundleWithoutID" koanf:"bundle-without-id"`

	K8sSchemaURL     string `yaml:"k8sSchemaURL" koanf:"k8s-schema-url"`
	K8sSchemaVersion string `yaml:"k8sSchemaVersion" koanf:"k8s-schema-version"`

	UseHelmDocs bool `yaml:"useHelmDocs" koanf:"use-helm-docs"`

	SchemaRoot SchemaRoot `yaml:"schemaRoot" koanf:"schema-root"`
}

func (c Config) Clone() Config {
	c.Values = slices.Clone(c.Values)
	c.RecursiveNeeds = slices.Clone(c.RecursiveNeeds)
	c.SchemaRoot = c.SchemaRoot.Clone()
	return c
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

func (s SchemaRoot) Clone() SchemaRoot {
	if s.AdditionalProperties != nil {
		s.AdditionalProperties = boolPtr(*s.AdditionalProperties)
	}
	return s
}
