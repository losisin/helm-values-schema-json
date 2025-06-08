package pkg

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBundleRefToID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "empty id",
			ref:  "",
			want: "",
		},
		{
			name: "valid",
			ref:  "https://localhost/foo/bar",
			want: "https://localhost/foo/bar",
		},
		{
			name: "keeps userinfo",
			ref:  "https://user:pass@localhost/foo/bar",
			want: "https://user:pass@localhost/foo/bar",
		},
		{
			name: "removes fragment",
			ref:  "https://localhost/foo/bar#mayo",
			want: "https://localhost/foo/bar",
		},
		{
			name: "invalid URL",
			ref:  "::",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimFragment(tt.ref)
			if got != tt.want {
				t.Fatalf("wrong result\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestGenerateBundledName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   string
		defs map[string]*Schema
		want string
	}{
		{
			name: "empty id",
			id:   "",
			defs: map[string]*Schema{},
			want: "",
		},
		{
			name: "new item",
			id:   "foo.json",
			defs: map[string]*Schema{},
			want: "foo.json",
		},
		{
			name: "colliding item",
			id:   "some/path/foo.json",
			defs: map[string]*Schema{
				"foo.json": {
					ID: "some/other/path/foo.json",
				},
			},
			want: "foo.json_2",
		},
		{
			name: "existing item",
			id:   "foo.json",
			defs: map[string]*Schema{
				"foo.json": {
					ID: "foo.json",
				},
			},
			want: "foo.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBundledName(tt.id, tt.defs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBundle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema *Schema
		loader Loader
		want   *Schema
	}{
		{
			name:   "empty schema",
			schema: &Schema{},
			loader: DummyLoader{},
			want:   &Schema{},
		},

		{
			name: "sets $id",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
				},
				Defs: map[string]*Schema{
					"file.json": {ID: "../some/file.json"},
				},
			},
		},

		{
			name: "sets context ID",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						ID:  "some-schema-id",
						Ref: "../some/file.json",
					},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					var referrer string
					if v, ok := ctx.Value(loaderContextReferrer).(string); ok {
						referrer = v
					}
					return &Schema{Comment: "Referred by: " + referrer}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						ID:  "some-schema-id",
						Ref: "../some/file.json",
					},
				},
				Defs: map[string]*Schema{
					"file.json": {
						ID:      "../some/file.json",
						Comment: "Referred by: some-schema-id",
					},
				},
			},
		},

		{
			name: "only bundle once",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
					"bar": {Ref: "../some/file.json"},
					"moo": {Ref: "../some/file.json"},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
					"bar": {Ref: "../some/file.json"},
					"moo": {Ref: "../some/file.json"},
				},
				Defs: map[string]*Schema{
					"file.json": {ID: "../some/file.json"},
				},
			},
		},

		{
			name: "already bundled self",
			schema: &Schema{
				Defs: map[string]*Schema{
					"file.json": {
						ID: "../some/file.json",
						Properties: map[string]*Schema{
							"foo": {Ref: "../some/file.json"},
						},
					},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Defs: map[string]*Schema{
					"file.json": {
						ID: "../some/file.json",
						Properties: map[string]*Schema{
							"foo": {Ref: "../some/file.json"},
						},
					},
				},
			},
		},

		{
			name: "subschema is bundled using id",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
			},
			loader: DummyLoader{
				func(ctx context.Context, ref *url.URL) (*Schema, error) {
					switch ref.String() {
					case "foo.json":
						return &Schema{
							Properties: map[string]*Schema{
								"num": {Ref: "bar.json"},
							},
							Defs: map[string]*Schema{
								"bar.json": {
									ID:   "bar.json",
									Type: "number",
								},
							},
						}, nil
					default:
						return nil, fmt.Errorf("undefined test schema: %s", ref)
					}
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID: "foo.json",
						Properties: map[string]*Schema{
							"num": {Ref: "bar.json"},
						},
					},
					"bar.json": {
						ID:   "bar.json",
						Type: "number",
					},
				},
			},
		},

		{
			name: "subschema is bundled without id",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
			},
			loader: DummyLoader{
				func(ctx context.Context, ref *url.URL) (*Schema, error) {
					switch ref.String() {
					case "foo.json":
						return &Schema{
							Properties: map[string]*Schema{
								"num": {Ref: "#/$defs/bar.json"},
							},
							Defs: map[string]*Schema{
								"bar.json": {
									Type: "number",
								},
							},
						}, nil
					default:
						return nil, fmt.Errorf("undefined test schema: %s", ref)
					}
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID: "foo.json",
						Properties: map[string]*Schema{
							"num": {Ref: "#/$defs/bar.json"},
						},
						Defs: map[string]*Schema{
							"bar.json": {
								Type: "number",
							},
						},
					},
				},
			},
		},

		{
			name:   "additionalProperties false",
			schema: &Schema{AdditionalProperties: &SchemaFalse},
			loader: DummyLoader{},
			want:   &Schema{AdditionalProperties: &SchemaFalse},
		},
		{
			name: "additionalProperties schema",
			schema: &Schema{
				AdditionalProperties: &Schema{Ref: "foo.json"},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{Type: "string"}, nil
				},
			},
			want: &Schema{
				AdditionalProperties: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID:   "foo.json",
						Type: "string",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleSchema(t.Context(), tt.loader, tt.schema, "/")
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)

			want, err := yaml.Marshal(tt.want)
			require.NoError(t, err)
			t.Logf("Want:\n%s", string(want))

			got, err := yaml.Marshal(tt.schema)
			require.NoError(t, err)
			t.Logf("Got:\n%s", string(got))
		})
	}
}

