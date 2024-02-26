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

// schemasEqual is a helper function to compare two Schema objects.
func schemasEqual(a, b *Schema) bool {
	if a == nil || b == nil {
		return a == b
	}
	// Compare simple fields
	if a.Type != b.Type || a.Pattern != b.Pattern || a.UniqueItems != b.UniqueItems || a.Title != b.Title || a.ReadOnly != b.ReadOnly {
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
			src: &Schema{Type: "object", Properties: map[string]*Schema{
				"bar": {Type: "string"},
			}},
			want: &Schema{Type: "object", Properties: map[string]*Schema{
				"foo": {Type: "integer"},
				"bar": {Type: "string"},
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
			dest: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10)},
			src:  &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10)},
			want: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10)},
		},
		{
			name: "meta-data properties",
			dest: &Schema{Type: "object", Title: "My Title", ReadOnly: true, Default: "default value"},
			src:  &Schema{Type: "object", Title: "My Title", ReadOnly: true, Default: "default value"},
			want: &Schema{Type: "object", Title: "My Title", ReadOnly: true, Default: "default value"},
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
				Type:          "object",
				MinProperties: uint64Ptr(1),
				MaxProperties: uint64Ptr(5),
				Properties: map[string]*Schema{
					"foo": {
						Type: "string",
					},
				},
				Required: []string{"foo"},
			},
			want: map[string]interface{}{
				"type":          "object",
				"minProperties": uint64(1),
				"maxProperties": uint64(5),
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"foo"},
			},
		},
		{
			name: "with nested items",
			schema: &Schema{
				Type:        "array",
				Items:       &Schema{Type: "string", MinLength: uint64Ptr(1), MaxLength: uint64Ptr(10)},
				MinItems:    uint64Ptr(1),
				MaxItems:    uint64Ptr(2),
				UniqueItems: true,
				ReadOnly:    true,
			},
			want: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":      "string",
					"minLength": uint64(1),
					"maxLength": uint64(10),
				},
				"minItems":    uint64(1),
				"maxItems":    uint64(2),
				"uniqueItems": true,
				"readOnly":    true,
			},
		},
		{
			name: "with all scalar types",
			schema: &Schema{
				Type:       "integer",
				MultipleOf: float64Ptr(3),
				Maximum:    float64Ptr(10),
				Minimum:    float64Ptr(1),
				Pattern:    "^abc",
				Title:      "My Title",
				Enum:       []interface{}{1, 2, 3},
				Default:    "default",
			},
			want: map[string]interface{}{
				"type":       "integer",
				"multipleOf": 3.0,
				"maximum":    10.0,
				"minimum":    1.0,
				"pattern":    "^abc",
				"title":      "My Title",
				"enum":       []interface{}{1, 2, 3},
				"default":    "default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSchemaToMap(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertSchemaToMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertSchemaToMap() got = %v, want %v", got, tt.want)
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := convertSchemaToMap(tc.schema)
			tc.expectedErr(t, err)
		})
	}
}
