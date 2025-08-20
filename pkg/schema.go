package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SchemaKind is an internal enum used to be able to parse
// an entire schema as a boolean, which is used on fields like
// "additionalProperties".
//
// The zero value is "treat this as an object".
type SchemaKind byte

const (
	SchemaKindObject SchemaKind = iota
	SchemaKindTrue
	SchemaKindFalse
)

// SchemaTrue returns a newly allocated schema that just evaluates to "true"
// when encoded as JSON/YAML.
func SchemaTrue() *Schema { return &Schema{kind: SchemaKindTrue} }

// SchemaTrue returns a newly allocated schema that just evaluates to "false"
// when encoded as JSON/YAML.
func SchemaFalse() *Schema { return &Schema{kind: SchemaKindFalse} }

func SchemaBool(value bool) *Schema {
	if value {
		return SchemaTrue()
	}
	return SchemaFalse()
}

// IsBool returns true when the [Schema] represents a boolean value
// instead of an object.
func (k SchemaKind) IsBool() bool {
	switch k {
	case SchemaKindTrue, SchemaKindFalse:
		return true
	default:
		return false
	}
}

// String implements [fmt.Stringer].
func (k SchemaKind) String() string {
	switch k {
	case SchemaKindTrue:
		return "true"
	case SchemaKindFalse:
		return "false"
	case SchemaKindObject:
		return "object"
	default:
		return fmt.Sprintf("SchemaKind(%d)", k)
	}
}

// GoString implements [fmt.GoStringer],
// and is used in debug output such as:
//
//	fmt.Sprint("%#v", kind)
func (k SchemaKind) GoString() string {
	return k.String()
}

type Schema struct {
	kind SchemaKind

	// Field ordering is taken from https://github.com/sourcemeta/core/blob/429eb970f3e303c3f61ba3cf066c7bd766453e15/src/core/jsonschema/jsonschema.cc#L459-L546
	Schema                string             `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	ID                    string             `json:"$id,omitempty" yaml:"$id,omitempty"`
	Title                 string             `json:"title,omitempty" yaml:"title,omitempty"`
	Description           string             `json:"description,omitempty" yaml:"description,omitempty"`
	Comment               string             `json:"$comment,omitempty" yaml:"$comment,omitempty"`
	Examples              []any              `json:"examples,omitempty" yaml:"examples,omitempty"`
	ReadOnly              bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	Default               any                `json:"default,omitempty" yaml:"default,omitempty"`
	Ref                   string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	RefReferrer           Referrer           `json:"-" yaml:"-"`
	Type                  any                `json:"type,omitempty" yaml:"type,omitempty"`
	Enum                  []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	AllOf                 []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOf                 []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOf                 []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Not                   *Schema            `json:"not,omitempty" yaml:"not,omitempty"`
	Maximum               *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Minimum               *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	MultipleOf            *float64           `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Pattern               string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxLength             *uint64            `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength             *uint64            `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxItems              *uint64            `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems              *uint64            `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems           bool               `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	Items                 *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	AdditionalItems       *Schema            `json:"additionalItems,omitempty" yaml:"additionalItems,omitempty"`
	Required              []string           `json:"required,omitempty" yaml:"required,omitempty"`
	MaxProperties         *uint64            `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties         *uint64            `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Properties            map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	PatternProperties     map[string]*Schema `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
	AdditionalProperties  *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	UnevaluatedProperties *bool              `json:"unevaluatedProperties,omitempty" yaml:"unevaluatedProperties,omitempty"`
	Const                 any                `json:"const,omitempty" yaml:"const,omitempty"`

	Defs map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
	// Deprecated: This field was renamed to "$defs" in draft 2019-09,
	// but the field is kept in this struct to allow bundled schemas to use them.
	Definitions map[string]*Schema `json:"definitions,omitempty" yaml:"definitions,omitempty"`

	SkipProperties   bool `json:"-" yaml:"-"`
	MergeProperties  bool `json:"-" yaml:"-"`
	Hidden           bool `json:"-" yaml:"-"`
	RequiredByParent bool `json:"-" yaml:"-"`
}

