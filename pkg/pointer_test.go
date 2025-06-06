package pkg

import (
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
		name   string
		ptr    Ptr
		prefix Ptr
		want   bool
	}{
		{
			name:   "empty both",
			ptr:    nil,
			prefix: nil,
			want:   true,
		},
		{
			name:   "empty prefix",
			ptr:    Ptr{"foo"},
			prefix: nil,
			want:   true,
		},
		{
			name:   "empty ptr",
			ptr:    nil,
			prefix: NewPtr("foo"),
			want:   false,
		},
		{
			name:   "longer prefix than ptr",
			ptr:    NewPtr("foo"),
			prefix: NewPtr("foo", "bar"),
			want:   false,
		},
		{
			name:   "match",
			ptr:    NewPtr("foo", "bar"),
			prefix: NewPtr("foo"),
			want:   true,
		},
		{
			name:   "match plain",
			ptr:    NewPtr("foo", "bar"),
			prefix: NewPtr("foo"),
			want:   true,
		},
		{
			name:   "match with special chars",
			ptr:    NewPtr("foo/bar"),
			prefix: NewPtr("foo/bar"),
			want:   true,
		},
		{
			name:   "no match because of special chars in ptr",
			ptr:    NewPtr("foo/bar"),
			prefix: NewPtr("foo"),
			want:   false,
		},
		{
			name:   "no match because of special chars in prefix",
			ptr:    NewPtr("foo", "bar"),
			prefix: NewPtr("foo/bar"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ptr.HasPrefix(tt.prefix)
			if got != tt.want {
				t.Fatalf("wrong result\nptr:    %q\nprefix: %q\nwant:   %t\ngot:    %t",
					tt.ptr, tt.prefix, tt.want, got)
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
