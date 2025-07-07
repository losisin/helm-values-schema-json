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

func splitCommentsByParts(commentLines []string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, comment := range commentLines {
			trimmed, ok := cutSchemaComment(comment)
			if !ok {
				continue
			}

			for part := range strings.SplitSeq(trimmed, ";") {
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

func processComment(schema *Schema, commentLines []string) error {
	for key, value := range splitCommentsByParts(commentLines) {
		switch key {
		case "enum":
			schema.Enum = processList(value, false)
		case "skipProperties":
			if err := processBoolComment(&schema.SkipProperties, value); err != nil {
				return fmt.Errorf("skipProperties: %w", err)
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
		case "default":
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
		case "itemEnum":
			if schema.Items == nil {
				schema.Items = &Schema{}
			}
			schema.Items.Enum = processList(value, false)
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
			var b bool
			if err := processBoolComment(&b, value); err != nil {
				return fmt.Errorf("unevaluatedProperties: %w", err)
			}
			schema.UnevaluatedProperties = &b
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
			if err := processObjectComment(&schema.Const, value); err != nil {
				return fmt.Errorf("const: %w", err)
			}
		default:
			return fmt.Errorf("unknown annotation %q", key)
		}
	}

	return nil
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
