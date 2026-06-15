package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Flags are only used in testing to achieve better test coverage
var (
	failBundleFileAbs     bool
	failBundleFileMarshal bool
)

// newBundleCmd creates the "bundle" subcommand, which reads an existing JSON
// schema file, bundles all its "$ref" subschemas into "$defs", and prints the
// result to stdout.
func newBundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle SCHEMA_FILE",
		Short: "Bundle referenced ($ref) subschemas of a JSON schema file into $defs",
		Long: "Bundle reads an existing JSON schema file, resolves all its \"$ref\" " +
			"subschemas, stores them inside \"$defs\", and prints the bundled schema to stdout.\n\n" +
			"This is the same bundling that \"helm schema --bundle\" performs while generating a " +
			"schema, but applied to an already-existing schema file.",
		Example: `  # Bundle a schema file and print the result to stdout
  helm schema bundle values.schema.json

  # Bundle local references located outside the current directory
  helm schema bundle values.schema.json --bundle-root ..`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// bundle does not call LoadConfig, so the inherited --config flag (and
			// .schema.yaml) has no effect here. Warn when it is set explicitly so
			// the silently-ignored config does not surprise the user.
			if cmd.Flags().Changed("config") {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
					"warning: --config (and .schema.yaml) is ignored by the bundle command; "+
						"pass --bundle-root, --indent and --k8s-schema-version directly")
			}

			// All flags are registered below, so these getters cannot fail; their
			// errors are ignored to avoid an unreachable, uncoverable error branch.
			indent, _ := cmd.Flags().GetInt("indent")
			bundleRoot, _ := cmd.Flags().GetString("bundle-root")
			bundleWithoutID, _ := cmd.Flags().GetBool("bundle-without-id")
			cacheMin, _ := cmd.Flags().GetString("bundle-cache-min")
			k8sSchemaURL, _ := cmd.Flags().GetString("k8s-schema-url")
			k8sSchemaVersion, _ := cmd.Flags().GetString("k8s-schema-version")

			return BundleFile(cmd.Context(), cmd.OutOrStdout(), BundleFileOptions{
				InputFile:        args[0],
				Indent:           indent,
				BundleRoot:       bundleRoot,
				BundleWithoutID:  bundleWithoutID,
				CacheMin:         cacheMin,
				K8sSchemaURL:     k8sSchemaURL,
				K8sSchemaVersion: k8sSchemaVersion,
			})
		},
	}

	registerSharedFlags(cmd.Flags())

	return cmd
}

// BundleFileOptions holds the inputs for [BundleFile].
type BundleFileOptions struct {
	// InputFile is the path to the JSON schema file to bundle.
	InputFile string
	// Indent is the number of spaces used to indent the bundled JSON output.
	Indent int
	// BundleRoot, BundleWithoutID, K8sSchemaURL and K8sSchemaVersion are passed
	// through to [Bundle].
	BundleRoot       string
	BundleWithoutID  bool
	K8sSchemaURL     string
	K8sSchemaVersion string
	// CacheMin is the raw --bundle-cache-min value (e.g. "24h"); it is parsed by
	// [ParseCacheMinDuration] and passed through to [Bundle] to raise the minimum
	// cache duration for downloaded schemas. An empty string means no override.
	CacheMin string
}

// BundleFile reads the JSON schema file referenced by opts.InputFile, bundles
// its "$ref" subschemas into "$defs" using [Bundle], and writes the indented
// result to out.
func BundleFile(ctx context.Context, out io.Writer, opts BundleFileOptions) error {
	if opts.Indent <= 0 {
		return errors.New("indentation must be a positive number")
	}
	if opts.Indent%2 != 0 {
		return errors.New("indentation must be an even number")
	}

	cacheMinDuration, err := ParseCacheMinDuration(opts.CacheMin)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(filepath.Clean(opts.InputFile))
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(content, &schema); err != nil {
		return fmt.Errorf("parse schema file %q: %w", opts.InputFile, err)
	}

	// Resolve "$ref" relative to the input file's directory.
	inputAbs, err := filepath.Abs(opts.InputFile)
	if err != nil || failBundleFileAbs {
		return fmt.Errorf("get absolute path of %q: %w", opts.InputFile, err)
	}
	schema.SetReferrer(ReferrerDir(filepath.Dir(inputAbs)))

	// Bundle's first path argument is treated as an output *file* path; it strips
	// the filename internally to derive the directory used as the cosmetic base
	// for relative $ref/$id paths. Passing the input file path makes those paths
	// relative to the schema file's own directory, matching the generate command.
	if err := Bundle(ctx, &schema, inputAbs, opts.BundleRoot, opts.BundleWithoutID, opts.K8sSchemaURL, opts.K8sSchemaVersion, cacheMinDuration); err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(&schema, "", strings.Repeat(" ", opts.Indent))
	if err != nil || failBundleFileMarshal {
		return fmt.Errorf("encode bundled schema: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')

	if _, err := out.Write(jsonBytes); err != nil {
		return fmt.Errorf("write bundled schema: %w", err)
	}
	return nil
}
