package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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
			expectedList: []any{},
		},
		{
			name:         "JSON string",
			comment:      "[\"value\"]",
			stringsOnly:  true,
			expectedList: []any{"value"},
		},
		{
			name:         "YAML string",
			comment:      "[value]",
			stringsOnly:  true,
			expectedList: []any{"value"},
		},
		{
			name:         "JSON string with escapes",
			comment:      "[\"foo\\\"bar\\\"moo\"]",
			stringsOnly:  true,
			expectedList: []any{"foo\"bar\"moo"},
		},
		{
			name:         "single string without quotes",
			comment:      "[: this is not YAML :]",
			stringsOnly:  true,
			expectedList: []any{": this is not YAML :"},
		},
		{
			name:         "invalid YAML but still using quotes",
			comment:      "[: this is not YAML :, \"escaping stuff \\\" works\" ]",
			stringsOnly:  true,
			expectedList: []any{": this is not YAML :", "escaping stuff \" works"},
		},
		{
			name:         "invalid YAML with null allowed",
			comment:      "[: this is not YAML :, null]",
			stringsOnly:  false,
			expectedList: []any{": this is not YAML :", nil},
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
			name:         "null as string",
			comment:      "[null]",
			stringsOnly:  true,
			expectedList: []any{"null"},
		},
		{
			name:         "numbers allowed",
			comment:      "[123]",
			stringsOnly:  false,
			expectedList: []any{123},
		},
		{
			name:         "numbers as string",
			comment:      "[123]",
			stringsOnly:  true,
			expectedList: []any{"123"},
		},
		{
			name:         "mixed strings and null",
			comment:      "[\"value1\", null, \"value2\"]",
			stringsOnly:  false,
			expectedList: []any{"value1", nil, "value2"},
		},
		{
			name:         "mixed strings and string null",
			comment:      "[\"value1\", null, \"value2\"]",
			stringsOnly:  true,
			expectedList: []any{"value1", "null", "value2"},
		},
		{
			name:         "whitespace trimming",
			comment:      "[ value1, value2 ]",
			stringsOnly:  true,
			expectedList: []any{"value1", "value2"},
		},
		{
			name:         "trailing comma",
			comment:      "[value1, value2,]",
			stringsOnly:  true,
			expectedList: []any{"value1", "value2"},
		},
		{
			name:         "list of lists",
			comment:      "[[foo], [bar, null]]",
			stringsOnly:  true,
			expectedList: []any{[]any{"foo"}, []any{"bar", "null"}},
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
		name       string
		schema     *Schema
		comment    string
		wantSchema *Schema
	}{
		{
			name:       "Empty comment",
			schema:     &Schema{},
			comment:    "# @schema ",
			wantSchema: &Schema{},
		},
		{
			name:       "Set type",
			schema:     &Schema{},
			comment:    "# @schema type:[string, null]",
			wantSchema: &Schema{Type: []any{"string", "null"}},
		},
		{
			name:       "Set enum",
			schema:     &Schema{},
			comment:    "# @schema enum:[one, two, null]",
			wantSchema: &Schema{Enum: []any{"one", "two", nil}},
		},
		{
			name:    "Set float integers",
			schema:  &Schema{},
			comment: "# @schema multipleOf:2; minimum:1; maximum:10",
			wantSchema: &Schema{
				MultipleOf: float64Ptr(2),
				Minimum:    float64Ptr(1),
				Maximum:    float64Ptr(10),
			},
		},
		{
			name:    "Set float decimals",
			schema:  &Schema{},
			comment: "# @schema multipleOf:2.5; minimum:1.5; maximum:10.5",
			wantSchema: &Schema{
				MultipleOf: float64Ptr(2.5),
				Minimum:    float64Ptr(1.5),
				Maximum:    float64Ptr(10.5),
			},
		},
		{
			name:   "Set float back to null",
			schema: &Schema{},
			comment: "# @schema multipleOf:2; minimum:1; maximum:10; " +
				"multipleOf:null; minimum:null; maximum:null",
			wantSchema: &Schema{
				MultipleOf: nil,
				Minimum:    nil,
				Maximum:    nil,
			},
		},
		{
			name:    "Set integers",
			schema:  &Schema{},
			comment: "# @schema minLength:1; maxLength:2; minItems:3; maxItems:4; minProperties:5; maxProperties:6",
			wantSchema: &Schema{
				MinLength:     uint64Ptr(1),
				MaxLength:     uint64Ptr(2),
				MinItems:      uint64Ptr(3),
				MaxItems:      uint64Ptr(4),
				MinProperties: uint64Ptr(5),
				MaxProperties: uint64Ptr(6),
			},
		},
		{
			name:   "Set integers back to null",
			schema: &Schema{},
			comment: "# @schema minLength:1; maxLength:2; minItems:3; maxItems:4; minProperties:5; maxProperties:6; " +
				"minLength:null; maxLength:null; minItems:null; maxItems:null; minProperties:null; maxProperties:null",
			wantSchema: &Schema{
				MinLength:     nil,
				MaxLength:     nil,
				MinItems:      nil,
				MaxItems:      nil,
				MinProperties: nil,
				MaxProperties: nil,
			},
		},
		{
			name:       "Set string",
			schema:     &Schema{},
			comment:    "# @schema pattern:^abv$;minLength:2;maxLength:10",
			wantSchema: &Schema{Pattern: "^abv$", MinLength: uint64Ptr(2), MaxLength: uint64Ptr(10)},
		},
		{
			name:       "Set array",
			schema:     &Schema{},
			comment:    "# @schema minItems:1;maxItems:10;uniqueItems:true;item:object;itemProperties:{\"key\": {\"type\": \"string\"}}",
			wantSchema: &Schema{MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true, Items: &Schema{Type: "object", Properties: map[string]*Schema{"key": {Type: "string"}}}},
		},
		{
			name:       "Set array only item enum",
			schema:     &Schema{},
			comment:    "# @schema itemEnum:[1,2]",
			wantSchema: &Schema{Items: &Schema{Enum: []any{1, 2}}},
		},
		{
			name:       "Set array item type and enum",
			schema:     &Schema{},
			comment:    "# @schema minItems:1;maxItems:10;uniqueItems:true;item:string;itemEnum:[\"one\",\"two\"]",
			wantSchema: &Schema{MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true, Items: &Schema{Type: "string", Enum: []any{"one", "two"}}},
		},
		{
			name:       "Set object",
			schema:     &Schema{},
			comment:    "# @schema minProperties:1;maxProperties:10;additionalProperties:false;$id:https://example.com/schema;$ref:schema/product.json",
			wantSchema: &Schema{MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), AdditionalProperties: SchemaFalse(), ID: "https://example.com/schema", Ref: "schema/product.json"},
		},
		{
			name:       "Set additionalProperties object",
			schema:     &Schema{},
			comment:    "# @schema additionalProperties:{\"type\":\"string\"}",
			wantSchema: &Schema{AdditionalProperties: &Schema{Type: "string"}},
		},
		{
			name:       "Set additionalProperties bool empty",
			schema:     &Schema{},
			comment:    "# @schema additionalProperties",
			wantSchema: &Schema{AdditionalProperties: SchemaTrue()},
		},
		{
			name:       "Set meta-data",
			schema:     &Schema{},
			comment:    "# @schema title:My Title;description: some description;readOnly:false;default:\"foo\";const:\"foo\"",
			wantSchema: &Schema{Title: "My Title", Description: "some description", ReadOnly: false, Default: "foo", Const: "foo"},
		},
		{
			name:       "Set skipProperties",
			schema:     &Schema{},
			comment:    "# @schema skipProperties:true;unevaluatedProperties:false",
			wantSchema: &Schema{SkipProperties: true, UnevaluatedProperties: boolPtr(false)},
		},
		{
			name:       "Set hidden",
			schema:     &Schema{},
			comment:    "# @schema hidden:true",
			wantSchema: &Schema{Hidden: true},
		},
		{
			name:       "Set and unset hidden",
			schema:     &Schema{},
			comment:    "# @schema hidden:true; hidden:false",
			wantSchema: &Schema{Hidden: false},
		},
		{
			name:       "Set required",
			schema:     &Schema{},
			comment:    "# @schema required:true",
			wantSchema: &Schema{RequiredByParent: true},
		},
		{
			name:       "Set and unset required",
			schema:     &Schema{},
			comment:    "# @schema required:true; required:false",
			wantSchema: &Schema{RequiredByParent: false},
		},
		{
			name:       "Set allOf",
			schema:     &Schema{},
			comment:    "# @schema allOf:[{\"type\":\"string\"}]",
			wantSchema: &Schema{AllOf: []*Schema{{Type: "string"}}},
		},
		{
			name:       "Set anyOf",
			schema:     &Schema{},
			comment:    "# @schema anyOf:[{\"type\":\"string\"}]",
			wantSchema: &Schema{AnyOf: []*Schema{{Type: "string"}}},
		},
		{
			name:       "Set oneOf",
			schema:     &Schema{},
			comment:    "# @schema oneOf:[{\"type\":\"string\"}]",
			wantSchema: &Schema{OneOf: []*Schema{{Type: "string"}}},
		},
		{
			name:       "Set not JSON",
			schema:     &Schema{},
			comment:    "# @schema not:{\"type\":\"string\"}",
			wantSchema: &Schema{Not: &Schema{Type: "string"}},
		},
		{
			name:       "Set examples",
			schema:     &Schema{},
			comment:    "# @schema examples:[foo, bar]",
			wantSchema: &Schema{Examples: []any{"foo", "bar"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processComment(tt.schema, []string{tt.comment})
			require.NoError(t, err)
			assert.Equal(t, tt.wantSchema, tt.schema)
		})
	}
}

