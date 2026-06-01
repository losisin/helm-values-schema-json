package pkg

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLint(t *testing.T) {
	validValues := []string{"../testdata/lint/values.yaml"}

	tests := []struct {
		name        string
		config      *Config
		opts        LintOptions
		wantErr     string
		wantContain []string
	}{
		{
			name:        "no issues",
			config:      &Config{Values: validValues, Draft: 2020, Indent: 4},
			wantContain: []string{"No issues found"},
		},
		{
			name:   "unknown fields warn",
			config: &Config{Values: validValues, Draft: 2020, Indent: 4},
			opts:   LintOptions{ConfigPath: "../testdata/lint/unknown.yaml"},
			wantContain: []string{
				"warning: line 4: field fooBar is not a known config field",
				"warning: line 6: field unknownNested is not a known config field",
				"Found 2 warning(s)",
			},
		},
		{
			name:    "unknown fields strict",
			config:  &Config{Values: validValues, Draft: 2020, Indent: 4},
			opts:    LintOptions{Strict: true, ConfigPath: "../testdata/lint/unknown.yaml"},
			wantErr: "found 2 warning(s) in strict mode",
		},
		{
			name:    "input parse error",
			config:  &Config{Values: []string{"../testdata/lint/values-bad.yaml"}, Draft: 2020, Indent: 4},
			wantErr: `invalid type "bogustype"`,
		},
		{
			name:    "malformed config",
			config:  &Config{Values: validValues, Draft: 2020, Indent: 4},
			opts:    LintOptions{ConfigPath: "../testdata/lint/malformed.yaml"},
			wantErr: "parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := ContextWithLogger(context.Background(), NewLogger(&buf))
			err := Lint(ctx, tt.config, tt.opts)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			for _, want := range tt.wantContain {
				assert.Contains(t, buf.String(), want)
			}
		})
	}
}

func TestLintConfigUnknownFields(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantWarnings int
		wantErr      string
	}{
		{name: "empty path", path: ""},
		{name: "missing file", path: "../testdata/lint/does-not-exist.yaml"},
		{name: "empty file", path: "../testdata/lint/empty.yaml"},
		{name: "all known", path: "../testdata/lint/valid-config.yaml"},
		{name: "unknown fields", path: "../testdata/lint/unknown.yaml", wantWarnings: 2},
		{name: "malformed", path: "../testdata/lint/malformed.yaml", wantErr: "parse config file"},
		{name: "type mismatch is a hard error", path: "../testdata/lint/type-mismatch.yaml", wantErr: "parse config file"},
		{name: "path is a directory", path: "../testdata/lint", wantErr: "read config file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := lintConfigUnknownFields(tt.path)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, warnings, tt.wantWarnings)
		})
	}
}

func TestUnknownFieldWarning(t *testing.T) {
	got, ok := unknownFieldWarning("line 4: field fooBar not found in type pkg.Config")
	assert.True(t, ok)
	assert.Equal(t, "line 4: field fooBar is not a known config field", got)

	got, ok = unknownFieldWarning("line 2: cannot unmarshal !!str into int")
	assert.False(t, ok)
	assert.Equal(t, "line 2: cannot unmarshal !!str into int", got)
}

func TestLintCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name: "valid config",
			args: []string{"lint", "--config", "../testdata/lint/valid-config.yaml"},
		},
		{
			name:    "unknown fields strict fails",
			args:    []string{"lint", "--config", "../testdata/lint/unknown.yaml", "--strict"},
			wantErr: "found 2 warning(s) in strict mode",
		},
		{
			name:    "input parse error",
			args:    []string{"lint", "--config", "../testdata/lint/bad-config.yaml"},
			wantErr: `invalid type "bogustype"`,
		},
		{
			name:    "config load error",
			args:    []string{"lint", "--config", "../testdata/lint/malformed.yaml"},
			wantErr: "load config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
