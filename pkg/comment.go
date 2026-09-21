package pkg

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

func getComments(keyNode, valNode *yaml.Node, useHelmDocs bool) (comments, helmDocs []string) {
	if keyNode != nil {
		if keyNode.HeadComment != "" {
			comments, helmDocs = SplitHelmDocsComment(keyNode.HeadComment)
			if !useHelmDocs {
				comments = append(comments, helmDocs...)
				helmDocs = nil
			}
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
	return comments, helmDocs
}

// annotation is a single "key: value" pair from a @schema comment.
// HasValue distinguishes "key" from "key:", which both leave Value empty but
// mean different things: only the former is the shorthand that reads the value
// off the YAML node.
type annotation struct {
	Key      string
	Value    string
	HasValue bool
}

func splitCommentsByParts(commentLines []string) iter.Seq[annotation] {
	return func(yield func(annotation) bool) {
		for _, comment := range commentLines {
			trimmed, ok := cutSchemaComment(comment)
			if !ok {
				continue
			}

			for part := range strings.SplitSeq(trimmed, ";") {
				key, value, hasValue := strings.Cut(part, ":")

				if !yield(annotation{
					Key:      strings.TrimSpace(key),
					Value:    strings.TrimSpace(value),
					HasValue: hasValue,
				}) {
					return
				}
			}
		}
	}
}

// cutSchemaComment turns this:
//
//	"# @schema foo bar"
//
// into this:
//
//	"foo bar"
func cutSchemaComment(line string) (string, bool) {
	withoutPound := strings.TrimSpace(strings.TrimPrefix(line, "#"))
	withoutSchema, ok := strings.CutPrefix(withoutPound, "@schema")
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(withoutSchema)
	if len(trimmed) == len(withoutSchema) {
		// this checks if we had "# @schemafoo" instead of "# @schema foo"
		// which works as we trimmed space before.
		// So the check is if len("foo") == len(" foo")
		return "", false
	}
	return trimmed, true
}

func getScalarType(shortTag string) string {
	switch shortTag {
	case "!!int":
		return "integer"
	case "!!float":
		return "number"
	case "!!bool":
		return "boolean"
	case "!!null":
		return "null"
	default:
		return "string"
	}
}

func processList(comment string, stringsOnly bool) []any {
	if strings.HasPrefix(comment, "[") {
		var list []any
		if err := yaml.Unmarshal([]byte(comment), &list); err == nil {
			if stringsOnly {
				convertScalarsToString(list)
			}
			return list
		}
	}

	if withoutLeft, ok := strings.CutPrefix(comment, "["); ok {
		comment = strings.TrimSuffix(withoutLeft, "]")
	}

	var list []any
	for item := range strings.SplitSeq(comment, ",") {
		trimmedItem := strings.TrimSpace(item)
		if !stringsOnly && trimmedItem == "null" {
			list = append(list, nil)
			continue
		}
		if strings.HasPrefix(trimmedItem, "\"") {
			if unquoted, err := strconv.Unquote(trimmedItem); err == nil {
				list = append(list, unquoted)
				continue
			}
		}
		trimmedItem = strings.Trim(trimmedItem, "\"")
		list = append(list, trimmedItem)
	}
	return list
}

func convertScalarsToString(slice []any) {
	for i, v := range slice {
		switch v := v.(type) {
		case nil:
			slice[i] = "null"
		case int, float64, bool:
			slice[i] = fmt.Sprint(v)
		case []any:
			convertScalarsToString(v)
		}
	}
}

func processComment(schema *Schema, commentLines []string, valNode *yaml.Node) error {
	// nullable is applied after the loop so it merges "null" into the final
	// type regardless of the order keywords appear in the comment.
	var nullable bool
	// Same for the const/default shorthands: they read the YAML node, and
	// skipProperties changes what that node contributes, so they cannot be
	// resolved until every annotation in the comment has been seen.
	var constShorthand, defaultShorthand bool
	for annot := range splitCommentsByParts(commentLines) {
		key, value := annot.Key, annot.Value
		switch key {
		case "enum":
			schema.Enum = processList(value, false)
		case "skipProperties":
			if err := processBoolComment(&schema.SkipProperties, value); err != nil {
				return fmt.Errorf("skipProperties: %w", err)
			}
		case "mergeProperties":
			if err := processBoolComment(&schema.MergeProperties, value); err != nil {
				return fmt.Errorf("mergeProperties: %w", err)
			}
		case "multipleOf":
			if err := processFloat64PtrComment(&schema.MultipleOf, value); err != nil {
				return fmt.Errorf("multipleOf: %w", err)
			}
			if schema.MultipleOf != nil && *schema.MultipleOf <= 0 {
				return fmt.Errorf("multipleOf: must be greater than zero")
			}
		case "maximum":
			if err := processFloat64PtrComment(&schema.Maximum, value); err != nil {
				return fmt.Errorf("maximum: %w", err)
			}
		case "minimum":
			if err := processFloat64PtrComment(&schema.Minimum, value); err != nil {
				return fmt.Errorf("minimum: %w", err)
			}
		case "maxLength":
			if err := processUint64PtrComment(&schema.MaxLength, value); err != nil {
				return fmt.Errorf("maxLength: %w", err)
			}
		case "minLength":
			if err := processUint64PtrComment(&schema.MinLength, value); err != nil {
				return fmt.Errorf("minLength: %w", err)
			}
		case "pattern":
			schema.Pattern = value
		case "maxItems":
			if err := processUint64PtrComment(&schema.MaxItems, value); err != nil {
				return fmt.Errorf("maxItems: %w", err)
			}
		case "minItems":
			if err := processUint64PtrComment(&schema.MinItems, value); err != nil {
				return fmt.Errorf("minItems: %w", err)
			}
		case "uniqueItems":
			if err := processBoolComment(&schema.UniqueItems, value); err != nil {
				return fmt.Errorf("uniqueItems: %w", err)
			}
		case "maxProperties":
			if err := processUint64PtrComment(&schema.MaxProperties, value); err != nil {
				return fmt.Errorf("maxProperties: %w", err)
			}
		case "minProperties":
			if err := processUint64PtrComment(&schema.MinProperties, value); err != nil {
				return fmt.Errorf("minProperties: %w", err)
			}
		case "patternProperties":
			if err := processObjectComment(&schema.PatternProperties, value); err != nil {
				return fmt.Errorf("patternProperties: %w", err)
			}
		case "required":
			if err := processBoolComment(&schema.RequiredByParent, value); err != nil {
				return fmt.Errorf("required: %w", err)
			}
		case "type":
			list := processList(value, true)
			schema.Type = list
			if len(list) == 1 {
				schema.Type = list[0]
			}
		case "nullable":
			if err := processBoolComment(&nullable, value); err != nil {
				return fmt.Errorf("nullable: %w", err)
			}
		case "title":
			schema.Title = value
		case "description":
			schema.Description = value
		case "examples":
			schema.Examples = processList(value, false)
		case "readOnly":
			if err := processBoolComment(&schema.ReadOnly, value); err != nil {
				return fmt.Errorf("readOnly: %w", err)
			}
		case "deprecated":
			if err := processBoolComment(&schema.Deprecated, value); err != nil {
				return fmt.Errorf("deprecated: %w", err)
			}
		case "default":
			if !annot.HasValue {
				defaultShorthand = true
				break
			}
			if err := processObjectComment(&schema.Default, value); err != nil {
				return fmt.Errorf("default: %w", err)
			}
		case "item":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			list := processList(value, true)
			schema.Items.Type = list
			if len(list) == 1 {
				schema.Items.Type = list[0]
			}
		case "itemProperties":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			if err := processObjectComment(&schema.Items.Properties, value); err != nil {
				return fmt.Errorf("itemProperties: %w", err)
			}
		case "itemRequired":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			itemRequired := processList(value, true)
			schema.Items.Required = make([]string, 0, len(itemRequired))
			for _, item := range itemRequired {
				required, ok := item.(string)
				if !ok {
					return fmt.Errorf("itemRequired: expected string, got %T", item)
				}
				schema.Items.Required = append(schema.Items.Required, required)
			}
		case "itemEnum":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			schema.Items.Enum = processList(value, false)
		case "itemPattern":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			schema.Items.Pattern = value
		case "itemRef":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			schema.Items.Ref = value
		case "additionalProperties":
			if strings.TrimSpace(value) == "" {
				schema.AdditionalProperties = SchemaTrue()
			} else if err := processObjectComment(&schema.AdditionalProperties, value); err != nil {
				return fmt.Errorf("additionalProperties: %w", err)
			}
		case "unevaluatedProperties":
			if strings.TrimSpace(value) == "" {
				schema.UnevaluatedProperties = SchemaTrue()
			} else if err := processObjectComment(&schema.UnevaluatedProperties, value); err != nil {
				return fmt.Errorf("unevaluatedProperties: %w", err)
			}
		case "$id":
			schema.ID = value
		case "$ref":
			schema.Ref = value
		case "hidden":
			if err := processBoolComment(&schema.Hidden, value); err != nil {
				return fmt.Errorf("hidden: %w", err)
			}
		case "allOf":
			if err := processObjectComment(&schema.AllOf, value); err != nil {
				return fmt.Errorf("allOf: %w", err)
			}
		case "anyOf":
			if err := processObjectComment(&schema.AnyOf, value); err != nil {
				return fmt.Errorf("anyOf: %w", err)
			}
		case "oneOf":
			if err := processObjectComment(&schema.OneOf, value); err != nil {
				return fmt.Errorf("oneOf: %w", err)
			}
		case "not":
			if err := processObjectComment(&schema.Not, value); err != nil {
				return fmt.Errorf("not: %w", err)
			}
		case "const":
			if !annot.HasValue {
				constShorthand = true
				break
			}
			if err := processObjectComment(&schema.Const, value); err != nil {
				return fmt.Errorf("const: %w", err)
			}
		default:
			return fmt.Errorf("unknown annotation %q", key)
		}
	}

	if constShorthand || defaultShorthand {
		// Report against whichever annotation asked for the shorthand, so the
		// error reads the same as the explicit form's.
		key := "const"
		if !constShorthand {
			key = "default"
		}
		if valNode == nil {
			return fmt.Errorf(`%s: parse object "": missing value`, key)
		}
		value, err := decodeValueNode(valNode)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if constShorthand {
			schema.Const = value
		}
		if defaultShorthand {
			schema.Default = value
		}
	}

	if nullable {
		schema.Type = appendNullType(schema.Type)
	}

	return nil
}

// decodeValueNode reads the value from the YAML node instead of from the
// "@schema annotation". For example:
//
//	foo: 123 # @schema default
//
// will use the YAML value 123.
//
// A property marked hidden is left out of the decoded value, so the shorthand
// cannot put back what the user asked to remove.
//
// skipProperties is not treated that way. It only drops the properties from
// the generated schema, and the decoded value still holds what the YAML
// defines. This matters when it is combined with $ref: the $ref supplies the
// schema, and default keeps the chart's own value.
func decodeValueNode(valNode *yaml.Node) (any, error) {
	switch valNode.Kind {
	case yaml.MappingNode:
		value := make(map[string]any, len(valNode.Content)/2)
		for i := 0; i+1 < len(valNode.Content); i += 2 {
			keyNode, childNode := valNode.Content[i], valNode.Content[i+1]
			hidden, err := valueNodeFlags(keyNode, childNode)
			if err != nil {
				return nil, err
			}
			if hidden {
				continue
			}
			child, err := decodeValueNode(childNode)
			if err != nil {
				return nil, err
			}
			value[keyNode.Value] = child
		}
		return value, nil

	case yaml.SequenceNode:
		value := make([]any, 0, len(valNode.Content))
		for _, itemNode := range valNode.Content {
			hidden, err := valueNodeFlags(nil, itemNode)
			if err != nil {
				return nil, err
			}
			if hidden {
				continue
			}
			item, err := decodeValueNode(itemNode)
			if err != nil {
				return nil, err
			}
			value = append(value, item)
		}
		return value, nil

	default:
		var value any
		if err := valNode.Decode(&value); err != nil {
			return nil, fmt.Errorf("decode YAML value: %w", err)
		}
		return value, nil
	}
}

// valueNodeFlags reports whether a node is annotated hidden, so
// [decodeValueNode] leaves it out of a shorthand const or default the same way
// the generated schema does.
func valueNodeFlags(keyNode, valNode *yaml.Node) (hidden bool, err error) {
	comments, _ := getComments(keyNode, valNode, false)
	for annot := range splitCommentsByParts(comments) {
		if annot.Key == "hidden" {
			if err := processBoolComment(&hidden, annot.Value); err != nil {
				return false, fmt.Errorf("hidden: %w", err)
			}
		}
	}
	return hidden, nil
}

// appendNullType adds "null" to the schema type, turning a single type into a
// list and leaving an existing "null" untouched. It backs the `nullable`
// annotation, e.g. `# @schema nullable` on a string value yields
// `"type": ["string", "null"]`.
func appendNullType(t any) any {
	switch v := t.(type) {
	case nil:
		return "null"
	case string:
		if v == "null" {
			return v
		}
		return []any{v, "null"}
	case []any:
		// v is the freshly built type list owned solely by schema.Type, so
		// appending in place is safe.
		for _, item := range v {
			if s, ok := item.(string); ok && s == "null" {
				return v
			}
		}
		return append(v, "null")
	default:
		return t
	}
}

func processObjectComment[T any](dest *T, comment string) error {
	comment = strings.TrimSpace(comment)
	switch comment {
	case "":
		return fmt.Errorf("parse object %q: missing value", comment)
	}
	var value T
	if err := yaml.Unmarshal([]byte(comment), &value); err != nil {
		return fmt.Errorf("parse object %q: %w", comment, err)
	}
	*dest = value
	return nil
}

func processBoolComment(dest *bool, comment string) error {
	switch strings.TrimSpace(comment) {
	case "true", "":
		*dest = true
		return nil
	case "false":
		*dest = false
		return nil
	default:
		return fmt.Errorf("invalid boolean %q, must be \"true\" or \"false\"", comment)
	}
}

func processUint64PtrComment(dest **uint64, comment string) error {
	comment = strings.TrimSpace(comment)
	if comment == "null" {
		*dest = nil
		return nil
	}
	if strings.HasPrefix(comment, "-") {
		return fmt.Errorf("invalid integer %q: negative values not allowed", comment)
	}
	num, err := strconv.ParseUint(comment, 10, 64)
	if err != nil {
		var numErr *strconv.NumError
		_ = errors.As(err, &numErr)
		// Reformat the error a little. Instead of this:
		// 	strconv.ParseUint: parsing "foo": invalid syntax
		// we get this:
		// 	invalid integer "foo": invalid syntax
		return fmt.Errorf("invalid integer %q: %w", comment, cmp.Or(numErr.Err, err))
	}
	*dest = &num
	return nil
}

func processFloat64PtrComment(dest **float64, comment string) error {
	comment = strings.TrimSpace(comment)
	if comment == "null" {
		*dest = nil
		return nil
	}
	num, err := strconv.ParseFloat(comment, 64)
	if err != nil {
		var numErr *strconv.NumError
		_ = errors.As(err, &numErr)
		// Reformat the error a little. Instead of this:
		// 	strconv.ParseUint: parsing "foo": invalid syntax
		// we get this:
		// 	invalid integer "foo": invalid syntax
		return fmt.Errorf("invalid number %q: %w", comment, cmp.Or(numErr.Err, err))
	}
	*dest = &num
	return nil
}