func TestProcessComment_Error(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		wantErr string
	}{
		{name: "unknown annotation", comment: "# @schema foobar: 123", wantErr: "unknown annotation \"foobar\""},

		{name: "required invalid bool", comment: "# @schema required: foo", wantErr: "required: invalid boolean"},
		{name: "readOnly invalid bool", comment: "# @schema readOnly: foo", wantErr: "readOnly: invalid boolean"},
		{name: "hidden invalid bool", comment: "# @schema hidden: foo", wantErr: "hidden: invalid boolean"},
		{name: "required invalid bool", comment: "# @schema required: foo", wantErr: "required: invalid boolean"},
		{name: "uniqueItems invalid bool", comment: "# @schema uniqueItems: foo", wantErr: "uniqueItems: invalid boolean"},
		{name: "skipProperties invalid bool", comment: "# @schema skipProperties: foo", wantErr: "skipProperties: invalid boolean"},
		{name: "unevaluatedProperties invalid bool", comment: "# @schema unevaluatedProperties: foo", wantErr: "unevaluatedProperties: invalid boolean"},

		{name: "maxLength invalid uint64", comment: "# @schema maxLength: foo", wantErr: "maxLength: invalid integer"},
		{name: "minLength invalid uint64", comment: "# @schema minLength: foo", wantErr: "minLength: invalid integer"},
		{name: "maxItems invalid uint64", comment: "# @schema maxItems: foo", wantErr: "maxItems: invalid integer"},
		{name: "minItems invalid uint64", comment: "# @schema minItems: foo", wantErr: "minItems: invalid integer"},
		{name: "maxProperties invalid uint64", comment: "# @schema maxProperties: foo", wantErr: "maxProperties: invalid integer"},
		{name: "minProperties invalid uint64", comment: "# @schema minProperties: foo", wantErr: "minProperties: invalid integer"},

		{name: "multipleOf invalid float64", comment: "# @schema multipleOf: foo", wantErr: "multipleOf: invalid number"},
		{name: "multipleOf zero", comment: "# @schema multipleOf: 0", wantErr: "multipleOf: must be greater than zero"},
		{name: "minimum invalid float64", comment: "# @schema minimum: foo", wantErr: "minimum: invalid number"},
		{name: "maximum invalid float64", comment: "# @schema maximum: foo", wantErr: "maximum: invalid number"},

		{name: "patternProperties invalid YAML", comment: "# @schema patternProperties: {", wantErr: "patternProperties: parse object \"{\": yaml"},
		{name: "default invalid YAML", comment: "# @schema default: {", wantErr: "default: parse object \"{\": yaml"},
		{name: "itemProperties invalid YAML", comment: "# @schema itemProperties: {", wantErr: "itemProperties: parse object \"{\": yaml"},
		{name: "additionalProperties invalid YAML", comment: "# @schema additionalProperties: {", wantErr: "additionalProperties: parse object \"{\": yaml"},
		{name: "allOf invalid YAML", comment: "# @schema allOf: {", wantErr: "allOf: parse object \"{\": yaml"},
		{name: "anyOf invalid YAML", comment: "# @schema anyOf: {", wantErr: "anyOf: parse object \"{\": yaml"},
		{name: "oneOf invalid YAML", comment: "# @schema oneOf: {", wantErr: "oneOf: parse object \"{\": yaml"},
		{name: "not invalid YAML", comment: "# @schema not: {", wantErr: "not: parse object \"{\": yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schema Schema
			err := processComment(&schema, []string{tt.comment})
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestProcessObjectComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    *Schema
		wantErr string
	}{
		{name: "empty", comment: "", wantErr: "parse object \"\": missing value"},
		{name: "empty object", comment: "{}", want: &Schema{}},
		{name: "null", comment: "null", want: nil},
		{name: "JSON syntax", comment: "{\"type\":\"string\"}", want: &Schema{Type: "string"}},
		{name: "YAML syntax", comment: "{type: string}", want: &Schema{Type: "string"}},
		{name: "invalid field", comment: "{\"readOnly\": \"foobar\"}", wantErr: "parse object \"{\\\"readOnly\\\": \\\"foobar\\\"}\": yaml: unmarshal errors:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Comment: %q", tt.comment)
			var got *Schema
			err := processObjectComment(&got, tt.comment)
			if tt.wantErr != "" {
				t.Logf("Unexpected value: %#v", got)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}

	t.Run("overrides instead of merges", func(t *testing.T) {
		schema := &Schema{
			Minimum: float64Ptr(123),
		}
		err := processObjectComment(&schema, "{\"type\": \"string\"}")
		require.NoError(t, err)

		want := &Schema{
			Type: "string",
			// we don't want "Minimum" to stick around
		}
		require.Equal(t, want, schema)
	})
}

func TestProcessBoolComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    bool
		wantErr string
	}{
		{name: "empty", comment: "", want: true},
		{name: "only spacing", comment: "  \t ", want: true},
		{name: "true", comment: "true", want: true},
		{name: "true with spacing", comment: " \t  true  \t ", want: true},
		{name: "true uppercase", comment: "TRUE", wantErr: "invalid boolean \"TRUE\", must be \"true\" or \"false\""},
		{name: "false", comment: "false", want: false},
		{name: "false with spacing", comment: " \t  false \t ", want: false},
		{name: "false uppercase", comment: "FALSE", wantErr: "invalid boolean \"FALSE\", must be \"true\" or \"false\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Comment: %q", tt.comment)
			var got bool
			err := processBoolComment(&got, tt.comment)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestProcessUint64PtrComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    *uint64
		wantErr string
	}{
		{name: "empty", comment: "", wantErr: "invalid integer \"\": invalid syntax"},
		{name: "only spacing", comment: "  \t ", wantErr: "invalid integer \"\": invalid syntax"},
		{name: "null", comment: "null", want: nil},
		{name: "integer", comment: "123", want: uint64Ptr(123)},
		{name: "negative integer", comment: "-123", wantErr: "invalid integer \"-123\": negative values not allowed"},
		{name: "float", comment: "1.23", wantErr: "invalid integer \"1.23\": invalid syntax"},
		{name: "hex", comment: "0x123", wantErr: "invalid integer \"0x123\": invalid syntax"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startValues := []struct {
				name  string
				value *uint64
			}{
				{name: "overriding nil", value: nil},
				{name: "overriding value", value: uint64Ptr(123)},
			}

			for _, startVal := range startValues {
				t.Run(startVal.name, func(t *testing.T) {
					t.Logf("Comment: %q", tt.comment)

					got := startVal.value
					err := processUint64PtrComment(&got, tt.comment)
					if tt.wantErr != "" {
						require.ErrorContains(t, err, tt.wantErr)
					} else {
						require.NoError(t, err)
						require.Equal(t, tt.want, got)
					}
				})
			}
		})
	}
}
