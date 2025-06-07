package pkg

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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

var (
	SchemaTrue  = Schema{kind: SchemaKindTrue}
	SchemaFalse = Schema{kind: SchemaKindFalse}
)

func SchemaBool(value bool) *Schema {
	if value {
		return &SchemaTrue
	}
	return &SchemaFalse
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
	ReadOnly              bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	Default               any        `json:"default,omitempty" yaml:"default,omitempty"`
	Ref                   string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type                  any        `json:"type,omitempty" yaml:"type,omitempty"`
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

	Defs map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
	// Deprecated: This field was renamed to "$defs" in draft 2019-09,
	// but the field is kept in this struct to allow bundled schemas to use them.
	Definitions map[string]*Schema `json:"definitions,omitempty" yaml:"definitions,omitempty"`

	SkipProperties bool `json:"-" yaml:"-"`
	Hidden         bool `json:"-" yaml:"-"`
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
		len(s.Definitions) > 0:
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
	type SchemaWithoutUnmarshaler Schema
	model := (*SchemaWithoutUnmarshaler)(s)
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
		type SchemaWithoutMarshaler Schema
		return json.Marshal((*SchemaWithoutMarshaler)(s))
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
	type SchemaWithoutUnmarshaler Schema
	model := (*SchemaWithoutUnmarshaler)(s)
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
		type SchemaWithoutMarshaler Schema
		return (*SchemaWithoutMarshaler)(s), nil
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
		panic(fmt.Errorf("Schema.SetKind(%#v): method reciever must not be nil", kind))
	}
	switch kind {
	case SchemaKindTrue:
		*s = SchemaTrue // will implicitly reset all other fields to zero
	case SchemaKindFalse:
		*s = SchemaFalse // will implicitly reset all other fields to zero
	case SchemaKindObject:
		s.kind = SchemaKindObject
	default:
		panic(fmt.Errorf("Schema.SetKind(%#v): unexpected kind", kind))
	}
}

func getYAMLKind(value string) string {
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "integer"
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "number"
	}
	if _, err := strconv.ParseBool(value); err == nil {
		return "boolean"
	}
	if value != "" {
		return "string"
	}
	return "null"
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

func getComments(keyNode, valNode *yaml.Node) []string {
	var comments []string
	if keyNode != nil {
		if keyNode.HeadComment != "" {
			comment := keyNode.HeadComment
			if index := strings.LastIndex(comment, "\n\n"); index != -1 {
				// Splits after the last "comment group". In other words, given this:
				//
				//	# foo
				//	# bar
				//
				//	# moo
				//	# doo
				//	hello: ""
				//
				// Then only consider the last "# moo" & "# doo" comments
				comment = comment[index+2:] // +2 to get rid of the "\n\n"
			}
			comments = strings.Split(comment, "\n")
		}
		if keyNode.LineComment != "" {
			comments = append(comments, keyNode.LineComment)
		}
	}
	if valNode.LineComment != "" {
		comments = append(comments, valNode.LineComment)
	}
	if keyNode != nil {
		// Append last as they come last
		if keyNode.FootComment != "" {
			comments = append(comments, strings.Split(keyNode.FootComment, "\n")...)
		}
	}
	return comments
}

func processList(comment string, stringsOnly bool) []any {
	comment = strings.Trim(comment, "[]")
	items := strings.Split(comment, ",")

	var list []any
	for _, item := range items {
		trimmedItem := strings.TrimSpace(item)
		if !stringsOnly && trimmedItem == "null" {
			list = append(list, nil)
		} else {
			trimmedItem = strings.Trim(trimmedItem, "\"")
			list = append(list, trimmedItem)
		}
	}
	return list
}

