package pkg

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSchemaKindString(t *testing.T) {
	tests := []struct {
		name string
		kind SchemaKind
		want string
	}{
		{name: "zero", kind: SchemaKind(0), want: "object"},
		{name: "object", kind: SchemaKindObject, want: "object"},
		{name: "true", kind: SchemaKindTrue, want: "true"},
		{name: "false", kind: SchemaKindFalse, want: "false"},
		{name: "undefined", kind: SchemaKind(123), want: "SchemaKind(123)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kind.String(), "SchemaKind.String()")
			assert.Equal(t, tt.want, tt.kind.GoString(), "SchemaKind.GoString()")
		})
	}
}

func TestSchemaKindIsBool(t *testing.T) {
	tests := []struct {
		name string
		kind SchemaKind
		want bool
	}{
		{name: "zero", kind: SchemaKind(0), want: false},
		{name: "object", kind: SchemaKindObject, want: false},
		{name: "true", kind: SchemaKindTrue, want: true},
		{name: "false", kind: SchemaKindFalse, want: true},
		{name: "undefined", kind: SchemaKind(123), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kind.IsBool())
		})
	}
}

func TestSchemaIsZero(t *testing.T) {
	var (
		exampleUint64      = uint64(1)
		exampleFloat64     = float64(1)
		exampleString      = "foo"
		exampleBool        = true
		exampleSchema      = &Schema{ID: exampleString}
		exampleMap         = map[string]*Schema{exampleString: exampleSchema}
		exampleSchemaSlice = []*Schema{exampleSchema}
		exampleAnySlice    = []any{exampleString}
	)
	tests := []struct {
		name   string
		schema *Schema
		want   bool
	}{
		{name: "nil", schema: nil, want: true},
		{name: "empty", schema: &Schema{}, want: true},
		{name: "SkipProperties", schema: &Schema{SkipProperties: true}, want: true},
		{name: "Hidden", schema: &Schema{Hidden: true}, want: true},

		{name: "Kind", schema: &Schema{kind: SchemaKindTrue}},
		{name: "Schema", schema: &Schema{Schema: exampleString}},
		{name: "ID", schema: &Schema{ID: exampleString}},
		{name: "Title", schema: &Schema{Title: exampleString}},
		{name: "Description", schema: &Schema{Description: exampleString}},
		{name: "Comment", schema: &Schema{Comment: exampleString}},
		{name: "ReadOnly", schema: &Schema{ReadOnly: exampleBool}},
		{name: "Default", schema: &Schema{Default: exampleString}},
		{name: "Ref", schema: &Schema{Ref: exampleString}},
		{name: "Type", schema: &Schema{Type: exampleString}},
		{name: "Enum", schema: &Schema{Enum: exampleAnySlice}},
		{name: "AllOf", schema: &Schema{AllOf: exampleSchemaSlice}},
		{name: "AnyOf", schema: &Schema{AnyOf: exampleSchemaSlice}},
		{name: "OneOf", schema: &Schema{OneOf: exampleSchemaSlice}},
		{name: "Not", schema: &Schema{Not: &Schema{ID: exampleString}}},
		{name: "Maximum", schema: &Schema{Maximum: &exampleFloat64}},
		{name: "Minimum", schema: &Schema{Minimum: &exampleFloat64}},
		{name: "MultipleOf", schema: &Schema{MultipleOf: &exampleFloat64}},
		{name: "Pattern", schema: &Schema{Pattern: exampleString}},
		{name: "MaxLength", schema: &Schema{MaxLength: &exampleUint64}},
		{name: "MinLength", schema: &Schema{MinLength: &exampleUint64}},
		{name: "MaxItems", schema: &Schema{MaxItems: &exampleUint64}},
		{name: "MinItems", schema: &Schema{MinItems: &exampleUint64}},
		{name: "UniqueItems", schema: &Schema{UniqueItems: exampleBool}},
		{name: "Items", schema: &Schema{Items: &Schema{ID: exampleString}}},
		{name: "AdditionalItems", schema: &Schema{AdditionalItems: &Schema{ID: exampleString}}},
		{name: "Required", schema: &Schema{Required: []string{exampleString}}},
		{name: "MaxProperties", schema: &Schema{MaxProperties: &exampleUint64}},
		{name: "MinProperties", schema: &Schema{MinProperties: &exampleUint64}},
		{name: "Properties", schema: &Schema{Properties: exampleMap}},
		{name: "PatternProperties", schema: &Schema{PatternProperties: exampleMap}},
		{name: "AdditionalProperties", schema: &Schema{AdditionalProperties: exampleSchema}},
		{name: "UnevaluatedProperties", schema: &Schema{UnevaluatedProperties: &exampleBool}},
		{name: "Defs", schema: &Schema{Defs: exampleMap}},
		{name: "Definitions", schema: &Schema{Definitions: exampleMap}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.schema.IsZero()
			if got != tt.want {
				t.Errorf("wrong result\nwant: %t\ngot:  %t", tt.want, got)
			}
		})
	}
}

func TestSchemaJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		want     *Schema
		wantKind SchemaKind
	}{
		{
			name:     "null",
			json:     `null`,
			want:     nil,
			wantKind: SchemaKindObject,
		},
		{
			name:     "true",
			json:     `true`,
			want:     &SchemaTrue,
			wantKind: SchemaKindTrue,
		},
		{
			name:     "false",
			json:     `false`,
			want:     &SchemaFalse,
			wantKind: SchemaKindFalse,
		},
		{
			name: "object",
			json: `{"$id": "hello there"}`,
			want: &Schema{
				kind: SchemaKindObject,
				ID:   "hello there",
			},
			wantKind: SchemaKindObject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Schema *Schema `json:"schema"`
			}
			err := json.Unmarshal([]byte(`{"schema":`+tt.json+`}`), &result)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result.Schema)
			assert.Equal(t, tt.wantKind, result.Schema.Kind())
		})
	}
}

func TestSchemaJSONMarshal(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		want   string
	}{
		{
			name:   "null",
			schema: nil,
			want:   `{"schema": null}`,
		},
		{
			name:   "true",
			schema: &SchemaTrue,
			want:   `{"schema": true}`,
		},
		{
			name:   "false",
			schema: &SchemaFalse,
			want:   `{"schema": false}`,
		},
		{
			name: "object",
			schema: &Schema{
				kind: SchemaKindObject,
				ID:   "hello there",
			},
			want: `{"schema": {"$id": "hello there"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := struct {
				Schema *Schema `json:"schema"`
			}{
				Schema: tt.schema,
			}
			b, err := json.Marshal(obj)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(b))
		})
	}
}

func TestSchemaYAMLUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want *Schema
	}{
		{
			name: "null",
			yaml: ` null `,
			want: nil,
		},
		{
			name: "true",
			yaml: ` true `,
			want: &SchemaTrue,
		},
		{
			name: "false",
			yaml: ` false `,
			want: &SchemaFalse,
		},
		{
			name: "object",
			yaml: `{"$id": "hello there"}`,
			want: &Schema{
				kind: SchemaKindObject,
				ID:   "hello there",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Schema *Schema `yaml:"schema"`
			}
			err := yaml.Unmarshal([]byte(`{"schema":`+tt.yaml+`}`), &result)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result.Schema)
		})
	}
}

func TestSchemaYAMLUnmarshal_error(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "bool",
			yaml:    `!!bool not a bool`,
			wantErr: "cannot decode !!str `not a bool` as a !!bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Schema *Schema `yaml:"schema"`
			}
			err := yaml.Unmarshal([]byte(`{"schema":`+tt.yaml+`}`), &result)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestSchemaYAMLMarshal(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		want   string
	}{
		{
			name:   "null",
			schema: nil,
			want:   `schema: null`,
		},
		{
			name:   "true",
			schema: &SchemaTrue,
			want:   `schema: true`,
		},
		{
			name:   "false",
			schema: &SchemaFalse,
			want:   `schema: false`,
		},
		{
			name: "object",
			schema: &Schema{
				kind: SchemaKindObject,
				ID:   "hello there",
			},
			want: `schema: {"$id": "hello there"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := struct {
				Schema *Schema `yaml:"schema"`
			}{
				Schema: tt.schema,
			}
			b, err := yaml.Marshal(obj)
			require.NoError(t, err)
			assert.YAMLEq(t, tt.want, string(b))
		})
	}
}