func TestBundle_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		schema  *Schema
		loader  Loader
		wantErr string
	}{
		{
			name:    "nil loader",
			schema:  &Schema{},
			loader:  nil,
			wantErr: "nil loader",
		},
		{
			name:    "nil schema",
			schema:  nil,
			loader:  DummyLoader{},
			wantErr: "nil schema",
		},
		{
			name: "invalid URL",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "::",
					},
				},
			},
			loader:  DummyLoader{},
			wantErr: `/properties/foo/$ref: parse "::": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleSchema(t.Context(), tt.loader, tt.schema, "/")
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestBundleRemoveIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema *Schema
		want   *Schema
	}{
		{
			name:   "empty schema",
			schema: &Schema{},
			want:   &Schema{},
		},

		{
			name: "single subschema has inline ref",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						ID:    "foo.json",
						Items: &Schema{Ref: "#/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
			want: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						Items: &Schema{Ref: "#/$defs/foo/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
		},

		{
			name: "sub-subschema has inline ref",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						ID:    "foo.json",
						Items: &Schema{Ref: "#/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Ref: "bar.json"},
						},
					},
					"bar": {
						ID:    "bar.json",
						Items: &Schema{Ref: "#/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
			want: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						Items: &Schema{Ref: "#/$defs/foo/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Ref: "#/$defs/bar"},
						},
					},
					"bar": {
						Items: &Schema{Ref: "#/$defs/bar/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleRemoveIDs(tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)

			want, err := yaml.Marshal(tt.want)
			require.NoError(t, err)
			t.Logf("Want:\n%s", string(want))

			got, err := yaml.Marshal(tt.schema)
			require.NoError(t, err)
			t.Logf("Got:\n%s", string(got))
		})
	}
}

func TestBundleRemoveIDs_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		schema  *Schema
		wantErr string
	}{
		{
			name:    "nil schema",
			schema:  nil,
			wantErr: "nil schema",
		},
		{
			name: "invalid URL",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "::",
					},
				},
			},
			wantErr: `/properties/foo: parse $ref="::" as URL: parse "::": missing protocol scheme`,
		},
		{
			name: "invalid ref",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "./no/$defs/with/this/ref",
					},
				},
			},
			wantErr: `/properties/foo: no $defs found that matches $ref="./no/$defs/with/this/ref"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleRemoveIDs(tt.schema)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestRemoveUnusedDefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema *Schema
		want   *Schema
	}{
		{
			name:   "empty schema",
			schema: &Schema{},
			want:   &Schema{},
		},

		{
			name: "remove single def",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
			want: &Schema{},
		},

		{
			name: "keep single def",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
		},

		{
			name: "keep some remove some",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
					"bar.json": {ID: "bar.json"},
					"moo.json": {ID: "moo.json"},
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
		},

		{
			name: "remove double-referenced",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {
						ID:    "foo.json",
						Items: &Schema{Ref: "bar.json"},
					},
					"bar.json": {ID: "bar.json"},
				},
			},
			want: &Schema{},
		},

		{
			name: "keep double referenced",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID:    "foo.json",
						Items: &Schema{Ref: "bar.json"},
					},
					"bar.json": {ID: "bar.json"},
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID:    "foo.json",
						Items: &Schema{Ref: "bar.json"},
					},
					"bar.json": {ID: "bar.json"},
				},
			},
		},

		{
			name: "remove some nested inline reference",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID:    "foo.json",
						Items: &Schema{Ref: "#/$defs/foo.json/definitions/bar.json"},
						Definitions: map[string]*Schema{
							"bar.json": {Type: "string"},
							"moo.json": {Type: "string"},
							"doo.json": {Type: "string"},
						},
					},
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID:    "foo.json",
						Items: &Schema{Ref: "#/$defs/foo.json/definitions/bar.json"},
						Definitions: map[string]*Schema{
							"bar.json": {Type: "string"},
						},
					},
				},
			},
		},

		{
			name: "ignore invalid refs",
			schema: &Schema{
				Items: &Schema{Ref: "#/foo.json"},
			},
			want: &Schema{
				Items: &Schema{Ref: "#/foo.json"},
			},
		},
		{
			name: "ignore invalid url",
			schema: &Schema{
				Items: &Schema{Ref: "::"},
			},
			want: &Schema{
				Items: &Schema{Ref: "::"},
			},
		},

		{
			name: "reference field in def",
			schema: &Schema{
				Items: &Schema{Ref: "#/$defs/foo.json/properties/moo"},
				Defs: map[string]*Schema{
					"foo.json": {
						Properties: map[string]*Schema{
							"moo": {Type: "string"},
						},
					},
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "#/$defs/foo.json/properties/moo"},
				Defs: map[string]*Schema{
					"foo.json": {
						Properties: map[string]*Schema{
							"moo": {Type: "string"},
						},
					},
				},
			},
		},

		{
			name: "remove self-referential",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {
						Properties: map[string]*Schema{
							"moo": {Ref: "#/$defs/foo.json"},
						},
					},
				},
			},
			want: &Schema{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RemoveUnusedDefs(tt.schema)
			assert.Equal(t, tt.want, tt.schema)

			want, err := yaml.Marshal(tt.want)
			require.NoError(t, err)
			t.Logf("Want:\n%s", string(want))

			got, err := yaml.Marshal(tt.schema)
			require.NoError(t, err)
			t.Logf("Got:\n%s", string(got))
		})
	}
}

