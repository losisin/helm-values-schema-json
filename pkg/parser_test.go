package pkg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

// schemasEqual is a helper function to compare two Schema objects.
func schemasEqual(a, b *Schema) bool {
	if a == nil || b == nil {
		return a == b
	}
	// Compare simple fields
	if a.Type != b.Type || a.Pattern != b.Pattern || a.UniqueItems != b.UniqueItems || a.Title != b.Title || a.Description != b.Description || a.ReadOnly != b.ReadOnly {
		return false
	}
	// Compare pointer fields
	if !comparePointer(a.MultipleOf, b.MultipleOf) ||
		!comparePointer(a.Maximum, b.Maximum) ||
		!comparePointer(a.Minimum, b.Minimum) ||
		!comparePointer(a.MaxLength, b.MaxLength) ||
		!comparePointer(a.MinLength, b.MinLength) ||
		!comparePointer(a.MaxItems, b.MaxItems) ||
		!comparePointer(a.MinItems, b.MinItems) ||
		!comparePointer(a.MaxProperties, b.MaxProperties) ||
		!comparePointer(a.MinProperties, b.MinProperties) {
		return false
	}
	// Compare slice fields
	if !reflect.DeepEqual(a.Enum, b.Enum) || !reflect.DeepEqual(a.Required, b.Required) {
		return false
	}
	// Recursively check nested fields
	if !schemasEqual(a.Items, b.Items) {
		return false
	}
	// Compare map fields (Properties)
	return reflect.DeepEqual(a.Properties, b.Properties)
}