func TestSchemaSetKind(t *testing.T) {
	tests := []struct {
		name   string
		kind   SchemaKind
		schema *Schema
		want   *Schema
	}{
		{
			name:   "set true resets other fields",
			kind:   SchemaKindTrue,
			schema: &Schema{kind: SchemaKindTrue, ID: "foobar"},
			want:   &Schema{kind: SchemaKindTrue},
		},
		{
			name:   "set false resets other fields",
			kind:   SchemaKindFalse,
			schema: &Schema{kind: SchemaKindFalse, ID: "foobar"},
			want:   &Schema{kind: SchemaKindFalse},
		},
		{
			name:   "set object keeps other fields",
			kind:   SchemaKindObject,
			schema: &Schema{kind: SchemaKindTrue, ID: "foobar"},
			want:   &Schema{kind: SchemaKindObject, ID: "foobar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.schema.SetKind(tt.kind)
			assert.Equal(t, tt.schema, tt.want)
		})
	}
}

func TestSchemaSetKind_panics(t *testing.T) {
	tests := []struct {
		name    string
		kind    SchemaKind
		schema  *Schema
		wantErr string
	}{
		{
			name:    "set nil",
			kind:    SchemaKindObject,
			schema:  nil,
			wantErr: "Schema.SetKind(object): method reciever must not be nil",
		},
		{
			name:    "set invalid kind",
			kind:    SchemaKind(123),
			schema:  &Schema{},
			wantErr: "Schema.SetKind(SchemaKind(123)): unexpected kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithError(t, tt.wantErr, func() { tt.schema.SetKind(tt.kind) })
		})
	}
}

func TestGetYAMLKind(t *testing.T) {
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
			result := getYAMLKind(tt.value)
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

func TestSchemaSubschemas_order(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
	}{
		{
			name: "items",
			schema: &Schema{
				Properties: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}},
				Items:      &Schema{ID: "c"},
			},
		},
		{
			name: "additionalItems",
			schema: &Schema{
				Properties:      map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}},
				AdditionalItems: &Schema{ID: "c"},
			},
		},
		{
			name: "properties",
			schema: &Schema{
				Properties: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "additionalProperties",
			schema: &Schema{
				Properties:           map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}},
				AdditionalProperties: &Schema{ID: "c"},
			},
		},
		{
			name: "patternProperties",
			schema: &Schema{
				PatternProperties: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "defs",
			schema: &Schema{
				Defs: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "definitions",
			schema: &Schema{
				Definitions: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "allOf",
			schema: &Schema{
				AllOf: []*Schema{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			},
		},
		{
			name: "anyOf",
			schema: &Schema{
				AnyOf: []*Schema{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			},
		},
		{
			name: "oneOf",
			schema: &Schema{
				OneOf: []*Schema{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			},
		},
		{
			name: "not",
			schema: &Schema{
				OneOf: []*Schema{{ID: "a"}, {ID: "b"}},
				Not:   &Schema{ID: "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to ensure we dont get lucky with the ordering
			for range 10 {
				var ids []string
				for _, sub := range tt.schema.Subschemas() {
					ids = append(ids, sub.ID)
					if len(ids) == 3 {
						break
					}
				}
				require.Equal(t, "abc", strings.Join(ids, ""))
			}
		})
	}
}