func TestResolvePtr(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		ptr    Ptr
		want   func(schema *Schema) []*Schema
	}{
		{
			name:   "nil ptr is same as root",
			schema: &Schema{ID: "root"},
			ptr:    nil,
			want: func(schema *Schema) []*Schema {
				return []*Schema{schema}
			},
		},
		{
			name:   "root ptr",
			schema: &Schema{ID: "root"},
			ptr:    Ptr{},
			want: func(schema *Schema) []*Schema {
				return []*Schema{schema}
			},
		},
		{
			name: "find in defs",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
			ptr: NewPtr("$defs", "foo.json"),
			want: func(schema *Schema) []*Schema {
				return []*Schema{
					schema,
					schema.Defs["foo.json"],
				}
			},
		},
		{
			name: "find in definitions",
			schema: &Schema{
				Definitions: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
			ptr: NewPtr("definitions", "foo.json"),
			want: func(schema *Schema) []*Schema {
				return []*Schema{
					schema,
					schema.Definitions["foo.json"],
				}
			},
		},
		{
			name: "find nested",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {
						ID: "foo.json",
						Definitions: map[string]*Schema{
							"bar.json": {ID: "bar.json"},
						},
					},
				},
			},
			ptr: NewPtr("$defs", "foo.json", "definitions", "bar.json"),
			want: func(schema *Schema) []*Schema {
				return []*Schema{
					schema,
					schema.Defs["foo.json"],
					schema.Defs["foo.json"].Definitions["bar.json"],
				}
			},
		},

		{
			name: "unknown property",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {ID: "foo.json"},
				},
			},
			ptr: NewPtr("foobar", "moodoo"),
			want: func(schema *Schema) []*Schema {
				return []*Schema{schema}
			},
		},
		{
			name: "unknown nested",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo.json": {
						ID: "foo.json",
						Definitions: map[string]*Schema{
							"bar.json": {ID: "bar.json"},
						},
					},
				},
			},
			ptr: NewPtr("$defs", "foo.json", "definitions", "moo.json"),
			want: func(schema *Schema) []*Schema {
				return []*Schema{
					schema,
					schema.Defs["foo.json"],
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := resolvePtr(tt.schema, tt.ptr)
			assert.Equal(t, tt.want(tt.schema), resolved)
		})
	}
}
