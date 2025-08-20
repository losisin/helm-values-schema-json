package pkg

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtr(t *testing.T) {
	tests := []struct {
		name string
		ptr  Ptr
		want string
	}{
		{
			name: "empty",
			ptr:  Ptr{},
			want: "/",
		},

		{
			name: "single prop",
			ptr:  Ptr{}.Prop("foo"),
			want: "/foo",
		},
		{
			name: "multiple props in different calls",
			ptr:  Ptr{}.Prop("foo").Prop("bar"),
			want: "/foo/bar",
		},
		{
			name: "multiple props in the same call",
			ptr:  Ptr{}.Prop("foo", "bar"),
			want: "/foo/bar",
		},

		{
			name: "single item",
			ptr:  Ptr{}.Item(1),
			want: "/1",
		},
		{
			name: "multiple items in different calls",
			ptr:  Ptr{}.Item(1).Item(2),
			want: "/1/2",
		},
		{
			name: "multiple items in the same call",
			ptr:  Ptr{}.Item(1, 2),
			want: "/1/2",
		},

		{
			name: "adding other pointers",
			ptr:  Ptr{"foo", "bar"}.Add(Ptr{"moo", "doo"}),
			want: "/foo/bar/moo/doo",
		},

		{
			name: "escapes slash",
			ptr:  Ptr{}.Prop("foo/bar"),
			want: "/foo~1bar",
		},
		{
			name: "escapes tilde",
			ptr:  Ptr{}.Prop("foo~bar"),
			want: "/foo~0bar",
		},
		{
			name: "escapes both",
			ptr:  Ptr{}.Prop("foo/bar~moo/doo"),
			want: "/foo~1bar~0moo~1doo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ptr.String()
			if got != tt.want {
				t.Fatalf("wrong result\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestParsePtr(t *testing.T) {
	tests := []struct {
		name string
		path string
		want Ptr
	}{
		{
			name: "empty",
			path: "",
			want: nil,
		},

		{
			name: "single prop",
			path: "/foo",
			want: NewPtr("foo"),
		},
		{
			name: "multiple props",
			path: "/foo/bar/12/lorem",
			want: NewPtr("foo", "bar").Item(12).Prop("lorem"),
		},
		{
			name: "special chars",
			path: "/foo~1bar/moo~0doo",
			want: NewPtr("foo/bar", "moo~doo"),
		},
		{
			name: "invalid syntax",
			path: "/foo~bar",
			want: NewPtr("foo~bar"),
		},
		{
			name: "with pound prefix",
			path: "#/foo",
			want: NewPtr("foo"),
		},
		{
			name: "without slash prefix",
			path: "foo/bar",
			want: NewPtr("foo", "bar"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePtr(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPtr_HasPrefix(t *testing.T) {
	tests := []struct {
		name      string
		ptr       Ptr
		prefix    Ptr
		want      bool
		wantAfter Ptr
	}{
		{
			name:      "empty both",
			ptr:       nil,
			prefix:    nil,
			want:      true,
			wantAfter: nil,
		},
		{
			name:      "empty prefix",
			ptr:       Ptr{"foo"},
			prefix:    nil,
			want:      true,
			wantAfter: Ptr{"foo"},
		},
		{
			name:      "empty ptr",
			ptr:       nil,
			prefix:    NewPtr("foo"),
			want:      false,
			wantAfter: nil,
		},
		{
			name:      "longer prefix than ptr",
			ptr:       NewPtr("foo"),
			prefix:    NewPtr("foo", "bar"),
			want:      false,
			wantAfter: NewPtr("foo"),
		},
		{
			name:      "match",
			ptr:       NewPtr("foo", "bar"),
			prefix:    NewPtr("foo"),
			want:      true,
			wantAfter: NewPtr("bar"),
		},
		{
			name:      "match plain",
			ptr:       NewPtr("foo", "bar"),
			prefix:    NewPtr("foo"),
			want:      true,
			wantAfter: NewPtr("bar"),
		},
		{
			name:      "match with special chars",
			ptr:       NewPtr("foo/bar"),
			prefix:    NewPtr("foo/bar"),
			want:      true,
			wantAfter: nil,
		},
		{
			name:      "no match because of special chars in ptr",
			ptr:       NewPtr("foo/bar"),
			prefix:    NewPtr("foo"),
			want:      false,
			wantAfter: NewPtr("foo/bar"),
		},
		{
			name:      "no match because of special chars in prefix",
			ptr:       NewPtr("foo", "bar"),
			prefix:    NewPtr("foo/bar"),
			want:      false,
			wantAfter: NewPtr("foo", "bar"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ptr.HasPrefix(tt.prefix)
			if got != tt.want {
				t.Errorf("wrong HasPrefix result\nptr:    %q\nprefix: %q\nwant:   %t\ngot:    %t",
					tt.ptr, tt.prefix, tt.want, got)
			}
			gotAfter, _ := tt.ptr.CutPrefix(tt.prefix)
			if !slices.Equal(gotAfter, tt.wantAfter) {
				t.Errorf("wrong CutPrefix result\nptr:    %q\nprefix: %q\nwant after: %[3]q (%#[3]v)\ngot after:  %[4]q (%#[4]v)",
					tt.ptr, tt.prefix, tt.wantAfter, gotAfter)
			}
		})
	}
}

func TestPtr_Equals(t *testing.T) {
	tests := []struct {
		name  string
		ptr   Ptr
		other Ptr
		want  bool
	}{
		{
			name:  "nil both",
			ptr:   nil,
			other: nil,
			want:  true,
		},
		{
			name:  "empty both",
			ptr:   NewPtr(),
			other: NewPtr(),
			want:  true,
		},
		{
			name:  "nil lhs",
			ptr:   nil,
			other: NewPtr(),
			want:  true,
		},
		{
			name:  "nil rhs",
			ptr:   NewPtr(),
			other: nil,
			want:  true,
		},
		{
			name:  "value and nil",
			ptr:   Ptr{"foo"},
			other: nil,
			want:  false,
		},
		{
			name:  "nil and value",
			ptr:   nil,
			other: Ptr{"foo"},
			want:  false,
		},
		{
			name:  "longer other than ptr",
			ptr:   NewPtr("foo"),
			other: NewPtr("foo", "bar"),
			want:  false,
		},
		{
			name:  "match with special chars",
			ptr:   NewPtr("foo/bar"),
			other: NewPtr("foo/bar"),
			want:  true,
		},
		{
			name:  "no match because of special chars in ptr",
			ptr:   NewPtr("foo/bar"),
			other: NewPtr("foo", "bar"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ptr.Equals(tt.other)
			if got != tt.want {
				t.Fatalf("wrong result\nptr:   %q\nother: %q\nwant:  %t\ngot:   %t",
					tt.ptr, tt.other, tt.want, got)
			}
		})
	}
}

func TestPtr_Resolve(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		ptr    Ptr
		want   func(schema *Schema) []ResolvedSchema
	}{
		{
			name:   "nil ptr is same as root",
			schema: &Schema{ID: "root"},
			ptr:    nil,
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{{nil, schema}}
			},
		},
		{
			name:   "root ptr",
			schema: &Schema{ID: "root"},
			ptr:    Ptr{},
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{{Ptr{}, schema}}
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
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("$defs", "foo.json"), schema.Defs["foo.json"]},
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
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("definitions", "foo.json"), schema.Definitions["foo.json"]},
				}
			},
		},
		{
			name: "find in patternProperties",
			schema: &Schema{PatternProperties: map[string]*Schema{
				"foo": {ID: "foo"},
			}},
			ptr: NewPtr("patternProperties", "foo"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("patternProperties", "foo"), schema.PatternProperties["foo"]},
				}
			},
		},
		{
			name: "find in properties",
			schema: &Schema{Properties: map[string]*Schema{
				"foo": {ID: "foo"},
			}},
			ptr: NewPtr("properties", "foo"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("properties", "foo"), schema.Properties["foo"]},
				}
			},
		},

		{
			name:   "find in allOf",
			schema: &Schema{AllOf: []*Schema{{ID: "foo"}}},
			ptr:    NewPtr("allOf", "0"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("allOf", "0"), schema.AllOf[0]},
				}
			},
		},
		{
			name:   "find in anyOf",
			schema: &Schema{AnyOf: []*Schema{{ID: "foo"}}},
			ptr:    NewPtr("anyOf", "0"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("anyOf", "0"), schema.AnyOf[0]},
				}
			},
		},
		{
			name:   "find in oneOf",
			schema: &Schema{OneOf: []*Schema{{ID: "foo"}}},
			ptr:    NewPtr("oneOf", "0"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("oneOf", "0"), schema.OneOf[0]},
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
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("$defs", "foo.json"), schema.Defs["foo.json"]},
					{NewPtr("$defs", "foo.json", "definitions", "bar.json"), schema.Defs["foo.json"].Definitions["bar.json"]},
				}
			},
		},

		{
			name:   "find additionalItems",
			schema: &Schema{AdditionalItems: &Schema{ID: "additionalItems"}},
			ptr:    NewPtr("additionalItems"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("additionalItems"), schema.AdditionalItems},
				}
			},
		},
		{
			name:   "find additionalProperties",
			schema: &Schema{AdditionalProperties: &Schema{ID: "additionalProperties"}},
			ptr:    NewPtr("additionalProperties"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("additionalProperties"), schema.AdditionalProperties},
				}
			},
		},
		{
			name:   "find items",
			schema: &Schema{Items: &Schema{ID: "items"}},
			ptr:    NewPtr("items"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("items"), schema.Items},
				}
			},
		},
		{
			name: "find nested items",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo": {Items: &Schema{ID: "items"}},
				},
			},
			ptr: NewPtr("$defs", "foo", "items"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("$defs", "foo"), schema.Defs["foo"]},
					{NewPtr("$defs", "foo", "items"), schema.Defs["foo"].Items},
				}
			},
		},
		{
			name:   "find not",
			schema: &Schema{Not: &Schema{ID: "not"}},
			ptr:    NewPtr("not"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("not"), schema.Not},
				}
			},
		},

		{
			name: "unknown property",
			schema: &Schema{
				Defs: map[string]*Schema{"foo.json": {ID: "foo.json"}},
			},
			ptr: NewPtr("foobar", "moodoo"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{{NewPtr(), schema}}
			},
		},
		{
			name: "unknown array",
			schema: &Schema{
				Defs: map[string]*Schema{"foo.json": {ID: "foo.json"}},
			},
			ptr: NewPtr("foobar", "0"),
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{{NewPtr(), schema}}
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
			want: func(schema *Schema) []ResolvedSchema {
				return []ResolvedSchema{
					{NewPtr(), schema},
					{NewPtr("$defs", "foo.json"), schema.Defs["foo.json"]},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := tt.ptr.Resolve(tt.schema)
			assert.Equal(t, tt.want(tt.schema), resolved)
		})
	}
}