func processComment(schema *Schema, commentLines []string) (isRequired, isHidden bool) {
	isRequired = false
	isHidden = false

	for key, value := range splitCommentByParts(commentLines) {
		switch key {
		case "enum":
			schema.Enum = processList(value, false)
		case "multipleOf":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				if v > 0 {
					schema.MultipleOf = &v
				}
			}
		case "maximum":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				schema.Maximum = &v
			}
		case "skipProperties":
			if v, err := strconv.ParseBool(value); err == nil && v {
				schema.SkipProperties = true
			}
		case "minimum":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				schema.Minimum = &v
			}
		case "maxLength":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				schema.MaxLength = &v
			}
		case "minLength":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				schema.MinLength = &v
			}
		case "pattern":
			schema.Pattern = value
		case "maxItems":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				schema.MaxItems = &v
			}
		case "minItems":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				schema.MinItems = &v
			}
		case "uniqueItems":
			if v, err := strconv.ParseBool(value); err == nil {
				schema.UniqueItems = v
			}
		case "maxProperties":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				schema.MaxProperties = &v
			}
		case "minProperties":
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				schema.MinProperties = &v
			}
		case "patternProperties":
			var jsonObject map[string]*Schema
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.PatternProperties = jsonObject
			}
		case "required":
			if strings.TrimSpace(value) == "true" {
				isRequired = true
			}
		case "type":
			schema.Type = processList(value, true)
		case "title":
			schema.Title = value
		case "description":
			schema.Description = value
		case "readOnly":
			if v, err := strconv.ParseBool(value); err == nil {
				schema.ReadOnly = v
			}
		case "default":
			var jsonObject any
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.Default = jsonObject
			}
		case "item":
			schema.Items = &Schema{
				Type: value,
			}
		case "itemProperties":
			if schema.Items.Type == "object" {
				var itemProps map[string]*Schema
				if err := json.Unmarshal([]byte(value), &itemProps); err == nil {
					schema.Items.Properties = itemProps
				}
			}
		case "itemEnum":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			schema.Items.Enum = processList(value, false)
		case "additionalProperties":
			if v, err := strconv.ParseBool(value); err == nil {
				if v {
					schema.AdditionalProperties = &SchemaTrue
				} else {
					schema.AdditionalProperties = &SchemaFalse
				}
			}
		case "unevaluatedProperties":
			if v, err := strconv.ParseBool(value); err == nil {
				schema.UnevaluatedProperties = &v
			}
		case "$id":
			schema.ID = value
		case "$ref":
			schema.Ref = value
		case "hidden":
			if v, err := strconv.ParseBool(value); err == nil && v {
				isHidden = true
			}
		case "allOf":
			var jsonObject []*Schema
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.AllOf = jsonObject
			}
		case "anyOf":
			var jsonObject []*Schema
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.AnyOf = jsonObject
			}
		case "oneOf":
			var jsonObject []*Schema
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.OneOf = jsonObject
			}
		case "not":
			var jsonObject *Schema
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.Not = jsonObject
			}
		}
	}

	return isRequired, isHidden
}

func splitCommentByParts(commentLines []string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, comment := range commentLines {
			withoutPound := strings.TrimSpace(strings.TrimPrefix(comment, "#"))
			withoutSchema, ok := strings.CutPrefix(withoutPound, "@schema")
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(withoutSchema)
			if len(trimmed) == len(withoutSchema) {
				// this checks if we had "# @schemafoo" instead of "# @schema foo"
				// which works as we trimmed space before.
				// So the check is if len("foo") == len(" foo")
				continue
			}

			parts := strings.Split(trimmed, ";")
			for _, part := range parts {
				key, value, _ := strings.Cut(part, ":")
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)

				if !yield(key, value) {
					return
				}
			}
		}
	}
}

func parseNode(keyNode, valNode *yaml.Node) (*Schema, bool) {
	schema := &Schema{}

	switch valNode.Kind {
	case yaml.MappingNode:
		properties := make(map[string]*Schema)
		required := []string{}
		for i := 0; i < len(valNode.Content); i += 2 {
			childKeyNode := valNode.Content[i]
			childValNode := valNode.Content[i+1]
			childSchema, childRequired := parseNode(childKeyNode, childValNode)

			// Exclude hidden child schemas
			if childSchema != nil && !childSchema.Hidden {
				if childSchema.SkipProperties && childSchema.Type == "object" {
					childSchema.Properties = nil
				}
				properties[childKeyNode.Value] = childSchema
				if childRequired {
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

		for _, itemNode := range valNode.Content {
			itemSchema, _ := parseNode(nil, itemNode)
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

	propIsRequired, isHidden := processComment(schema, getComments(keyNode, valNode))
	if isHidden {
		return nil, false
	}

	if schema.SkipProperties && schema.Type == "object" {
		schema.Properties = nil
	}

	return schema, propIsRequired
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

func iterMapOrdered[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
