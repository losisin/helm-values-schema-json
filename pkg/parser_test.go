package pkg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			dest: &Schema{Type: "array", Items: &Schema{Type: "string"}, AdditionalItems: &Schema{Type: "string"}, MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true},
			src:  &Schema{Type: "array", Items: &Schema{Type: "string"}, AdditionalItems: &Schema{Type: "string"}, MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true},
			want: &Schema{Type: "array", Items: &Schema{Type: "string"}, AdditionalItems: &Schema{Type: "string"}, MinItems: uint64Ptr(1), MaxItems: uint64Ptr(10), UniqueItems: true},
		},
		{
			name: "object properties",
			dest: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: &SchemaFalse, UnevaluatedProperties: boolPtr(false)},
			src:  &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: &SchemaFalse, UnevaluatedProperties: boolPtr(false)},
			want: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: &SchemaFalse, UnevaluatedProperties: boolPtr(false)},
		},
		{
			name: "meta-data properties",
			dest: &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", ID: "http://example.com/schema", Ref: "schema/product.json", Schema: "https://my-schema", Comment: "Lorem ipsum"},
			src:  &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", ID: "http://example.com/schema", Ref: "schema/product.json", Schema: "https://my-schema", Comment: "Lorem ipsum"},
			want: &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", ID: "http://example.com/schema", Ref: "schema/product.json", Schema: "https://my-schema", Comment: "Lorem ipsum"},
		},
		{
			name: "allOf",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", AllOf: []*Schema{{Type: "string"}}},
			want: &Schema{Type: "object", AllOf: []*Schema{{Type: "string"}}},
		},
		{
			name: "anyOf",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", AnyOf: []*Schema{{Type: "string"}}},
			want: &Schema{Type: "object", AnyOf: []*Schema{{Type: "string"}}},
		},
		{
			name: "oneOf",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", OneOf: []*Schema{{Type: "string"}}},
			want: &Schema{Type: "object", OneOf: []*Schema{{Type: "string"}}},
		},
		{
			name: "not",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", Not: &Schema{Type: "string"}},
			want: &Schema{Type: "object", Not: &Schema{Type: "string"}},
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

func TestEnsureCompliant(t *testing.T) {
	tests := []struct {
		name                   string
		schema                 *Schema
		noAdditionalProperties bool
		draft                  int
		want                   *Schema
		wantErr                string
	}{
		{
			name:   "nil schema",
			schema: nil,
		},

		{
			name:   "bool schema",
			schema: &SchemaTrue,
			want:   &SchemaTrue,
		},

		{
			name: "override additionalProperties",
			schema: &Schema{
				Type:                 "object",
				AdditionalProperties: nil,
			},
			noAdditionalProperties: true,
			want: &Schema{
				Type:                 "object",
				AdditionalProperties: &SchemaFalse,
			},
		},

		{
			name: "keep additionalProperties when not object",
			schema: &Schema{
				Type:                 "array",
				AdditionalProperties: &Schema{ID: "foo"},
			},
			noAdditionalProperties: true,
			want: &Schema{
				Type:                 "array",
				AdditionalProperties: &Schema{ID: "foo"},
			},
		},

		{
			name: "keep additionalProperties when config not enabled",
			schema: &Schema{
				Type:                 "object",
				AdditionalProperties: &Schema{ID: "foo"},
			},
			noAdditionalProperties: false,
			want: &Schema{
				Type:                 "object",
				AdditionalProperties: &Schema{ID: "foo"},
			},
		},

		{
			name:   "unset type when allOf",
			schema: &Schema{Type: "object", AllOf: []*Schema{{ID: "foo"}}},
			want:   &Schema{AllOf: []*Schema{{ID: "foo"}}},
		},
		{
			name:   "unset type when anyOf",
			schema: &Schema{Type: "object", AnyOf: []*Schema{{ID: "foo"}}},
			want:   &Schema{AnyOf: []*Schema{{ID: "foo"}}},
		},
		{
			name:   "unset type when oneOf",
			schema: &Schema{Type: "object", OneOf: []*Schema{{ID: "foo"}}},
			want:   &Schema{OneOf: []*Schema{{ID: "foo"}}},
		},
		{
			name:   "unset type when not",
			schema: &Schema{Type: "object", Not: &Schema{ID: "foo"}},
			want:   &Schema{Not: &Schema{ID: "foo"}},
		},

		{
			name:   "keep ref with other fields when draft 2019",
			schema: &Schema{Ref: "#", Type: "object"},
			draft:  2019,
			want:   &Schema{Ref: "#", Type: "object"},
		},
		{
			name:   "keep ref without fields when draft 7",
			schema: &Schema{Ref: "#"},
			draft:  7,
			want:   &Schema{Ref: "#"},
		},
		{
			name:   "change ref with other fields to allOf when draft 7",
			schema: &Schema{Ref: "#", Type: "object"},
			draft:  7,
			want: &Schema{
				AllOf: []*Schema{
					{Ref: "#"},
					{Type: "object"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.draft == 0 {
				tt.draft = 2020
			}
			err := ensureCompliant(tt.schema, tt.noAdditionalProperties, tt.draft)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)
		})
	}
}

func TestEnsureCompliant_recursive(t *testing.T) {
	recursiveSchema := &Schema{}
	recursiveSchema.Properties = map[string]*Schema{
		"circular": recursiveSchema,
	}

	tests := []struct {
		name   string
		schema *Schema
	}{
		{
			name:   "recursive items",
			schema: &Schema{Items: recursiveSchema},
		},
		{
			name:   "recursive properties",
			schema: &Schema{Properties: map[string]*Schema{"circular": recursiveSchema}},
		},
		{
			name:   "recursive patternProperties",
			schema: &Schema{PatternProperties: map[string]*Schema{"circular": recursiveSchema}},
		},
		{
			name:   "recursive defs",
			schema: &Schema{Defs: map[string]*Schema{"circular": recursiveSchema}},
		},
		{
			name:   "recursive definitions",
			schema: &Schema{Definitions: map[string]*Schema{"circular": recursiveSchema}},
		},
		{
			name:   "recursive allOf",
			schema: &Schema{AllOf: []*Schema{recursiveSchema}},
		},
		{
			name:   "recursive anyOf",
			schema: &Schema{AnyOf: []*Schema{recursiveSchema}},
		},
		{
			name:   "recursive oneOf",
			schema: &Schema{OneOf: []*Schema{recursiveSchema}},
		},
		{
			name:   "recursive not",
			schema: &Schema{Not: recursiveSchema},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureCompliant(tc.schema, false, 2020)
			assert.Error(t, err)
		})
	}
}

func TestUpdateRefK8sAlias(t *testing.T) {
	tests := []struct {
		name        string
		schema      *Schema
		urlTemplate string
		version     string
		wantErr     string
		want        *Schema
	}{
		{
			name:   "empty schema",
			schema: &Schema{},
			want:   &Schema{},
		},
		{
			name:        "with trailing slash",
			urlTemplate: "http://example.com/{{ .K8sSchemaVersion }}/",
			version:     "v1.2.3",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/foobar.json"},
			},
			want: &Schema{
				Items: &Schema{Ref: "http://example.com/v1.2.3/foobar.json"},
			},
		},
		{
			name:        "without trailing slash",
			urlTemplate: "http://example.com/{{ .K8sSchemaVersion }}",
			version:     "v1.2.3",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/foobar.json"},
			},
			want: &Schema{
				Items: &Schema{Ref: "http://example.com/v1.2.3/foobar.json"},
			},
		},
		{
			name:        "with fragment",
			urlTemplate: "http://example.com/{{ .K8sSchemaVersion }}",
			version:     "v1.2.3",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/foobar.json#/properties/foo"},
			},
			want: &Schema{
				Items: &Schema{Ref: "http://example.com/v1.2.3/foobar.json#/properties/foo"},
			},
		},

		{
			name:        "missing version",
			urlTemplate: "http://example.com/{{ .K8sSchemaVersion }}",
			version:     "",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/foobar.json#/properties/foo"},
			},
			wantErr: "/items: must set k8sSchemaVersion config",
		},
		{
			name:        "invalid template",
			urlTemplate: "http://example.com/{{",
			version:     "v1.2.3",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/foobar.json#/properties/foo"},
			},
			wantErr: "/items: parse k8sSchemaURL template: template: :1: unclosed action",
		},
		{
			name:        "invalid variable",
			urlTemplate: "http://example.com/{{ .Foobar }}",
			version:     "v1.2.3",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/foobar.json#/properties/foo"},
			},
			wantErr: "can't evaluate field Foobar in type",
		},
		{
			name:        "invalid ref",
			urlTemplate: "http://example.com/{{ .K8sSchemaVersion }}",
			version:     "v1.2.3",
			schema: &Schema{
				Items: &Schema{Ref: "$k8s/#/properties/foo"},
			},
			wantErr: "/items: invalid $k8s schema alias: must have a path but only got \"$k8s/#/properties/foo\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateRefK8sAlias(tt.schema, tt.urlTemplate, tt.version)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)
		})
	}
}
