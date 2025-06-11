package pkg

import (
	"encoding/json"
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

func processComment(schema *Schema, commentLines []string) (isRequired, isHidden bool) {
	isRequired = false
	isHidden = false

	for key, value := range splitCommentsByParts(commentLines) {
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
			if v, err := strconv.ParseBool(value); err == nil {
				schema.ReadOnly = v
			}
		case "default":
			var jsonObject any
			if err := json.Unmarshal([]byte(value), &jsonObject); err == nil {
				schema.Default = jsonObject
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
			if schema.Items.IsType("object") {
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
