package yamlutil

import (
	"strconv"

	"go.yaml.in/yaml/v3"
)

// Consts are copied from [go.yaml.in/yaml/v3] source code.
const (
	nullTag      = "!!null"
	boolTag      = "!!bool"
	strTag       = "!!str"
	intTag       = "!!int"
	floatTag     = "!!float"
	timestampTag = "!!timestamp"
	seqTag       = "!!seq"
	mapTag       = "!!map"
	binaryTag    = "!!binary"
	mergeTag     = "!!merge"
)

func Bool(v bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: boolTag, Value: strconv.FormatBool(v)}
}

func String(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: strTag, Value: v}
}

func Int(v int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: intTag, Value: strconv.FormatInt(int64(v), 10)}
}

func Uint(v uint) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: intTag, Value: strconv.FormatUint(uint64(v), 10)}
}

func Float32(v float32) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: floatTag, Value: strconv.FormatFloat(float64(v), 'f', -1, 32)}
}

func Float64(v float32) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: floatTag, Value: strconv.FormatFloat(float64(v), 'f', -1, 64)}
}

func Map(kv ...*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: kv,
	}
}

func Seq(items ...*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: items,
	}
}

func WithLineComment(comment string, node *yaml.Node) *yaml.Node {
	node.LineComment = comment
	return node
}

func WithHeadComment(comment string, node *yaml.Node) *yaml.Node {
	node.HeadComment = comment
	return node
}

func WithFootComment(comment string, node *yaml.Node) *yaml.Node {
	node.FootComment = comment
	return node
}