func (s *Schema) IsZero() bool {
	if s == nil {
		return true
	}
	switch {
	case s.kind != 0,
		len(s.Schema) > 0,
		len(s.ID) > 0,
		len(s.Title) > 0,
		len(s.Description) > 0,
		len(s.Comment) > 0,
		len(s.Examples) > 0,
		s.ReadOnly,
		s.Default != nil,
		len(s.Ref) > 0,
		s.Type != nil,
		len(s.Enum) > 0,
		len(s.AllOf) > 0,
		len(s.AnyOf) > 0,
		len(s.OneOf) > 0,
		s.Not != nil,
		s.Maximum != nil,
		s.Minimum != nil,
		s.MultipleOf != nil,
		len(s.Pattern) > 0,
		s.MaxLength != nil,
		s.MinLength != nil,
		s.MaxItems != nil,
		s.MinItems != nil,
		s.UniqueItems,
		s.Items != nil,
		s.AdditionalItems != nil,
		len(s.Required) > 0,
		s.MaxProperties != nil,
		s.MinProperties != nil,
		len(s.Properties) > 0,
		len(s.PatternProperties) > 0,
		s.AdditionalProperties != nil,
		s.UnevaluatedProperties != nil,
		len(s.Defs) > 0,
		len(s.Definitions) > 0,
		s.Const != nil:
		return false
	default:
		return true
	}
}

var (
	_ json.Unmarshaler = &Schema{}
	_ json.Marshaler   = &Schema{}
	_ yaml.Unmarshaler = &Schema{}
	_ yaml.Marshaler   = &Schema{}
)

// UnmarshalJSON implements [json.Unmarshaler].
func (s *Schema) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	// checking length to not create too many intermediate strings
	if len(trimmed) <= 5 {
		switch string(trimmed) {
		case "true":
			s.SetKind(SchemaKindTrue)
			return nil
		case "false":
			s.SetKind(SchemaKindFalse)
			return nil
		}
	}

	// Unmarshal using a new type to not cause infinite recursion when unmarshalling
	type schema Schema
	model := (*schema)(s)
	return json.Unmarshal(data, model)
}

// MarshalJSON implements [json.Marshaler].
func (s *Schema) MarshalJSON() ([]byte, error) {
	switch s.Kind() {
	case SchemaKindTrue:
		return []byte("true"), nil
	case SchemaKindFalse:
		return []byte("false"), nil
	default:
		type schema Schema
		return json.Marshal((*schema)(s))
	}
}

// UnmarshalYAML implements [yaml.Unmarshaler].
func (s *Schema) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.ShortTag() == "!!bool" {
		var b bool
		if err := value.Decode(&b); err != nil {
			return err
		}
		if b {
			s.SetKind(SchemaKindTrue)
		} else {
			s.SetKind(SchemaKindFalse)
		}
		return nil
	}

	// Unmarshal using a new type to not cause infinite recursion when unmarshalling
	type schema Schema
	model := (*schema)(s)
	return value.Decode(model)
}

// MarshalYAML implements [yaml.Marshaler].
func (s *Schema) MarshalYAML() (any, error) {
	switch s.Kind() {
	case SchemaKindTrue:
		return true, nil
	case SchemaKindFalse:
		return false, nil
	default:
		type schema Schema
		return (*schema)(s), nil
	}
}

func (s *Schema) Kind() SchemaKind {
	if s == nil {
		return SchemaKindObject
	}
	return s.kind
}

func (s *Schema) SetKind(kind SchemaKind) {
	if s == nil {
		panic(fmt.Errorf("Schema.SetKind(%#v): method receiver must not be nil", kind))
	}
	switch kind {
	case SchemaKindTrue:
		*s = Schema{kind: SchemaKindTrue} // will implicitly reset all other fields to zero
	case SchemaKindFalse:
		*s = Schema{kind: SchemaKindFalse} // will implicitly reset all other fields to zero
	case SchemaKindObject:
		s.kind = SchemaKindObject
	default:
		panic(fmt.Errorf("Schema.SetKind(%#v): unexpected kind", kind))
	}
}

func (s *Schema) IsType(t string) bool {
	switch value := s.Type.(type) {
	case []any:
		for _, v := range value {
			if v == t {
				return true
			}
		}
		return false
	default:
		return value == t
	}
}

func getSchemaURL(draft int) (string, error) {
	switch draft {
	case 4:
		return "http://json-schema.org/draft-04/schema#", nil
	case 6:
		return "http://json-schema.org/draft-06/schema#", nil
	case 7:
		return "http://json-schema.org/draft-07/schema#", nil
	case 2019:
		return "https://json-schema.org/draft/2019-09/schema", nil
	case 2020:
		return "https://json-schema.org/draft/2020-12/schema", nil
	default:
		return "", errors.New("invalid draft version. Please use one of: 4, 6, 7, 2019, 2020")
	}
}

