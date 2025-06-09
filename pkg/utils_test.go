package pkg

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniqueStringAppend(t *testing.T) {
	tests := []struct {
		name string
		dest []string
		src  []string
		want []string
	}{
		{
			name: "empty slices",
			dest: []string{},
			src:  []string{},
			want: []string{},
		},
		{
			name: "unique items",
			dest: []string{"a", "b"},
			src:  []string{"c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "duplicate items",
			dest: []string{"a", "b"},
			src:  []string{"b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "all duplicate items",
			dest: []string{"a", "b"},
			src:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "src only has duplicates",
			dest: []string{"c", "d"},
			src:  []string{"c", "c", "d", "d"},
			want: []string{"c", "d"},
		},
		{
			name: "empty dest slice",
			dest: []string{},
			src:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "empty src slice",
			dest: []string{"a", "b"},
			src:  []string{},
			want: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the 'dest' slice to preserve the original.
			destCopy := make([]string, len(tt.dest))
			copy(destCopy, tt.dest)

			// Call the function with the copy.
			got := uniqueStringAppend(destCopy, tt.src...)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uniqueStringAppend() = %v, want %v", got, tt.want)
			}

			// Verify that the original 'dest' slice is not modified.
			if !reflect.DeepEqual(tt.dest, destCopy) {
				t.Errorf("uniqueStringAppend() unexpectedly modified the original dest slice")
			}
		})
	}
}

func TestMustParseURL(t *testing.T) {
	want := &url.URL{
		Scheme: "http",
		Host:   "example.com",
	}
	assert.Equal(t, want, mustParseURL("http://example.com"))
	assert.Panics(t, func() { mustParseURL("::") })
}

func TestComparePointer(t *testing.T) {
	tests := []struct {
		name string
		a, b *bool
		want bool
	}{
		{name: "a false b false", a: boolPtr(false), b: boolPtr(false), want: true},
		{name: "a false b nil", a: boolPtr(false), b: nil, want: false},
		{name: "a false b true", a: boolPtr(false), b: boolPtr(true), want: false},
		{name: "a nil b false", a: nil, b: boolPtr(false), want: false},
		{name: "a nil b nil", a: nil, b: nil, want: true},
		{name: "a nil b true", a: nil, b: boolPtr(true), want: false},
		{name: "a true b false", a: boolPtr(true), b: boolPtr(false), want: false},
		{name: "a true b nil", a: boolPtr(true), b: nil, want: false},
		{name: "a true b true", a: boolPtr(true), b: boolPtr(true), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.a != nil {
				t.Logf("a: (*bool)(%t)", *tt.a)
			} else {
				t.Logf("a: (*bool)(nil)")
			}
			if tt.b != nil {
				t.Logf("b: (*bool)(%t)", *tt.b)
			} else {
				t.Logf("b: (*bool)(nil)")
			}
			t.Logf("want: %t", tt.want)

			got := comparePointer(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("got:  %t", got)
			}
		})
	}
}