// comparePointer is a helper function for comparing pointer fields
func comparePointer[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func TestMergeSchemas(t *testing.T) {
	tests := []struct {
		name string
		dest *Schema
		src  *Schema
		want *Schema
	}{
		{
			name: "dest nil",
			dest: nil,
			src:  &Schema{Type: "string"},
			want: &Schema{Type: "string"},
		},
		{
			name: "src nil",
			dest: &Schema{Type: "string"},
			src:  nil,
			want: &Schema{Type: "string"},
		},
		{
			name: "both non-nil same type",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object"},
			want: &Schema{Type: "object"},
		},
		{
			name: "both non-nil different type",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "string"},
			want: &Schema{Type: "string"},
		},
		{
			name: "nested properties",
			dest: &Schema{Type: "object", Properties: map[string]*Schema{
				"foo": {Type: "integer"},
			}},
			src: &Schema{Type: "object", PatternProperties: map[string]*Schema{
				"^[a-z]$": {Type: "integer"},
			}},
			want: &Schema{Type: "object", Properties: map[string]*Schema{
				"foo": {Type: "integer"},
			}, PatternProperties: map[string]*Schema{
				"^[a-z]$": {Type: "integer"},
			}},
		},
		{
			name: "items",
			dest: &Schema{Type: "array", Items: &Schema{Type: "string"}},
			src:  &Schema{Type: "array", Items: &Schema{Type: "string"}},
			want: &Schema{Type: "array", Items: &Schema{Type: "string"}},
		},
		{
			name: "merge existing properties",
			dest: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"shared": {Type: "integer"},
				},
			},
			src: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"shared": {Type: "string", MinLength: uint64Ptr(1)},
				},
			},
			want: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"shared": {Type: "string", MinLength: uint64Ptr(1)},
				},
			},
		},
		{
			name: "merge multiple defs",
			dest: &Schema{
				Type: "object",
				Defs: map[string]*Schema{
					"file:a.json": {Type: "object"},
				},
			},
			src: &Schema{
				Type: "object",
				Defs: map[string]*Schema{
					"file:b.json": {Type: "integer"},
				},
			},
			want: &Schema{
				Type: "object",
				Defs: map[string]*Schema{
					"file:a.json": {Type: "object"},
					"file:b.json": {Type: "integer"},
				},
			},
		},
		{
			name: "merge existing defs",
			dest: &Schema{
				Type: "object",
				Defs: map[string]*Schema{
					"shared": {Type: "integer"},
				},
			},
			src: &Schema{
				Type: "object",
				Defs: map[string]*Schema{
					"shared": {Type: "string", MinLength: uint64Ptr(1)},
				},
			},
			want: &Schema{
				Type: "object",
				Defs: map[string]*Schema{
					"shared": {Type: "string", MinLength: uint64Ptr(1)},
				},
			},
		},
		{
			name: "numeric properties",
			dest: &Schema{Type: "integer", MultipleOf: float64Ptr(2), Minimum: float64Ptr(1), Maximum: float64Ptr(10)},
			src:  &Schema{Type: "integer", MultipleOf: float64Ptr(2), Minimum: float64Ptr(1), Maximum: float64Ptr(10)},
			want: &Schema{Type: "integer", MultipleOf: float64Ptr(2), Minimum: float64Ptr(1), Maximum: float64Ptr(10)},
		},
		{
			name: "string properties",
			dest: &Schema{Type: "string", Pattern: "^abc", MinLength: uint64Ptr(1), MaxLength: uint64Ptr(10)},
			src:  &Schema{Type: "string", Pattern: "^abc", MinLength: uint64Ptr(1), MaxLength: uint64Ptr(10)},
			want: &Schema{Type: "string", Pattern: "^abc", MinLength: uint64Ptr(1), MaxLength: uint64Ptr(10)},
		},
		{
			name: "array properties",
			dest: &Schema{Type: "array", Items: &Schema{Type: "string"}, MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true},
			src:  &Schema{Type: "array", Items: &Schema{Type: "string"}, MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true},
			want: &Schema{Type: "array", Items: &Schema{Type: "string"}, MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true},
		},
		{
			name: "object properties",
			dest: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: boolPtr(false), UnevaluatedProperties: boolPtr(false)},
			src:  &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: boolPtr(false), UnevaluatedProperties: boolPtr(false)},
			want: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: boolPtr(false), UnevaluatedProperties: boolPtr(false)},
		},
		{
			name: "meta-data properties",
			dest: &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", ID: "http://example.com/schema", Ref: "schema/product.json"},
			src:  &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", ID: "http://example.com/schema", Ref: "schema/product.json"},
			want: &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", ID: "http://example.com/schema", Ref: "schema/product.json"},
		},
		{
			name: "allOf",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", AllOf: []any{map[string]any{"type": "string"}}},
			want: &Schema{Type: "object", AllOf: []any{map[string]any{"type": "string"}}},
		},
		{
			name: "anyOf",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", AnyOf: []any{map[string]any{"type": "string"}}},
			want: &Schema{Type: "object", AnyOf: []any{map[string]any{"type": "string"}}},
		},
		{
			name: "oneOf",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", OneOf: []any{map[string]any{"type": "string"}}},
			want: &Schema{Type: "object", OneOf: []any{map[string]any{"type": "string"}}},
		},
		{
			name: "not",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", Not: []any{map[string]any{"type": "string"}}},
			want: &Schema{Type: "object", Not: []any{map[string]any{"type": "string"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSchemas(tt.dest, tt.src)
			if !schemasEqual(got, tt.want) {
				t.Errorf("mergeSchemas() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertSchemaToMap(t *testing.T) {
	tests := []struct {
		name    string
		schema  *Schema
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:   "nil schema",
			schema: nil,
			want:   nil,
		},
		{
			name: "with properties",
			schema: &Schema{
				Type:                  "object",
				MinProperties:         uint64Ptr(1),
				MaxProperties:         uint64Ptr(5),
				UnevaluatedProperties: boolPtr(false),
				Properties: map[string]*Schema{
					"foo": {
						Type: "string",
					},
				},
				Required: []string{"foo"},
				ID:       "http://example.com/schema",
				Ref:      "schema/product.json",
				AnyOf:    []any{map[string]any{"type": "string"}},
				Not:      []any{map[string]any{"type": "string"}},
			},
			want: map[string]interface{}{
				"minProperties":         uint64(1),
				"maxProperties":         uint64(5),
				"unevaluatedProperties": false,
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"foo"},
				"$id":      "http://example.com/schema",
				"$ref":     "schema/product.json",
				"anyOf":    []any{map[string]any{"type": "string"}},
				"not":      []any{map[string]any{"type": "string"}},
			},
		},
		{
			name: "with nested items",
			schema: &Schema{
				Type:        "array",
				Items:       &Schema{Type: "string", MinLength: uint64Ptr(1), MaxLength: uint64Ptr(10), AdditionalProperties: boolPtr(false)},
				MinItems:    uint64Ptr(1),
				MaxItems:    uint64Ptr(2),
				UniqueItems: true,
				ReadOnly:    true,
				AllOf:       []any{map[string]any{"type": "string"}},
			},
			want: map[string]interface{}{
				"items": map[string]interface{}{
					"type":                 "string",
					"minLength":            uint64(1),
					"maxLength":            uint64(10),
					"additionalProperties": false,
				},
				"minItems":    uint64(1),
				"maxItems":    uint64(2),
				"uniqueItems": true,
				"readOnly":    true,
				"allOf":       []any{map[string]any{"type": "string"}},
			},
		},
		{
			name: "with all scalar types",
			schema: &Schema{
				Type:        "integer",
				MultipleOf:  float64Ptr(3),
				Maximum:     float64Ptr(10),
				Minimum:     float64Ptr(1),
				Pattern:     "^abc",
				Title:       "My Title",
				Description: "some description",
				Enum:        []interface{}{1, 2, 3},
				Default:     "default",
				OneOf:       []any{map[string]any{"type": "string"}},
			},
			want: map[string]interface{}{
				"multipleOf":  3.0,
				"maximum":     10.0,
				"minimum":     1.0,
				"pattern":     "^abc",
				"title":       "My Title",
				"description": "some description",
				"enum":        []interface{}{1, 2, 3},
				"default":     "default",
				"oneOf":       []any{map[string]any{"type": "string"}},
			},
		},
		{
			name: "with defs",
			schema: &Schema{
				Type:                  "object",
				MinProperties:         uint64Ptr(1),
				MaxProperties:         uint64Ptr(5),
				UnevaluatedProperties: boolPtr(false),
				ID:                    "http://example.com/schema",
				Defs: map[string]*Schema{
					"foo": {
						ID:   "http://example.com/subschema",
						Type: "string",
						Properties: map[string]*Schema{
							"foo": &Schema{
								Type: "string",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"type":                  "object",
				"minProperties":         uint64(1),
				"maxProperties":         uint64(5),
				"unevaluatedProperties": false,
				"$id":                   "http://example.com/schema",
				"$defs": map[string]interface{}{
					"foo": map[string]interface{}{
						"$id":  "http://example.com/subschema",
						"type": "string",
						"properties": map[string]interface{}{
							"foo": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSchemaToMap(tt.schema, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertSchemaToMap()\nerror   %v\nwantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertSchemaToMap()\ngot  %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestConvertSchemaToMapFail(t *testing.T) {
	recursiveSchema := &Schema{}
	recursiveSchema.Properties = map[string]*Schema{
		"circular": recursiveSchema,
	}

	schemaWithItemsError := &Schema{
		Type:  "array",
		Items: recursiveSchema,
	}

	schemaWithPropertiesError := &Schema{
		Type:       "object",
		Properties: map[string]*Schema{"circular": recursiveSchema},
	}

	schemaWithPatternPropertiesError := &Schema{
		Type:              "object",
		PatternProperties: map[string]*Schema{"circular": recursiveSchema},
	}

	tests := []struct {
		name        string
		schema      *Schema
		expectedErr assert.ErrorAssertionFunc
	}{
		{
			name:        "Error with recursive items",
			schema:      schemaWithItemsError,
			expectedErr: assert.Error,
		},
		{
			name:        "Error with recursive properties",
			schema:      schemaWithPropertiesError,
			expectedErr: assert.Error,
		},
		{
			name:        "Error with recursive patternProperties",
			schema:      schemaWithPatternPropertiesError,
			expectedErr: assert.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := convertSchemaToMap(tc.schema, false)
			tc.expectedErr(t, err)
		})
	}
}
