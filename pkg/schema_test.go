package pkg

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGetKind(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "Boolean true",
			value:    "true",
			expected: "boolean",
		},
		{
			name:     "Boolean false",
			value:    "false",
			expected: "boolean",
		},
		{
			name:     "Integer zero",
			value:    "0",
			expected: "integer",
		},
		{
			name:     "Positive integer",
			value:    "123",
			expected: "integer",
		},
		{
			name:     "Negative integer",
			value:    "-123",
			expected: "integer",
		},
		{
			name:     "Float",
			value:    "123.456",
			expected: "number",
		},
		{
			name:     "Float with exponent",
			value:    "5e7",
			expected: "number",
		},
		{
			name:     "Non-empty string",
			value:    "test",
			expected: "string",
		},
		{
			name:     "Empty string",
			value:    "",
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getKind(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSchemaURL(t *testing.T) {
	tests := []struct {
		name        string
		draft       int
		expectedURL string
		expectedErr error
	}{
		{
			name:        "Draft 4",
			draft:       4,
			expectedURL: "http://json-schema.org/draft-04/schema#",
			expectedErr: nil,
		},
		{
			name:        "Draft 6",
			draft:       6,
			expectedURL: "http://json-schema.org/draft-06/schema#",
			expectedErr: nil,
		},
		{
			name:        "Draft 7",
			draft:       7,
			expectedURL: "http://json-schema.org/draft-07/schema#",
			expectedErr: nil,
		},
		{
			name:        "Draft 2019",
			draft:       2019,
			expectedURL: "https://json-schema.org/draft/2019-09/schema",
			expectedErr: nil,
		},
		{
			name:        "Draft 2020",
			draft:       2020,
			expectedURL: "https://json-schema.org/draft/2020-12/schema",
			expectedErr: nil,
		},
		{
			name:        "Invalid Draft",
			draft:       5,
			expectedURL: "",
			expectedErr: errors.New("invalid draft version. Please use one of: 4, 6, 7, 2019, 2020"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := getSchemaURL(tt.draft)
			if url != tt.expectedURL {
				t.Errorf("getSchemaURL(%d) got URL = %s, want %s", tt.draft, url, tt.expectedURL)
			}
			if (err != nil && tt.expectedErr == nil) || (err == nil && tt.expectedErr != nil) {
				t.Errorf("getSchemaURL(%d) got err = %v, want %v", tt.draft, err, tt.expectedErr)
			}
			if err != nil && tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("getSchemaURL(%d) got err = %v, want %v", tt.draft, err, tt.expectedErr)
			}
		})
	}
}

func TestGetComment(t *testing.T) {
	tests := []struct {
		name            string
		keyNode         *yaml.Node
		valNode         *yaml.Node
		expectedComment string
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
			expectedComment: "# Value comment",
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
			expectedComment: "# Key comment",
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
			expectedComment: "# Key comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := getComment(tt.keyNode, tt.valNode)
			assert.Equal(t, tt.expectedComment, comment, "getComment returned an unexpected comment")
		})
	}
}

func TestProcessList(t *testing.T) {
	tests := []struct {
		name         string
		comment      string
		stringsOnly  bool
		expectedList []interface{}
	}{
		{
			name:         "empty list",
			comment:      "[]",
			stringsOnly:  false,
			expectedList: []interface{}{""},
		},
		{
			name:         "single string",
			comment:      "[\"value\"]",
			stringsOnly:  true,
			expectedList: []interface{}{"value"},
		},
		{
			name:         "single string without quotes",
			comment:      "[value]",
			stringsOnly:  true,
			expectedList: []interface{}{"value"},
		},
		{
			name:         "multiple strings",
			comment:      "[\"value1\", \"value2\"]",
			stringsOnly:  true,
			expectedList: []interface{}{"value1", "value2"},
		},
		{
			name:         "null allowed",
			comment:      "[null]",
			stringsOnly:  false,
			expectedList: []interface{}{nil},
		},
		{
			name:         "null not treated as special",
			comment:      "[null]",
			stringsOnly:  true,
			expectedList: []interface{}{"null"},
		},
		{
			name:         "mixed strings and null",
			comment:      "[\"value1\", null, \"value2\"]",
			stringsOnly:  false,
			expectedList: []interface{}{"value1", nil, "value2"},
		},
		{
			name:         "whitespace trimming",
			comment:      "[ value1, value2 ]",
			stringsOnly:  true,
			expectedList: []interface{}{"value1", "value2"},
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
			expectedSchema:   &Schema{MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), AdditionalProperties: boolPtr(false), ID: "https://example.com/schema", Ref: "schema/product.json"},
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
			expectedSchema:   &Schema{AllOf: []any{map[string]any{"type": "string"}}},
			expectedRequired: false,
		},
		{
			name:             "Set anyOf",
			schema:           &Schema{},
			comment:          "# @schema anyOf:[{\"type\":\"string\"}]",
			expectedSchema:   &Schema{AnyOf: []any{map[string]any{"type": "string"}}},
			expectedRequired: false,
		},
		{
			name:             "Set oneOf",
			schema:           &Schema{},
			comment:          "# @schema oneOf:[{\"type\":\"string\"}]",
			expectedSchema:   &Schema{OneOf: []any{map[string]any{"type": "string"}}},
			expectedRequired: false,
		},
		{
			name:             "Set not",
			schema:           &Schema{},
			comment:          "# @schema not:[{\"type\":\"string\"}]",
			expectedSchema:   &Schema{Not: []any{map[string]any{"type": "string"}}},
			expectedRequired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var required bool
			processComment(tt.schema, tt.comment)
			assert.Equal(t, tt.expectedSchema, tt.schema)
			assert.Equal(t, tt.expectedRequired, required)
		})
	}
}

func TestParseNode(t *testing.T) {
	tests := []struct {
		name          string
		keyNode       *yaml.Node
		valNode       *yaml.Node
		expectedType  string
		expectedProps map[string]*Schema
		expectedItems *Schema
		expectedReq   []string
		isRequired    bool
	}{
		{
			name: "parse object node",
			valNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key"},
					{Kind: yaml.ScalarNode, Value: "value"},
				},
			},
			expectedType:  "object",
			expectedProps: map[string]*Schema{"key": {Type: "string"}},
			expectedReq:   nil,
		},
		{
			name: "parse array node",
			valNode: &yaml.Node{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "value"},
				},
			},
			expectedType:  "array",
			expectedItems: &Schema{Type: "string"},
		},
		{
			name: "parse scalar node",
			valNode: &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "value",
				Style: yaml.DoubleQuotedStyle,
			},
			expectedType: "string",
		},
		{
			name: "parse object node with skipProperties:true",
			valNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Value: "key",
					},
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: "nestedKey"},
							{Kind: yaml.ScalarNode, Value: "nestedValue"},
						},
						LineComment: "# @schema skipProperties:true",
					},
				},
			},
			expectedType:  "object",
			expectedProps: map[string]*Schema{"key": {Type: "object", Properties: nil, SkipProperties: true}},
			expectedReq:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, isRequired := parseNode(tt.keyNode, tt.valNode)
			assert.Equal(t, tt.expectedType, schema.Type)
			assert.Equal(t, tt.expectedProps, schema.Properties)
			assert.Equal(t, tt.expectedItems, schema.Items)
			assert.Equal(t, tt.expectedReq, schema.Required)
			assert.Equal(t, tt.isRequired, isRequired)
		})
	}
}
