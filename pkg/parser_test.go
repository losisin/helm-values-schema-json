package pkg

import (
	"testing"

	"github.com/losisin/helm-values-schema-json/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			dest: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: SchemaFalse(), UnevaluatedProperties: SchemaFalse()},
			src:  &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: SchemaFalse(), UnevaluatedProperties: SchemaFalse()},
			want: &Schema{Type: "object", MinProperties: uint64Ptr(1), MaxProperties: uint64Ptr(10), PatternProperties: map[string]*Schema{"^.$": {Type: "string"}}, AdditionalProperties: SchemaFalse(), UnevaluatedProperties: SchemaFalse()},
		},
		{
			name: "meta-data properties",
			dest: &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", Const: "const value", ID: "http://example.com/schema", Ref: "schema/product.json", Schema: "https://my-schema", Comment: "Old comment", Examples: []any{"foo", 1}},
			src:  &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", Const: "const value", ID: "http://example.com/schema", Ref: "schema/product.json", Schema: "https://my-schema", Comment: "New comment", Examples: []any{"bar"}},
			want: &Schema{Type: "object", Title: "My Title", Description: "My description", ReadOnly: true, Default: "default value", Const: "const value", ID: "http://example.com/schema", Ref: "schema/product.json", Schema: "https://my-schema", Comment: "New comment", Examples: []any{"bar"}},
		},
		{
			name: "vocabulary",
			dest: &Schema{Vocabulary: map[string]bool{"a": true, "b": false, "c": true}},
			src:  &Schema{Vocabulary: map[string]bool{"b": true, "c": false, "d": true}},
			want: &Schema{Vocabulary: map[string]bool{"a": true, "b": true, "c": false, "d": true}},
		},
		{
			name: "vocabulary nil",
			dest: &Schema{Vocabulary: nil},
			src:  &Schema{Vocabulary: map[string]bool{"a": true}},
			want: &Schema{Vocabulary: map[string]bool{"a": true}},
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
		{
			name: "if",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", If: &Schema{Type: "string"}},
			want: &Schema{Type: "object", If: &Schema{Type: "string"}},
		},
		{
			name: "then",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", Then: &Schema{Type: "string"}},
			want: &Schema{Type: "object", Then: &Schema{Type: "string"}},
		},
		{
			name: "else",
			dest: &Schema{Type: "object"},
			src:  &Schema{Type: "object", Else: &Schema{Type: "string"}},
			want: &Schema{Type: "object", Else: &Schema{Type: "string"}},
		},
		{
			name: "refReferrer",
			dest: &Schema{Ref: "foo", RefReferrer: ReferrerDir("/foo")},
			src:  &Schema{Ref: "foobar", RefReferrer: ReferrerDir("/foo/bar")},
			want: &Schema{Ref: "foobar", RefReferrer: ReferrerDir("/foo/bar")},
		},
		{
			name: "dynamicRefReferrer",
			dest: &Schema{DynamicRef: "foo", DynamicRefReferrer: ReferrerDir("/foo")},
			src:  &Schema{DynamicRef: "foobar", DynamicRefReferrer: ReferrerDir("/foo/bar")},
			want: &Schema{DynamicRef: "foobar", DynamicRefReferrer: ReferrerDir("/foo/bar")},
		},
		{
			name: "contains",
			dest: &Schema{Contains: &Schema{ID: "dest"}, MinContains: uint64Ptr(1), MaxLength: uint64Ptr(2)},
			src:  &Schema{Contains: &Schema{ID: "src"}, MinContains: uint64Ptr(10), MaxLength: uint64Ptr(20)},
			want: &Schema{Contains: &Schema{ID: "src"}, MinContains: uint64Ptr(10), MaxLength: uint64Ptr(20)},
		},
		{
			name: "prefixItems",
			dest: &Schema{PrefixItems: []*Schema{{ID: "dest"}}},
			src:  &Schema{PrefixItems: []*Schema{{ID: "src"}}},
			want: &Schema{PrefixItems: []*Schema{{ID: "src"}}},
		},
		{
			name: "RequiredByParent",
			dest: &Schema{RequiredByParent: false},
			src:  &Schema{RequiredByParent: true},
			want: &Schema{RequiredByParent: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSchemas(tt.dest, tt.src)
			testutil.Equal(t, tt.want, got)
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
			schema: SchemaTrue(),
			want:   SchemaTrue(),
		},

		{
			name:    "invalid type string",
			schema:  &Schema{Type: "foobar"},
			wantErr: "/type: invalid type \"foobar\", must be one of: array, boolean, integer, null, number, object, string",
		},
		{
			name:    "invalid type string in array",
			schema:  &Schema{Type: []any{"object", "foobar"}},
			wantErr: "/type/1: invalid type \"foobar\", must be one of: array, boolean, integer, null, number, object, string",
		},
		{
			name:    "duplicate type",
			schema:  &Schema{Type: []any{"string", "string"}},
			wantErr: "/type/1: type list must be unique, but found \"string\" multiple times",
		},
		{
			name:    "invalid type array",
			schema:  &Schema{Type: []any{[]any{}}},
			wantErr: "/type/0: type list must only contain strings",
		},
		{
			name:    "invalid type value",
			schema:  &Schema{Type: true},
			wantErr: "/type: type only be string or array of strings",
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
				AdditionalProperties: SchemaFalse(),

				// accidentally testing the "add default global" code here too
				Properties: map[string]*Schema{"global": defaultGlobal()},
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
					{Type: "object"},
					{Ref: "#"},
				},
			},
		},
		{
			name: "preserve $defs at root when wrapping ref with allOf for draft 7",
			schema: &Schema{
				Ref:  "#",
				Type: "object",
				Defs: map[string]*Schema{
					"foo": {Type: "string"},
					"bar": {Type: "integer"},
				},
			},
			draft: 7,
			want: &Schema{
				AllOf: []*Schema{
					{Type: "object"},
					{Ref: "#"},
				},
				Defs: map[string]*Schema{
					"foo": {Type: "string"},
					"bar": {Type: "integer"},
				},
			},
		},
		{
			name: "preserve definitions at root when wrapping ref with allOf for draft 7",
			schema: &Schema{
				Ref:  "foo.json",
				Type: "object",
				Definitions: map[string]*Schema{
					"myDef": {Type: "boolean"},
				},
			},
			draft: 7,
			want: &Schema{
				AllOf: []*Schema{
					{Type: "object"},
					{Ref: "foo.json"},
				},
				Definitions: map[string]*Schema{
					"myDef": {Type: "boolean"},
				},
			},
		},
		{
			name: "preserve both $defs and definitions at root when wrapping ref with allOf for draft 7",
			schema: &Schema{
				Ref:  "bar.json",
				Type: "object",
				Properties: map[string]*Schema{
					"name": {Type: "string"},
				},
				Defs: map[string]*Schema{
					"foo": {Type: "string"},
				},
				Definitions: map[string]*Schema{
					"myDef": {Type: "boolean"},
				},
			},
			draft: 7,
			want: &Schema{
				AllOf: []*Schema{
					{
						Type: "object",
						Properties: map[string]*Schema{
							"name": {Type: "string"},
						},
					},
					{Ref: "bar.json"},
				},
				Defs: map[string]*Schema{
					"foo": {Type: "string"},
				},
				Definitions: map[string]*Schema{
					"myDef": {Type: "boolean"},
				},
			},
		},
		{
			name: "update internal references when wrapping for draft 7",
			schema: &Schema{
				Ref:  "external.json",
				Type: "object",
				Properties: map[string]*Schema{
					"foo": {Type: "string"},
					"bar": {Ref: "#/properties/foo"},
				},
			},
			draft: 7,
			want: &Schema{
				AllOf: []*Schema{
					{
						Type: "object",
						Properties: map[string]*Schema{
							"foo": {Type: "string"},
							"bar": {Ref: "#/allOf/0/properties/foo"},
						},
					},
					{Ref: "external.json"},
				},
			},
		},
		{
			name: "update deeper nested internal references when wrapping for draft 7",
			schema: &Schema{
				Ref:  "external.json",
				Type: "object",
				Properties: map[string]*Schema{
					"foo": {
						Type: "object",
						Properties: map[string]*Schema{
							"lorem": {Type: "string"},
						},
					},
					"bar": {
						Items: &Schema{Ref: "#/properties/foo/properties/lorem"},
					},
				},
			},
			draft: 7,
			want: &Schema{
				AllOf: []*Schema{
					{
						Type: "object",
						Properties: map[string]*Schema{
							"foo": {
								Type: "object",
								Properties: map[string]*Schema{
									"lorem": {Type: "string"},
								},
							},
							"bar": {
								Items: &Schema{Ref: "#/allOf/0/properties/foo/properties/lorem"},
							},
						},
					},
					{Ref: "external.json"},
				},
			},
		},
		{
			name: "keep $defs references at root when wrapping for draft 7",
			schema: &Schema{
				Ref:  "external.json",
				Type: "object",
				Properties: map[string]*Schema{
					"foo": {Ref: "#/$defs/myDef"},
				},
				Defs: map[string]*Schema{
					"myDef": {Type: "string"},
				},
			},
			draft: 7,
			want: &Schema{
				AllOf: []*Schema{
					{
						Type: "object",
						Properties: map[string]*Schema{
							"foo": {Ref: "#/$defs/myDef"}, // Should NOT be updated
						},
					},
					{Ref: "external.json"},
				},
				Defs: map[string]*Schema{
					"myDef": {Type: "string"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.draft == 0 {
				tt.draft = 2020
			}
			err := ensureCompliant(tt.schema, tt.noAdditionalProperties, false, tt.draft)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			testutil.Equal(t, tt.want, tt.schema)
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
			err := ensureCompliant(tc.schema, false, false, 2020)
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
			testutil.Equal(t, tt.want, tt.schema)
		})
	}
}

