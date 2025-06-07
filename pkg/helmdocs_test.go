package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHelmDocsComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    HelmDocsComment
	}{
		{
			name:    "empty",
			comment: "",
			want:    HelmDocsComment{},
		},
		{
			name:    "no helm-docs",
			comment: "# @schema type:string",
			want: HelmDocsComment{
				CommentsAbove: []string{"# @schema type:string"},
			},
		},

		{
			name:    "description simple",
			comment: "# -- This is my description",
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name: "description multiline",
			comment: "" +
				"# -- This is\n" +
				"# my description",
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name: "description only second line",
			comment: "" +
				"# --\n" +
				"#This is my description",
			want: HelmDocsComment{
				Description: " This is my description",
			},
		},
		{
			name: "description multiline no spacing",
			comment: "" +
				"# --This is\n" +
				"#my description",
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name:    "description no spacing",
			comment: "# --This is my description",
			want: HelmDocsComment{
				Description: "This is my description",
			},
		},
		{
			name:    "description extra dashes",
			comment: "# ------- This is my description",
			want: HelmDocsComment{
				Description: "----- This is my description",
			},
		},
		{
			name: "description continue after keyword",
			comment: "" +
				"# -- This is\n" +
				"# @default -- foo\n" +
				"# my description",
			want: HelmDocsComment{
				Description: "This is my description",
				Default:     "foo",
			},
		},

		{
			name:    "type only",
			comment: "# -- (myType)",
			want: HelmDocsComment{
				Description: "",
				Type:        "myType",
			},
		},
		{
			name:    "type with description",
			comment: "# -- (myType) This is my description",
			want: HelmDocsComment{
				Description: "This is my description",
				Type:        "myType",
			},
		},

		{
			name:    "path only",
			comment: "# myField --",
			want: HelmDocsComment{
				Path: []string{"myField"},
			},
		},
		{
			name:    "path with description",
			comment: "# myField -- This is my description",
			want: HelmDocsComment{
				Description: "This is my description",
				Path:        []string{"myField"},
			},
		},

		{
			name: "notationType tpl",
			comment: "" +
				"# --\n" +
				"# @notationType -- tpl",
			want: HelmDocsComment{
				NotationType: "tpl",
			},
		},
		{
			name: "notationType fail",
			comment: "" +
				"# --\n" +
				"# @notationType tpl",
			want: HelmDocsComment{
				Description: " @notationType tpl",
			},
		},

		{
			name: "default value",
			comment: "" +
				"# --\n" +
				"# @default -- 123",
			want: HelmDocsComment{
				Default: "123",
			},
		},
		{
			name: "default fail",
			comment: "" +
				"# --\n" +
				"# @default 123",
			want: HelmDocsComment{
				Description: " @default 123",
			},
		},

		{
			name: "section value",
			comment: "" +
				"# --\n" +
				"# @section -- 123",
			want: HelmDocsComment{
				Default: "123",
			},
		},
		{
			name: "section fail",
			comment: "" +
				"# --\n" +
				"# @section 123",
			want: HelmDocsComment{
				Description: " @section 123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmDocs, err := ParseHelmDocsComment(tt.comment)
			require.NoError(t, err)
			assert.Equal(t, tt.want, helmDocs)
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
			name: "ignore after invalid first path quoted",
			comment: "" +
				"# @schema type:string\n" +
				"# \"foo\" -- Quotes on first element is not supported by helm-docs\n" +
				"# @schema foo:bar",
			wantSchema: []string{
				"# @schema type:string",
				"# \"foo\" -- Quotes on first element is not supported by helm-docs",
				"# @schema foo:bar",
			},
			wantHelmDocs: nil,
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
			schemaComments, helmDocsComments := splitHeadCommentsByHelmDocs(tt.comment)
			assert.Equal(t, tt.wantSchema, schemaComments, "Schema comments")
			assert.Equal(t, tt.wantHelmDocs, helmDocsComments, "helm-docs comments")
		})
	}
}
