package pkg

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleCmd(t *testing.T) {
	golden, err := os.ReadFile("../testdata/bundle/cmd.bundled.json")
	require.NoError(t, err)
	// Normalize line endings so the byte-exact comparison holds on Windows,
	// where the golden file is checked out with CRLF.
	golden = bytes.ReplaceAll(golden, []byte("\r\n"), []byte("\n"))

	tests := []struct {
		name        string
		args        []string
		wantErr     string
		wantOut     string
		wantContain []string
		wantMissing []string
	}{
		{
			name:    "success",
			args:    []string{"bundle", "--bundle-root", "../testdata/bundle", "../testdata/bundle/cmd.schema.json"},
			wantOut: string(golden),
		},
		{
			name: "bundle without id",
			args: []string{"bundle", "--bundle-without-id", "--bundle-root", "../testdata/bundle", "../testdata/bundle/cmd.schema.json"},
			wantContain: []string{
				`"$ref": "#/$defs/simple-subschema.schema.json"`,
				`"$defs"`,
			},
			wantMissing: []string{`"$id"`},
		},
		{
			name: "custom indent",
			args: []string{"bundle", "--indent", "2", "--bundle-root", "../testdata/bundle", "../testdata/bundle/cmd.schema.json"},
			wantContain: []string{
				"\n  \"type\": \"object\",",
				`"$defs"`,
			},
		},
		{
			name: "config flag warns it is ignored",
			args: []string{"bundle", "--config", ".schema.yaml", "--bundle-root", "../testdata/bundle", "../testdata/bundle/cmd.schema.json"},
			wantContain: []string{
				"warning: --config (and .schema.yaml) is ignored by the bundle command",
				`"$defs"`,
			},
		},
		{
			name:    "odd indent",
			args:    []string{"bundle", "--indent", "3", "../testdata/bundle/cmd.schema.json"},
			wantErr: "indentation must be an even number",
		},
		{
			name:    "missing file",
			args:    []string{"bundle", "../testdata/bundle/does-not-exist.schema.json"},
			wantErr: "read schema file",
		},
		{
			name:    "invalid json",
			args:    []string{"bundle", "../testdata/bundle/invalid-schema.json"},
			wantErr: "parse schema file",
		},
		{
			// Without --bundle-root the referenced file escapes the default
			// sandbox (current working directory), so Bundle returns an error.
			name:    "reference escapes bundle root",
			args:    []string{"bundle", "../testdata/bundle/cmd.schema.json"},
			wantErr: "bundle schemas",
		},
		{
			name:    "no args",
			args:    []string{"bundle"},
			wantErr: "accepts 1 arg(s)",
		},
		{
			name:    "too many args",
			args:    []string{"bundle", "a.json", "b.json"},
			wantErr: "accepts 1 arg(s)",
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
			require.NoError(t, err)
			if tt.wantOut != "" {
				assert.Equal(t, tt.wantOut, buf.String())
			}
			for _, want := range tt.wantContain {
				assert.Contains(t, buf.String(), want)
			}
			for _, missing := range tt.wantMissing {
				assert.NotContains(t, buf.String(), missing)
			}
		})
	}
}

func TestBundleFile_IndentValidation(t *testing.T) {
	tests := []struct {
		name    string
		indent  int
		wantErr string
	}{
		{name: "zero", indent: 0, wantErr: "indentation must be a positive number"},
		{name: "negative", indent: -2, wantErr: "indentation must be a positive number"},
		{name: "odd", indent: 3, wantErr: "indentation must be an even number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := BundleFile(context.Background(), &buf, BundleFileOptions{
				InputFile: "../testdata/bundle/cmd.schema.json",
				Indent:    tt.indent,
			})
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, buf.String())
		})
	}
}

func TestBundleFile_CacheMinValidation(t *testing.T) {
	var buf bytes.Buffer
	err := BundleFile(context.Background(), &buf, BundleFileOptions{
		InputFile: "../testdata/bundle/cmd.schema.json",
		Indent:    DefaultConfig.Indent,
		CacheMin:  "not-a-duration",
	})
	assert.ErrorContains(t, err, "parse bundle cache min duration")
	assert.Empty(t, buf.String())
}

// errWriter always fails, used to exercise the output write error path.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, assert.AnError }

func TestBundleFile_WriteError(t *testing.T) {
	err := BundleFile(context.Background(), errWriter{}, BundleFileOptions{
		InputFile:    "../testdata/bundle/cmd.schema.json",
		Indent:       DefaultConfig.Indent,
		BundleRoot:   "../testdata/bundle",
		K8sSchemaURL: DefaultConfig.K8sSchemaURL,
	})
	assert.ErrorContains(t, err, "write bundled schema")
}

func TestBundleFile_Indent(t *testing.T) {
	var buf bytes.Buffer
	err := BundleFile(context.Background(), &buf, BundleFileOptions{
		InputFile:    "../testdata/bundle/cmd.schema.json",
		Indent:       2,
		BundleRoot:   "../testdata/bundle",
		K8sSchemaURL: DefaultConfig.K8sSchemaURL,
	})
	require.NoError(t, err)
	// 2-space indent puts top-level keys two spaces in.
	assert.Contains(t, buf.String(), "\n  \"type\": \"object\",")
}

func TestBundleFile_AbsError(t *testing.T) {
	failBundleFileAbs = true
	defer func() { failBundleFileAbs = false }()

	var buf bytes.Buffer
	err := BundleFile(context.Background(), &buf, BundleFileOptions{
		InputFile:    "../testdata/bundle/cmd.schema.json",
		Indent:       DefaultConfig.Indent,
		BundleRoot:   "../testdata/bundle",
		K8sSchemaURL: DefaultConfig.K8sSchemaURL,
	})
	assert.ErrorContains(t, err, "get absolute path of")
}

func TestBundleFile_MarshalError(t *testing.T) {
	failBundleFileMarshal = true
	defer func() { failBundleFileMarshal = false }()

	var buf bytes.Buffer
	err := BundleFile(context.Background(), &buf, BundleFileOptions{
		InputFile:    "../testdata/bundle/cmd.schema.json",
		Indent:       DefaultConfig.Indent,
		BundleRoot:   "../testdata/bundle",
		K8sSchemaURL: DefaultConfig.K8sSchemaURL,
	})
	assert.ErrorContains(t, err, "encode bundled schema")
}
