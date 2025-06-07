package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHelmDocsComment(t *testing.T) {
	tests := []struct {
		name    string
		comment []string
		want    HelmDocsComment
	}{
		{
			name:    "empty slice",
			comment: []string{},
			want:    HelmDocsComment{},
		},
		{
			name:    "empty string",
			comment: []string{""},
			want:    HelmDocsComment{},
		},

		{
			name: "description simple",
			comment: []string{
				"# -- This is my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name: "description multiline",
			comment: []string{
				"# -- This is",
				"# my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name: "description only second line",
			comment: []string{
				"# --",
				"#This is my description",
			},
			want: HelmDocsComment{
				Description: " This is my description",
			},
		},
		{
			name: "description multiline no spacing",
			comment: []string{
				"# --This is",
				"#my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name: "description no spacing",
			comment: []string{
				"# --This is my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name: "description extra dashes",
			comment: []string{
				"# ------- This is my description",
			},
			want: HelmDocsComment{
				Description: "----- This is my description",
			},
		},
		{
			name: "description continue after keyword",
			comment: []string{
				"# -- This is",
				"# @default -- foo",
				"# my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
				Default:     "foo",
			},
		},

		{
			name: "type only",
			comment: []string{
				"# -- (myType)",
			},
			want: HelmDocsComment{
				Description: "",
				Type:        "myType",
			},
		},
		{
			name: "type with description",
			comment: []string{
				"# -- (myType) This is my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
				Type:        "myType",
			},
		},

		{
			name: "path only",
			comment: []string{
				"# myField --",
			},
			want: HelmDocsComment{
				Path: []string{"myField"},
			},
		},
		{
			name: "path with description",
			comment: []string{
				"# myField -- This is my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
				Path:        []string{"myField"},
			},
		},
		{
			name: "path with segments",
			comment: []string{
				"# myField.foo.bar -- This is my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
				Path:        []string{"myField", "foo", "bar"},
			},
		},
		{
			name: "path with quoted segments",
			comment: []string{
				"# myField.\"foo.bar\" -- This is my description",
			},
			want: HelmDocsComment{
				Description: "This is my description",
				Path:        []string{"myField", "foo.bar"},
			},
		},

		{
			name: "notationType tpl",
			comment: []string{
				"# --",
				"# @notationType -- tpl",
			},
			want: HelmDocsComment{
				NotationType: "tpl",
			},
		},
		{
			name: "notationType fail",
			comment: []string{
				"# --",
				"# @notationType tpl",
			},
			want: HelmDocsComment{
				Description: " @notationType tpl",
			},
		},

		{
			name: "default value",
			comment: []string{
				"# --",
				"# @default -- 123",
			},
			want: HelmDocsComment{
				Default: "123",
			},
		},
		{
			name: "default fail",
			comment: []string{
				"# --",
				"# @default 123",
			},
			want: HelmDocsComment{
				Description: " @default 123",
			},
		},

		{
			name: "section value",
			comment: []string{
				"# --",
				"# @section -- foo",
			},
			want: HelmDocsComment{
				Section: "foo",
			},
		},
		{
			name: "section fail",
			comment: []string{
				"# --",
				"# @section foo",
			},
			want: HelmDocsComment{
				Description: " @section foo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmDocs, err := ParseHelmDocsComment(tt.comment)
			require.NoErrorf(t, err, "Comment: %q", tt.comment)
			assert.Equalf(t, tt.want, helmDocs, "Comment: %q", tt.comment)
		})
	}
}

func TestParseHelmDocsComment_Error(t *testing.T) {
	tests := []struct {
		name    string
		comment []string
		wantErr string
	}{
		{
			name: "schema annotations in helm-docs",
			comment: []string{
				"# -- This is my description",
				"# @schema foo: bar",
			},
			wantErr: "'# @schema' comments are not supported in helm-docs comments",
		},
		{
			name: "schema annotations with minimal spacing in helm-docs",
			comment: []string{
				"# -- This is my description",
				"#@schema foo:bar",
			},
			wantErr: "'# @schema' comments are not supported in helm-docs comments",
		},
		{
			name: "invalid path",
			comment: []string{
				"# foo.\"bar -- This is my description",
			},
			wantErr: "invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseHelmDocsComment(tt.comment)
			require.ErrorContainsf(t, err, tt.wantErr, "Comment: %q", tt.comment)
		})
	}
}

func TestParseHelmDocsPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "empty",
			path: ``,
			want: nil,
		},
		{
			name: "single value",
			path: `foo`,
			want: []string{"foo"},
		},
		{
			name: "quote in unquoted",
			path: `foo"bar"baz`,
			want: []string{"foo\"bar\"baz"},
		},
		{
			name: "quote in unquoted then EOL",
			path: `foo"bar"`,
			want: []string{"foo\"bar\""},
		},
		{
			name: "quote in unquoted then next key",
			path: `foo"bar".baz`,
			want: []string{"foo\"bar\"", "baz"},
		},
		{
			name: "quote with special char in unquoted",
			path: `foo"./bar"`,
			want: []string{"foo\"./bar\""},
		},
		{
			name: "multiple values",
			path: `foo.bar.moo.doo`,
			want: []string{"foo", "bar", "moo", "doo"},
		},
		{
			name: "quoted simple value",
			path: `foo."barmoo".doo`,
			want: []string{"foo", "barmoo", "doo"},
		},
		{
			name: "quoted value with special characters",
			path: `foo."bar.moo/baz!".doo`,
			want: []string{"foo", "bar.moo/baz!", "doo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHelmDocsPath(tt.path)
			require.NoErrorf(t, err, "Path: %q", tt.path)
			assert.Equalf(t, tt.want, got, "Path: %q", tt.path)
		})
	}
}

