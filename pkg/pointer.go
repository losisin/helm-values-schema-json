package pkg

import (
	"slices"
	"strconv"
	"strings"
)

// Ptr is a JSON Ptr [https://datatracker.ietf.org/doc/html/rfc6901].
//
// The type is meant to be used in an immutable way, where all methods
// return a new pointer with the appropriate changes.
//
// You don't have to initialize this struct with the [NewPtr] function.
// A nil value is equivalent to an empty path: "/"
type Ptr []string

// NewPtr returns a new [Ptr] with optionally provided property names
func NewPtr(name ...string) Ptr {
	return (Ptr{}).Prop(name...)
}

func ParsePtr(path string) Ptr {
	path = strings.TrimPrefix(path, "#")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil
	}
	split := strings.Split(path, "/")
	for i := range split {
		split[i] = pointerReplacer.Replace(pointerReplacerReverse.Replace(split[i]))
	}
	return Ptr(split)
}

// pointerReplacer contains the replcements defined in https://datatracker.ietf.org/doc/html/rfc6901#section-3
var pointerReplacer = strings.NewReplacer(
	"~", "~0",
	"/", "~1",
)

var pointerReplacerReverse = strings.NewReplacer(
	"~0", "~",
	"~1", "/",
)

func (p Ptr) Prop(name ...string) Ptr {
	for _, s := range name {
		p = append(p, pointerReplacer.Replace(s))
	}
	return p
}

func (p Ptr) Item(index ...int) Ptr {
	for _, i := range index {
		p = append(p, strconv.Itoa(i))
	}
	return p
}

func (p Ptr) Add(other ...Ptr) Ptr {
	for _, o := range other {
		p = append(p, o...)
	}
	return p
}

func (p Ptr) HasPrefix(prefix Ptr) bool {
	if len(prefix) > len(p) {
		return false
	}
	return slices.Equal(p[:len(prefix)], prefix)
}

func (p Ptr) CutPrefix(prefix Ptr) (after Ptr, ok bool) {
	if !p.HasPrefix(prefix) {
		return p, false
	}
	return p[len(prefix):], true
}

func (p Ptr) Equals(other Ptr) bool {
	if len(other) != len(p) {
		return false
	}
	return slices.Equal(p, other)
}

// String returns a slash-delimited string of the pointer.
//
// Example:
//
//	NewPtr("foo", "bar").String()
//	// => "/foo/bar"
func (p Ptr) String() string {
	return "/" + strings.Join(p, "/")
}

type ResolvedSchema struct {
	Ptr    Ptr
	Schema *Schema
}

// Resolve returns all of the matched subschemas.
// The last slice element is the deepest subschema along the pointer's path.
//
// For example:
//
//	Resolve(schema, NewPtr("/$defs/foo/items/type"))
//	// => []*Schema{ "/$defs", "/$defs/foo", "/$defs/foo/items" }
func (ptr Ptr) Resolve(schema *Schema) []ResolvedSchema {
	var result []ResolvedSchema
	var offset int
	for schema != nil {
		result = append(result, ResolvedSchema{ptr[:offset], schema})
		ptrRest := ptr[offset:]

		if len(ptrRest) == 0 {
			return result
		} else if s, ok := ptrRest.resolveField(schema); ok {
			offset += 1
			schema = s
			continue
		}

		if len(ptrRest) < 2 {
			return result
		} else if s, ok := ptrRest.resolveMap(schema); ok {
			offset += 2
			schema = s
		} else if s, ok := ptrRest.resolveSlice(schema); ok {
			offset += 2
			schema = s
		} else {
			return result
		}
	}
	return result
}

func (ptr Ptr) resolveField(schema *Schema) (*Schema, bool) {
	switch ptr[0] {
	case "additionalItems":
		return schema.AdditionalItems, true
	case "additionalProperties":
		return schema.AdditionalProperties, true
	case "items":
		return schema.Items, true
	case "not":
		return schema.Not, true
	default:
		return nil, false
	}
}

func (ptr Ptr) resolveMap(schema *Schema) (*Schema, bool) {
	switch ptr[0] {
	case "$defs":
		return schema.Defs[ptr[1]], true
	case "definitions":
		return schema.Definitions[ptr[1]], true
	case "patternProperties":
		return schema.PatternProperties[ptr[1]], true
	case "properties":
		return schema.Properties[ptr[1]], true
	default:
		return nil, false
	}
}

func (ptr Ptr) resolveSlice(schema *Schema) (*Schema, bool) {
	index, err := strconv.Atoi(ptr[1])
	if err != nil {
		return nil, false
	}
	switch ptr[0] {
	case "allOf":
		return tryIndex(schema.AllOf, index), true
	case "anyOf":
		return tryIndex(schema.AnyOf, index), true
	case "oneOf":
		return tryIndex(schema.OneOf, index), true
	default:
		return nil, false
	}
}
