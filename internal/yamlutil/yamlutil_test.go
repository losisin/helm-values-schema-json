package yamlutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestCreateNode(t *testing.T) {
	tests := []struct {
		name string
		f    func() *yaml.Node
		want any
	}{
		{
			name: "bool",
			f:    func() *yaml.Node { return Bool(true) },
			want: true,
		},
		{
			name: "string",
			f:    func() *yaml.Node { return String("foobar") },
			want: "foobar",
		},
		{
			name: "int",
			f:    func() *yaml.Node { return Int(123) },
			want: 123,
		},
		{
			name: "uint",
			f:    func() *yaml.Node { return Uint(123) },
			want: 123,
		},
		{
			name: "float32",
			f:    func() *yaml.Node { return Float32(12.5) },
			want: 12.5,
		},
		{
			name: "float64",
			f:    func() *yaml.Node { return Float64(12.5) },
			want: 12.5,
		},
		{
			name: "map",
			f: func() *yaml.Node {
				return Map(
					String("foo"),
					String("bar"),
					String("moo"),
					Int(123),
				)
			},
			want: map[string]any{
				"foo": "bar",
				"moo": 123,
			},
		},
		{
			name: "seq",
			f: func() *yaml.Node {
				return Seq(
					String("foo"),
					String("bar"),
					Int(123),
				)
			},
			want: []any{"foo", "bar", 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.f()
			var decoded any
			require.NoError(t, node.Decode(&decoded))
			assert.Equal(t, tt.want, decoded)
		})
	}
}

func TestWithLineComment(t *testing.T) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "foobar",
	}

	node = WithLineComment("some line comment", node)

	b, err := yaml.Marshal(node)
	require.NoError(t, err)

	assert.Equal(t, "foobar # some line comment\n", string(b))
}