func TestAddMissingGlobalProperty(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		want   *Schema
	}{
		{name: "nil", schema: nil, want: nil},
		{name: "empty", schema: &Schema{}, want: &Schema{}},
		{name: "bool/true", schema: SchemaTrue(), want: SchemaTrue()},
		{name: "bool/false", schema: SchemaTrue(), want: SchemaTrue()},

		{
			name:   "add when no additional properties",
			schema: &Schema{AdditionalProperties: SchemaFalse()},
			want: &Schema{
				AdditionalProperties: SchemaFalse(),
				Properties: map[string]*Schema{
					"global": defaultGlobal(),
				},
			},
		},
		{
			name:   "dont add when allowing additional properties",
			schema: &Schema{AdditionalProperties: SchemaTrue()},
			want:   &Schema{AdditionalProperties: SchemaTrue()},
		},
		{
			name:   "add when additional properties doesnt allow objects",
			schema: &Schema{AdditionalProperties: &Schema{Type: "string"}},
			want: &Schema{
				AdditionalProperties: &Schema{Type: "string"},
				Properties: map[string]*Schema{
					"global": defaultGlobal(),
				},
			},
		},
		{
			name:   "dont add additional properties allows objects",
			schema: &Schema{AdditionalProperties: &Schema{Type: []any{"string", "object"}}},
			want:   &Schema{AdditionalProperties: &Schema{Type: []any{"string", "object"}}},
		},
		{
			name:   "dont add additional properties allows any type",
			schema: &Schema{AdditionalProperties: &Schema{Type: nil}},
			want:   &Schema{AdditionalProperties: &Schema{Type: nil}},
		},

		{
			name: "dont add when global is already set",
			schema: &Schema{
				AdditionalProperties: SchemaFalse(),
				Properties: map[string]*Schema{
					"global": {Description: "foobar"},
				},
			},
			want: &Schema{
				AdditionalProperties: SchemaFalse(),
				Properties: map[string]*Schema{
					"global": {Description: "foobar"},
				},
			},
		},

		{
			// We practically ignore $ref here. It might allow/disallow "global",
			// but that's too convoluted to check for.
			name:   "dont add even though ref might disallow additional properties",
			schema: &Schema{Ref: "foobar.json"},
			want:   &Schema{Ref: "foobar.json"},
		},
		{
			// We practically ignore $ref here. It might allow/disallow "global",
			// but that's too convoluted to check for.
			name: "add even though ref might add global property",
			schema: &Schema{
				Ref:                  "foobar.json",
				AdditionalProperties: SchemaFalse(),
			},
			want: &Schema{
				Ref:                  "foobar.json",
				AdditionalProperties: SchemaFalse(),
				Properties: map[string]*Schema{
					"global": defaultGlobal(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addMissingGlobalProperty(tt.schema)
			testutil.Equal(t, tt.want, tt.schema)
		})
	}
}