func parseNode(ptr Ptr, keyNode, valNode *yaml.Node, useHelmDocs bool) (*Schema, error) {
	schema := &Schema{}

	var orderedMapProperties []*Schema

	switch valNode.Kind {
	case yaml.MappingNode:
		orderedMapProperties = make([]*Schema, 0, len(valNode.Content)/2)
		properties := make(map[string]*Schema, len(valNode.Content)/2)
		required := []string{}
		for i := 0; i < len(valNode.Content); i += 2 {
			childKeyNode := valNode.Content[i]
			childValNode := valNode.Content[i+1]
			childSchema, err := parseNode(ptr.Prop(childKeyNode.Value), childKeyNode, childValNode, useHelmDocs)
			if err != nil {
				return nil, err
			}

			// Exclude hidden child schemas
			if childSchema != nil && !childSchema.Hidden {
				if childSchema.SkipProperties && childSchema.IsType("object") {
					childSchema.Properties = nil
				}
				orderedMapProperties = append(orderedMapProperties, childSchema)
				properties[childKeyNode.Value] = childSchema
				if childSchema.RequiredByParent {
					required = append(required, childKeyNode.Value)
				}
			}
		}

		schema.Type = "object"
		schema.Properties = properties

		if len(required) > 0 {
			schema.Required = required
		}

	case yaml.SequenceNode:
		schema.Type = "array"

		mergedItemSchema := &Schema{}
		hasItems := false

		for i, itemNode := range valNode.Content {
			itemSchema, err := parseNode(ptr.Item(i), nil, itemNode, useHelmDocs)
			if err != nil {
				return nil, err
			}
			if itemSchema != nil && !itemSchema.Hidden {
				mergedItemSchema = mergeSchemas(mergedItemSchema, itemSchema)
				hasItems = true
			}
		}

		if hasItems {
			schema.Items = mergedItemSchema
		}

	case yaml.ScalarNode:
		if valNode.Style == yaml.DoubleQuotedStyle || valNode.Style == yaml.SingleQuotedStyle {
			schema.Type = "string"
		} else {
			schema.Type = getYAMLKind(valNode.Value)
		}
	}

	schemaComments, helmDocsComments := getComments(keyNode, valNode, useHelmDocs)

	if useHelmDocs {
		helmDocs, err := ParseHelmDocsComment(helmDocsComments)
		if err != nil {
			return nil, fmt.Errorf("%s: parse helm-docs comment: %w", ptr, err)
		}
		if len(helmDocs.Path) == 0 || ptr.Equals(NewPtr(helmDocs.Path...)) {
			schema.Description = helmDocs.Description
		}
	}

	if err := processComment(schema, schemaComments); err != nil {
		return nil, fmt.Errorf("%s: parse @schema comments: %w", ptr, err)
	}

	if schema.SkipProperties && schema.IsType("object") {
		schema.Properties = nil
	} else if schema.MergeProperties && len(orderedMapProperties) > 0 {
		result := orderedMapProperties[0]
		for _, prop := range orderedMapProperties[1:] {
			result = mergeSchemas(result, prop)
		}
		schema.AdditionalProperties = result
		schema.Properties = nil
	}

	return schema, nil
}

func (schema *Schema) Subschemas() iter.Seq2[Ptr, *Schema] {
	return func(yield func(Ptr, *Schema) bool) {
		for key, subSchema := range iterMapOrdered(schema.Properties) {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("properties", key), subSchema) {
				return
			}
		}
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Kind() == SchemaKindObject {
			if !yield(NewPtr("additionalProperties"), schema.AdditionalProperties) {
				return
			}
		}
		for key, subSchema := range iterMapOrdered(schema.PatternProperties) {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("patternProperties", key), subSchema) {
				return
			}
		}
		if schema.Items != nil && schema.Items.Kind() == SchemaKindObject {
			if !yield(NewPtr("items"), schema.Items) {
				return
			}
		}
		if schema.AdditionalItems != nil && schema.AdditionalItems.Kind() == SchemaKindObject {
			if !yield(NewPtr("additionalItems"), schema.AdditionalItems) {
				return
			}
		}
		for key, subSchema := range iterMapOrdered(schema.Defs) {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("$defs", key), subSchema) {
				return
			}
		}
		for key, subSchema := range iterMapOrdered(schema.Definitions) {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("definitions", key), subSchema) {
				return
			}
		}
		for index, subSchema := range schema.AllOf {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("allOf").Item(index), subSchema) {
				return
			}
		}
		for index, subSchema := range schema.AnyOf {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("anyOf").Item(index), subSchema) {
				return
			}
		}
		for index, subSchema := range schema.OneOf {
			if subSchema.Kind() == SchemaKindObject && !yield(NewPtr("anyOf").Item(index), subSchema) {
				return
			}
		}
		if schema.Not != nil {
			if schema.Not.Kind() == SchemaKindObject && !yield(NewPtr("not"), schema.Not) {
				return
			}
		}
	}
}

