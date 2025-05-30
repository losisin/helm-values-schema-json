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

// String returns a slash-delimited string of the pointer.
//
// Example:
//
//	NewPtr("foo", "bar").String()
//	// => "/foo/bar"
func (p Ptr) String() string {
	return "/" + strings.Join(p, "/")
}
