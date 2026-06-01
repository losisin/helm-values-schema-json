package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

// newLintCmd creates the "lint" subcommand, which parses the configured input
// files using the same parsing as schema generation and reports any errors, and
// warns about unknown fields in the config file.
func newLintCmd() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint the config file and its input values files",
		Long: "Lint parses the configured input values files using the same parsing as " +
			"schema generation and reports any errors. It also checks the config file " +
			"(.schema.yaml) for unknown fields and logs them as warnings.",
		Example: `  # Lint using .schema.yaml in the current directory
  helm schema lint

  # Fail with a non-zero exit code when any warning is reported
  helm schema lint --strict`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := LoadConfig(cmd)
			if err != nil {
				return err
			}
			return Lint(cmd.Context(), config, LintOptions{
				Strict:     strict,
				ConfigPath: cmd.Flag("config").Value.String(),
			})
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Fail with a non-zero exit code when any warning is reported")

	return cmd
}

// LintOptions configures [Lint].
type LintOptions struct {
	// Strict makes Lint return an error when at least one warning is reported.
	Strict bool
	// ConfigPath is the path to the config file checked for unknown fields.
	ConfigPath string
}

// Lint parses the configured input files (reusing the same parsing as schema
// generation) and checks the config file for unknown fields, logging each one
// as a warning. It returns an error when parsing fails, or when LintOptions.Strict
// is set and at least one warning was reported.
func Lint(ctx context.Context, config *Config, opts LintOptions) error {
	logger := LoggerFromContext(ctx)

	// Reuse the exact same parsing and validation as schema generation.
	if _, err := buildJSONSchema(ctx, config); err != nil {
		return err
	}

	warnings, err := lintConfigUnknownFields(opts.ConfigPath)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		logger.Logf("warning: %s", warning)
	}

	if len(warnings) > 0 {
		logger.Logf("Found %d warning(s)", len(warnings))
		if opts.Strict {
			return fmt.Errorf("found %d warning(s) in strict mode", len(warnings))
		}
		return nil
	}

	logger.Log("No issues found")
	return nil
}

// lintConfigUnknownFields decodes the config file with strict field checking and
// returns one warning per unknown field. A missing config file yields no
// warnings, matching the lenient behavior of config loading.
func lintConfigUnknownFields(configPath string) ([]string, error) {
	if configPath == "" {
		return nil, nil
	}

	content, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", configPath, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)

	var config Config
	switch err := decoder.Decode(&config); {
	case err == nil:
		return nil, nil
	case errors.Is(err, io.EOF):
		// Empty config file: no fields, so no unknown fields.
		return nil, nil
	default:
		// KnownFields(true) reports both unknown fields and type mismatches as a
		// *yaml.TypeError. Only unknown fields are lint warnings; a type mismatch
		// means the config is invalid, so surface it as a hard error.
		var typeErr *yaml.TypeError
		if errors.As(err, &typeErr) {
			var warnings, invalid []string
			for _, msg := range typeErr.Errors {
				if cleaned, ok := unknownFieldWarning(msg); ok {
					warnings = append(warnings, cleaned)
				} else {
					invalid = append(invalid, msg)
				}
			}
			if len(invalid) > 0 {
				return nil, fmt.Errorf("parse config file %q: %s", configPath, strings.Join(invalid, "; "))
			}
			return warnings, nil
		}
		// Any other error means the config YAML itself is malformed.
		return nil, fmt.Errorf("parse config file %q: %w", configPath, err)
	}
}

// unknownFieldWarning rewrites the YAML decoder's
// "field X not found in type pkg.Config" message into a user-facing warning.
// The second return value is false when msg is not an unknown-field message
// (e.g. a type mismatch), so the caller can treat it as a hard error instead.
//
// This is coupled to the go.yaml.in/yaml/v3 error wording; TestCleanUnknownFieldMessage
// guards the phrasing so a library bump surfaces as a test failure.
func unknownFieldWarning(msg string) (string, bool) {
	if idx := strings.Index(msg, " not found in type "); idx != -1 {
		return msg[:idx] + " is not a known config field", true
	}
	return msg, false
}