func (s *Schema) ParseRef() (*url.URL, error) {
	if s == nil || s.Ref == "" {
		return nil, nil
	}
	ref, err := url.Parse(s.Ref)
	if err != nil {
		return nil, err
	}
	if s.RefReferrer.IsZero() {
		return ref, nil
	}
	if ref.Scheme != "" && ref.Scheme != "file" {
		// Only have custom logic when $ref is a local file
		return ref, nil
	}
	refFile, err := ParseRefFileURL(ref)
	if err != nil {
		return nil, err
	}
	return s.RefReferrer.Join(refFile), nil
}

func (s *Schema) SetReferrer(ref Referrer) {
	if s == nil {
		return
	}
	for _, sub := range s.Subschemas() {
		sub.SetReferrer(ref)
	}
	if s.Ref != "" {
		s.RefReferrer = ref
	}
}

// Referrer holds information about what is referencing a schema.
// This is used when resolving $ref to load the appropriate files or URLs.
// Only one of "File" or "URL" should to be set at a time.
type Referrer struct {
	dir string
	url *url.URL
}

// ReferrerDir returns a [Referrer] using an path to a directory.
func ReferrerDir(dir string) Referrer {
	return Referrer{dir: dir}
}

// ReferrerURL returns a [Referrer] using a URL.
func ReferrerURL(url *url.URL) Referrer {
	// Clone it just to make sure we don't get any weird memory reuse bugs
	clone := *url
	return Referrer{url: &clone}
}

// IsZero returns true when neither File nor URL has been set.
func (r Referrer) IsZero() bool {
	return r == (Referrer{})
}

func (r Referrer) Join(refFile RefFile) *url.URL {
	if r.url != nil {
		urlClone := *r.url
		urlClone.Path = path.Join(urlClone.Path, refFile.Path)
		urlClone.Fragment = refFile.Frag
		return &urlClone
	}

	return &url.URL{
		Path:     path.Join(filepath.ToSlash(r.dir), refFile.Path),
		Fragment: refFile.Frag,
	}
}

// RefFile is a parsed "$ref: file://" schema property
type RefFile struct {
	Path string
	Frag string
}

func (r RefFile) String() string {
	if r.Frag != "" {
		return fmt.Sprintf("%s#%s", r.Path, r.Frag)
	}
	return r.Path
}

func ParseRefFile(ref string) (RefFile, error) {
	if ref == "" {
		return RefFile{}, nil
	}
	if after, ok := strings.CutPrefix(ref, "#"); ok {
		return RefFile{Frag: after}, nil
	}
	u, err := url.Parse(ref)
	if err != nil {
		return RefFile{}, err
	}
	return ParseRefFileURL(u)
}

func ParseRefFileURL(u *url.URL) (RefFile, error) {
	refFile, err := ParseRefFileURLAllowAbs(u)

	if strings.HasPrefix(refFile.Path, "/") {
		// Treat "/foo" & "file:///" as invalid
		return RefFile{}, fmt.Errorf("absolute paths not supported")
	}

	return refFile, err
}

func ParseRefFileURLAllowAbs(u *url.URL) (RefFile, error) {
	switch {
	case u.Scheme != "" && u.Scheme != "file":
		return RefFile{}, nil

	case u.RawQuery != "":
		return RefFile{}, fmt.Errorf("file query parameters not supported")

	case u.User != nil:
		return RefFile{}, fmt.Errorf("file URL user info not supported")

	case u.Scheme == "file" && u.Host == "" && u.Path == "":
		return RefFile{}, fmt.Errorf("unexpected empty file://")
	}

	clone := *u
	if clone.Host != "" {
		clone.Path = path.Join(u.Host, u.Path)
		clone.Host = ""
	}

	return RefFile{
		Path: clone.Path,
		Frag: clone.Fragment,
	}, nil
}