func TestParseHelmDocsPath_Error(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "start with dot",
			path:    `.foo`,
			wantErr: "invalid syntax",
		},
		{
			// This may seem like valid syntax. But helm-docs does not support it, so neither do we
			name:    "start with quote",
			path:    `"foo"`,
			wantErr: "must not start with a quote",
		},
		{
			name:    "unclosed quote",
			path:    `foo."bar`,
			wantErr: "invalid syntax",
		},
		{
			name:    "escaped quote",
			path:    `foo."bar\"moo"`,
			wantErr: "expected dot separator, but got 'm' in: foo.\"bar\\\"moo\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseHelmDocsPath(tt.path)
			require.ErrorContainsf(t, err, tt.wantErr, "Path: %q", tt.path)
		})
	}
}

func TestSplitHeadCommentsByHelmDocs(t *testing.T) {
	tests := []struct {
		name         string
		comment      string
		wantSchema   []string
		wantHelmDocs []string
	}{
		{
			name:         "empty",
			comment:      "",
			wantSchema:   nil,
			wantHelmDocs: nil,
		},
		{
			name:         "no helm-docs",
			comment:      "# @schema type:string",
			wantSchema:   []string{"# @schema type:string"},
			wantHelmDocs: nil,
		},
		{
			name:         "only helm-docs",
			comment:      "# -- This is my description",
			wantSchema:   []string{},
			wantHelmDocs: []string{"# -- This is my description"},
		},
		{
			name: "only last comment block",
			comment: "" +
				"# comment block 1\n" +
				"# foobar\n" +
				"\n" +
				"# comment block 2\n" +
				"# moo doo\n" +
				"\n" +
				"# comment block 3\n" +
				"# lorem ipsum",
			wantSchema: []string{
				"# comment block 3",
				"# lorem ipsum",
			},
			wantHelmDocs: nil,
		},
		{
			name: "split after comment",
			comment: "" +
				"# @schema type:string\n" +
				"# -- This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# -- This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name: "split after multiline comment",
			comment: "" +
				"# @schema type:string\n" +
				"# -- This is my description\n" +
				"# some other text for the description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# -- This is my description",
				"# some other text for the description",
				"# @schema foo:bar",
			},
		},
		{
			name: "split after typed comment",
			comment: "" +
				"# @schema type:string\n" +
				"# -- (string) This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# -- (string) This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name: "split after pathed comment",
			comment: "" +
				"# @schema type:string\n" +
				"# myField.foobar -- (string) This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# myField.foobar -- (string) This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name: "split after quoted path comment",
			comment: "" +
				"# @schema type:string\n" +
				"# myField.\"foo bar! :D\".hello -- (string) This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# myField.\"foo bar! :D\".hello -- (string) This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name: "split after compact comment",
			comment: "" +
				"# @schema type:string\n" +
				"# --(string)This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# --(string)This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name: "split after a lot of dashes",
			comment: "" +
				"# @schema type:string\n" +
				"# ----- This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
			},
			wantHelmDocs: []string{
				"# ----- This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name: "ignore after invalid pathed comment",
			comment: "" +
				"# @schema type:string\n" +
				"# a b -- This is my description\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
				"# a b -- This is my description",
				"# @schema foo:bar",
			},
			wantHelmDocs: nil,
		},
		{
			// The "# -- Description" line is required. Without it helm-docs ignores its other comments like "@default"
			name: "ignore default without description line",
			comment: "" +
				"# @schema type:string\n" +
				"# @default -- foobar\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
				"# @default -- foobar",
				"# @schema foo:bar",
			},
			wantHelmDocs: nil,
		},
		{
			name: "ignore when no spacing before dashes",
			comment: "" +
				"# @schema type:string\n" +
				"#-- foobar\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
				"#-- foobar",
				"# @schema foo:bar",
			},
			wantHelmDocs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaComments, helmDocsComments := SplitHelmDocsComment(tt.comment)
			assert.Equal(t, tt.wantSchema, schemaComments, "Schema comments")
			assert.Equal(t, tt.wantHelmDocs, helmDocsComments, "helm-docs comments")
		})
	}
}
