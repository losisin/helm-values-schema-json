package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGetComment(t *testing.T) {
	tests := []struct {
		name         string
		keyNode      *yaml.Node
		valNode      *yaml.Node
		useHelmDocs  bool
		wantComment  []string
		wantHelmDocs []string
	}{
		{
			name: "value node with comment",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				LineComment: "",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "some value",
				LineComment: "# Value comment",
			},
			wantComment: []string{"# Value comment"},
		},
		{
			name: "value node without comment, key node with comment",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				LineComment: "# Key comment",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "some value",
				LineComment: "",
			},
			wantComment: []string{"# Key comment"},
		},
		{
			name: "empty value node, key node with comment",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				LineComment: "# Key comment",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "",
				LineComment: "",
			},
			wantComment: []string{"# Key comment"},
		},
		{
			name: "head comment single line",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				HeadComment: "# Key comment",
				LineComment: "",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "",
				LineComment: "",
			},
			wantComment: []string{"# Key comment"},
		},
		{
			name: "head comment multi line",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				HeadComment: "# Key comment\n# Second line",
				LineComment: "",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "",
				LineComment: "",
			},
			wantComment: []string{"# Key comment", "# Second line"},
		},
		{
			name: "head comment only last comment group",
			keyNode: &yaml.Node{
				Kind: yaml.ScalarNode,
				HeadComment: "# First comment group\n# second line\n\n" +
					"# Second comment group\n# second line 2\n\n" +
					"# Last comment group\n# second line 3",
				LineComment: "",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "",
				LineComment: "",
			},
			wantComment: []string{"# Last comment group", "# second line 3"},
		},
		{
			name: "foot comment multi line",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				FootComment: "# Key comment\n# Second line",
				LineComment: "",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "",
				LineComment: "",
			},
			wantComment: []string{"# Key comment", "# Second line"},
		},
		{
			name: "head, line, and foot comment",
			keyNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				HeadComment: "# Head comment",
				LineComment: "# Line comment",
				FootComment: "# Foot comment",
			},
			valNode: &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "",
				LineComment: "",
			},
			wantComment: []string{"# Head comment", "# Line comment", "# Foot comment"},
		},

		{
			// Helm-docs comments are further tested in TestSplitHeadCommentsByHelmDocs
			name:        "helm-docs/on",
			useHelmDocs: true,
			keyNode: &yaml.Node{
				Kind: yaml.ScalarNode,
				HeadComment: "" +
					"# @schema type:string\n" +
					"# -- This is my description\n" +
					"# @schema foo:bar",
				LineComment: "# Line comment",
				FootComment: "# Foot comment",
			},
			valNode: &yaml.Node{},
			wantComment: []string{
				"# @schema type:string",
				"# Line comment",
				"# Foot comment",
			},
			wantHelmDocs: []string{
				"# -- This is my description",
				"# @schema foo:bar",
			},
		},
		{
			name:        "helm-docs/off",
			useHelmDocs: false,
			keyNode: &yaml.Node{
				Kind: yaml.ScalarNode,
				HeadComment: "" +
					"# @schema type:string\n" +
					"# -- This is my description\n" +
					"# @schema foo:bar",
				LineComment: "# Line comment",
				FootComment: "# Foot comment",
			},
			valNode: &yaml.Node{},
			wantComment: []string{
				"# @schema type:string",
				"# -- This is my description",
				"# @schema foo:bar",
				"# Line comment",
				"# Foot comment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments, helmDocs := getComments(tt.keyNode, tt.valNode, tt.useHelmDocs)
			assert.Equal(t, tt.wantComment, comments, "schema comments")
			assert.Equal(t, tt.wantHelmDocs, helmDocs, "helm-docs comments")
		})
	}
}

