package pkg

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Schema struct {
	Type                  interface{}        `json:"type,omitempty" yaml:"type,omitempty"`
	Enum                  []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	MultipleOf            *float64           `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum               *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Minimum               *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	MaxLength             *uint64            `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength             *uint64            `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern               string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxItems              *uint64            `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems              *uint64            `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems           bool               `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxProperties         *uint64            `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties         *uint64            `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	PatternProperties     map[string]*Schema `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
	Required              []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Items                 *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	ItemsEnum             []any              `json:"itemsEnum,omitempty" yaml:"itemsEnum,omitempty"`
	ItemProperties        map[string]*Schema `json:"itemProperties,omitempty" yaml:"itemProperties,omitempty"`
	Properties            map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Title                 string             `json:"title,omitempty" yaml:"title,omitempty"`
	Description           string             `json:"description,omitempty" yaml:"description,omitempty"`
	ReadOnly              bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	Default               interface{}        `json:"default,omitempty" yaml:"default,omitempty"`
	AdditionalProperties  *bool              `json:"additionalProperties" yaml:"additionalProperties"`
	UnevaluatedProperties *bool              `json:"unevaluatedProperties" yaml:"unevaluatedProperties"`
	SkipProperties        bool               `json:"skipProperties,omitempty" yaml:"skipProperties,omitempty"`
	Hidden                bool               `json:"-" yaml:"-"`
	ID                    string             `json:"$id,omitempty" yaml:"$id,omitempty"`
	Ref                   string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Schema                string             `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Comment               string             `json:"$comment,omitempty" yaml:"$comment,omitempty"`
	Defs                  map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
	AllOf                 []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOf                 []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOf                 []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Not                   *Schema            `json:"not,omitempty" yaml:"not,omitempty"`

	// Deprecated: This field was renamed to "$defs" in draft 2019-09,
	// but the field is kept in this struct to allow bundled schemas to use them.
	Definitions map[string]*Schema `json:"definitions,omitempty" yaml:"definitions,omitempty"`
}

func getKind(value string) string {
	kindMapping := map[string]string{
		"boolean": "boolean",
		"integer": "integer",
		"number":  "number",
		"string":  "string",
	}

	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return kindMapping["integer"]
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return kindMapping["number"]
	}
	if _, err := strconv.ParseBool(value); err == nil {
		return kindMapping["boolean"]
	}
	if value != "" {
		return kindMapping["string"]
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

func getComment(keyNode *yaml.Node, valNode *yaml.Node) string {
	if valNode.LineComment != "" {
		return valNode.LineComment
	}
	if keyNode != nil {
		return keyNode.LineComment
	}
	return ""
}

func processList(comment string, stringsOnly bool) []interface{} {
	comment = strings.Trim(comment, "[]")
	items := strings.Split(comment, ",")

	var list []interface{}
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

func processComment(schema *Schema, comment string) (isRequired bool, isHidden bool) {
	isRequired = false
	isHidden = false

	parts := strings.Split(strings.TrimPrefix(comment, "# @schema "), ";")
	for _, part := range parts {
		keyValue := strings.SplitN(part, ":", 2)
		if len(keyValue) == 2 {
			key := strings.TrimSpace(keyValue[0])
			value := strings.TrimSpace(keyValue[1])

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
				var jsonObject interface{}
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
					schema.AdditionalProperties = &v
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
	}

	return isRequired, isHidden
}

func parseNode(keyNode *yaml.Node, valNode *yaml.Node) (*Schema, bool) {
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
			schema.Type = getKind(valNode.Value)
		}
	}

	propIsRequired, isHidden := processComment(schema, getComment(keyNode, valNode))
	if isHidden {
		return nil, false
	}

	if schema.SkipProperties && schema.Type == "object" {
		schema.Properties = nil
	}

	return schema, propIsRequired
}
