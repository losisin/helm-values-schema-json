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