func TestSplitCommentByParts(t *testing.T) {
	type Pair struct {
		Key, Value string
	}
	tests := []struct {
		name     string
		comments []string
		want     []Pair
	}{
		{
			name:     "empty",
			comments: nil,
			want:     nil,
		},
		{
			name:     "no keys",
			comments: []string{"# @schema "},
			want:     nil,
		},
		{
			// https://github.com/losisin/helm-values-schema-json/issues/152
			name:     "ignore when missing @schema",
			comments: []string{"# ; type:string"},
			want:     nil,
		},
		{
			name:     "without whitespace",
			comments: []string{"#@schema type:string"},
			want:     []Pair{{"type", "string"}},
		},
		{
			name:     "with extra whitespace",
			comments: []string{"#  \t  @schema \t type :  string"},
			want:     []Pair{{"type", "string"}},
		},
		{
			name:     "missing whitespace after after @schema",
			comments: []string{"# @schematype:string"},
			want:     nil,
		},
		{
			name:     "tab after @schema",
			comments: []string{"# @schema\ttype:string"},
			want:     []Pair{{"type", "string"}},
		},
		{
			name:     "only key",
			comments: []string{"# @schema type"},
			want:     []Pair{{"type", ""}},
		},
		{
			name:     "only value",
			comments: []string{"# @schema : string"},
			want:     []Pair{{"", "string"}},
		},
		{
			name:     "multiple pairs",
			comments: []string{"# @schema type:string; foo:bar"},
			want:     []Pair{{"type", "string"}, {"foo", "bar"}},
		},
		{
			name:     "same pair multiple times",
			comments: []string{"# @schema type:string; type:integer"},
			want:     []Pair{{"type", "string"}, {"type", "integer"}},
		},
		{
			name:     "array value",
			comments: []string{"# @schema type:[string, integer]"},
			want:     []Pair{{"type", "[string, integer]"}},
		},
		{
			name: "multiple comments",
			comments: []string{
				"# @schema type:string",
				"# @schema foo:bar",
				"# @schema moo:doo",
			},
			want: []Pair{
				{"type", "string"},
				{"foo", "bar"},
				{"moo", "doo"},
			},
		},
		{
			name: "multiple pairs on multiple comments",
			comments: []string{
				"# @schema type:string; lorem:ipsum",
				"# @schema foo:bar; foz:baz",
				"# @schema moo:doo; moz:doz",
			},
			want: []Pair{
				{"type", "string"},
				{"lorem", "ipsum"},
				{"foo", "bar"},
				{"foz", "baz"},
				{"moo", "doo"},
				{"moz", "doz"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pairs []Pair
			for key, value := range splitCommentsByParts(tt.comments) {
				pairs = append(pairs, Pair{key, value})
			}
			assert.Equal(t, tt.want, pairs)
		})
	}
}

func TestSplitCommentsByParts_break(t *testing.T) {
	type Pair struct {
		Key, Value string
	}

	comments := []string{"# @schema foo:bar; moo:doo; baz:boz"}

	var pairs []Pair
	for key, value := range splitCommentsByParts(comments) {
		pairs = append(pairs, Pair{key, value})
		if len(pairs) == 2 {
			break
		}
	}

	want := []Pair{{"foo", "bar"}, {"moo", "doo"}}
	assert.Equal(t, want, pairs)
}

func TestProcessList(t *testing.T) {
	tests := []struct {
		name         string
		comment      string
		stringsOnly  bool
		expectedList []any
	}{
		{
			name:         "empty list",
			comment:      "[]",
			stringsOnly:  false,
			expectedList: []any{""},
		},
		{
			name:         "single string",
			comment:      "[\"value\"]",
			stringsOnly:  true,
			expectedList: []any{"value"},
		},
		{
			name:         "single string without quotes",
			comment:      "[value]",
			stringsOnly:  true,
			expectedList: []any{"value"},
		},
		{
			name:         "multiple strings",
			comment:      "[\"value1\", \"value2\"]",
			stringsOnly:  true,
			expectedList: []any{"value1", "value2"},
		},
		{
			name:         "null allowed",
			comment:      "[null]",
			stringsOnly:  false,
			expectedList: []any{nil},
		},
		{
			name:         "null not treated as special",
			comment:      "[null]",
			stringsOnly:  true,
			expectedList: []any{"null"},
		},
		{
			name:         "mixed strings and null",
			comment:      "[\"value1\", null, \"value2\"]",
			stringsOnly:  false,
			expectedList: []any{"value1", nil, "value2"},
		},
		{
			name:         "whitespace trimming",
			comment:      "[ value1, value2 ]",
			stringsOnly:  true,
			expectedList: []any{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := processList(tt.comment, tt.stringsOnly)
			assert.Equal(t, tt.expectedList, list)
		})
	}
}

func TestProcessComment(t *testing.T) {
	tests := []struct {
		name             string
		schema           *Schema
		comment          string
		expectedSchema   *Schema
		expectedRequired bool
	}{
		{
			name:             "Empty comment",
			schema:           &Schema{},
			comment:          "# @schema ",
			expectedSchema:   &Schema{},
			expectedRequired: false,
		},
		{
			name:             "Set type",
			schema:           &Schema{},
			comment:          "# @schema type:[string, null]",
			expectedSchema:   &Schema{Type: []any{"string", "null"}},
			expectedRequired: false,
		},
		{
			name:             "Set enum",
			schema:           &Schema{},
			comment:          "# @schema enum:[one, two, null]",
			expectedSchema:   &Schema{Enum: []any{"one", "two", nil}},
			expectedRequired: false,
		},
		{
			name:             "Set numeric",
			schema:           &Schema{},
			comment:          "# @schema multipleOf:2;minimum:1;maximum:10",
			expectedSchema:   &Schema{MultipleOf: float64Ptr(2), Minimum: float64Ptr(1), Maximum: float64Ptr(10)},
			expectedRequired: false,
		},
		{
			name:             "Set string",
			schema:           &Schema{},
			comment:          "# @schema pattern:^abv$;minLength:2;maxLength:10",
			expectedSchema:   &Schema{Pattern: "^abv$", MinLength: uint64Ptr(2), MaxLength: uint64Ptr(10)},
			expectedRequired: false,
		},
		{
			name:             "Set array",
			schema:           &Schema{},
			comment:          "# @schema minItems:1;maxItems:10;uniqueItems:true;item:object;itemProperties:{\"key\": {\"type\": \"string\"}}",
			expectedSchema:   &Schema{MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true, Items: &Schema{Type: "object", Properties: map[string]*Schema{"key": {Type: "string"}}}},
			expectedRequired: false,
		},
		{
			name:             "Set array only item enum",
			schema:           &Schema{},
			comment:          "# @schema itemEnum:[1,2]",
			expectedSchema:   &Schema{Items: &Schema{Enum: []any{"1", "2"}}},
			expectedRequired: false,
		},
		{
			name:             "Set array item type and enum",
			schema:           &Schema{},
			comment:          "# @schema minItems:1;maxItems:10;uniqueItems:true;item:string;itemEnum:[\"one\",\"two\"]",
			expectedSchema:   &Schema{MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true, Items: &Schema{Type: "string", Enum: []any{"one", "two"}}},
			expectedRequired: false,
		},
		{
			name:             "Set object",
			schema:           &Schema{},
			comment:          "# @schema minProperties:1;maxProperties:10;additionalProperties:false;$id:https://example.com/schema;$ref:schema/product.json",
			expectedSchema:   &Schema{MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), AdditionalProperties: &SchemaFalse, ID: "https://example.com/schema", Ref: "schema/product.json"},
			expectedRequired: false,
		},
		{
			name:             "Set meta-data",
			schema:           &Schema{},
			comment:          "# @schema title:My Title;description: some description;readOnly:false;default:\"foo\"",
			expectedSchema:   &Schema{Title: "My Title", Description: "some description", ReadOnly: false, Default: "foo"},
			expectedRequired: false,
		},
		{
			name:             "Set skipProperties",
			schema:           &Schema{},
			comment:          "# @schema skipProperties:true;unevaluatedProperties:false",
			expectedSchema:   &Schema{SkipProperties: true, UnevaluatedProperties: boolPtr(false)},
			expectedRequired: false,
		},
		{
			name:             "Set hidden",
			schema:           &Schema{},
			comment:          "# @schema hidden:true",
			expectedSchema:   &Schema{},
			expectedRequired: false,
		},
		{
			name:             "Set allOf",
			schema:           &Schema{},
			comment:          "# @schema allOf:[{\"type\":\"string\"}]",
			expectedSchema:   &Schema{AllOf: []*Schema{{Type: "string"}}},
			expectedRequired: false,
		},
		{
			name:             "Set anyOf",
			schema:           &Schema{},
			comment:          "# @schema anyOf:[{\"type\":\"string\"}]",
			expectedSchema:   &Schema{AnyOf: []*Schema{{Type: "string"}}},
			expectedRequired: false,
		},
		{
			name:             "Set oneOf",
			schema:           &Schema{},
			comment:          "# @schema oneOf:[{\"type\":\"string\"}]",
			expectedSchema:   &Schema{OneOf: []*Schema{{Type: "string"}}},
			expectedRequired: false,
		},
		{
			name:             "Set not",
			schema:           &Schema{},
			comment:          "# @schema not:{\"type\":\"string\"}",
			expectedSchema:   &Schema{Not: &Schema{Type: "string"}},
			expectedRequired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var required bool
			processComment(tt.schema, []string{tt.comment})
			assert.Equal(t, tt.expectedSchema, tt.schema)
			assert.Equal(t, tt.expectedRequired, required)
		})
	}
}
