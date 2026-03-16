package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandRefs_NilSchema(t *testing.T) {
	err := ExpandRefs(nil)
	assert.ErrorContains(t, err, "nil schema")
}

func TestExpandRefs_NoRefs(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"foo": {Type: "string"},
		},
	}
	require.NoError(t, ExpandRefs(schema))
	assert.Equal(t, "string", schema.Properties["foo"].Type)
	assert.Nil(t, schema.Defs)
}

func TestExpandRefs_SimpleRef(t *testing.T) {
	// Schema: { properties: { foo: { $ref: "#/$defs/Foo" } }, $defs: { Foo: { type: "string" } } }
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"foo": {Ref: "#/$defs/Foo"},
		},
		Defs: map[string]*Schema{
			"Foo": {Type: "string"},
		},
	}
	require.NoError(t, ExpandRefs(schema))

	foo := schema.Properties["foo"]
	assert.Empty(t, foo.Ref, "$ref should be cleared after expansion")
	assert.Equal(t, "string", foo.Type)
	assert.Nil(t, schema.Defs, "$defs should be removed after expansion")
}

func TestExpandRefs_SiblingKeywordsPreserved(t *testing.T) {
	// When $ref coexists with other keywords (e.g. title), the local keywords win.
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"foo": {
				Ref:         "#/$defs/Foo",
				Title:       "My Override Title",
				Description: "Custom description",
			},
		},
		Defs: map[string]*Schema{
			"Foo": {
				Type:        "object",
				Title:       "Original Title",
				Description: "Original description",
				Properties: map[string]*Schema{
					"bar": {Type: "integer"},
				},
			},
		},
	}
	require.NoError(t, ExpandRefs(schema))

	foo := schema.Properties["foo"]
	assert.Empty(t, foo.Ref)
	assert.Equal(t, "object", foo.Type)
	assert.Equal(t, "My Override Title", foo.Title, "local title should win")
	assert.Equal(t, "Custom description", foo.Description, "local description should win")
	assert.NotNil(t, foo.Properties["bar"])
}

func TestExpandRefs_ChainExpansion(t *testing.T) {
	// A -> B -> C: all should be inlined.
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"a": {Ref: "#/$defs/A"},
		},
		Defs: map[string]*Schema{
			"A": {Ref: "#/$defs/B"},
			"B": {Type: "boolean"},
		},
	}
	require.NoError(t, ExpandRefs(schema))

	a := schema.Properties["a"]
	assert.Empty(t, a.Ref)
	assert.Equal(t, "boolean", a.Type)
	assert.Nil(t, schema.Defs)
}

func TestExpandRefs_CircularReference(t *testing.T) {
	// A refs B, B refs A — should not loop; the cycle ref is left in place.
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"root": {Ref: "#/$defs/A"},
		},
		Defs: map[string]*Schema{
			"A": {
				Type: "object",
				Properties: map[string]*Schema{
					"child": {Ref: "#/$defs/B"},
				},
			},
			"B": {
				Type: "object",
				Properties: map[string]*Schema{
					"back": {Ref: "#/$defs/A"},
				},
			},
		},
	}
	require.NoError(t, ExpandRefs(schema), "circular refs should not cause an error")

	// The outermost ref should be expanded.
	root := schema.Properties["root"]
	assert.Empty(t, root.Ref, "top-level ref should be expanded")
	assert.Equal(t, "object", root.Type)

	// The back-reference (cycle point) should be left as $ref.
	back := root.Properties["child"].Properties["back"]
	assert.Equal(t, "#/$defs/A", back.Ref, "cycle ref should be left in place")

	// $defs.A must be kept — back.$ref still points to it.
	assert.NotNil(t, schema.Defs["A"], "$defs.A should be preserved for the dangling circular ref")
}

func TestExpandRefs_ExternalRefSkipped(t *testing.T) {
	// External $refs (not starting with "#/") are left unchanged.
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"foo": {Ref: "https://example.com/schema.json"},
		},
	}
	require.NoError(t, ExpandRefs(schema))
	assert.Equal(t, "https://example.com/schema.json", schema.Properties["foo"].Ref, "external ref must not be touched")
}

func TestExpandRefs_RefNotFound(t *testing.T) {
	schema := &Schema{
		Properties: map[string]*Schema{
			"foo": {Ref: "#/$defs/Missing"},
		},
	}
	err := ExpandRefs(schema)
	assert.ErrorContains(t, err, `expand $ref "#/$defs/Missing": not found in schema`)
}

func TestExpandRefs_ErrorInExpandedCopy(t *testing.T) {
	// When A is expanded, its deep-copy contains a $ref that doesn't exist —
	// the error from the recursive expansion should propagate.
	schema := &Schema{
		Properties: map[string]*Schema{
			"x": {Ref: "#/$defs/A"},
		},
		Defs: map[string]*Schema{
			"A": {Ref: "#/$defs/Missing"},
		},
	}
	err := ExpandRefs(schema)
	assert.ErrorContains(t, err, `expand $ref "#/$defs/Missing": not found in schema`)
}

func TestExpandRefs_ClearsDefsAndDefinitions(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"foo": {Ref: "#/$defs/Foo"},
		},
		Defs: map[string]*Schema{
			"Foo": {Type: "number"},
		},
		Definitions: map[string]*Schema{
			"Bar": {Type: "string"},
		},
	}
	require.NoError(t, ExpandRefs(schema))
	assert.Nil(t, schema.Defs)
	assert.Nil(t, schema.Definitions)
}

