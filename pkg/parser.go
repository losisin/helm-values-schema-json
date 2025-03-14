package pkg

import (
	"errors"
	"reflect"
)

func mergeSchemas(dest, src *Schema) *Schema {
	if dest == nil {
		return src
	}
	if src == nil {
		return dest
	}

	// Resolve simple fields by favoring the fields from 'src' if they're provided
	if src.Type != "" {
		dest.Type = src.Type
	}
	if src.MultipleOf != nil {
		dest.MultipleOf = src.MultipleOf
	}
	if src.Maximum != nil {
		dest.Maximum = src.Maximum
	}
	if src.Minimum != nil {
		dest.Minimum = src.Minimum
	}
	if src.MaxLength != nil {
		dest.MaxLength = src.MaxLength
	}
	if src.MinLength != nil {
		dest.MinLength = src.MinLength
	}
	if src.Pattern != "" {
		dest.Pattern = src.Pattern
	}
	if src.MaxItems != nil {
		dest.MaxItems = src.MaxItems
	}
	if src.MinItems != nil {
		dest.MinItems = src.MinItems
	}
	if src.UniqueItems {
		dest.UniqueItems = src.UniqueItems
	}
	if src.MaxProperties != nil {
		dest.MaxProperties = src.MaxProperties
	}
	if src.MinProperties != nil {
		dest.MinProperties = src.MinProperties
	}
	if src.PatternProperties != nil {
		dest.PatternProperties = src.PatternProperties
	}
	if src.Title != "" {
		dest.Title = src.Title
	}
	if src.Description != "" {
		dest.Description = src.Description
	}
	if src.ReadOnly {
		dest.ReadOnly = src.ReadOnly
	}
	if src.Default != nil {
		dest.Default = src.Default
	}
	if src.AdditionalProperties != nil {
		dest.AdditionalProperties = src.AdditionalProperties
	}
	if src.UnevaluatedProperties != nil {
		dest.UnevaluatedProperties = src.UnevaluatedProperties
	}
	if src.ID != "" {
		dest.ID = src.ID
	}
	if src.Ref != "" {
		dest.Ref = src.Ref
	}
	if src.AllOf != nil {
		dest.AllOf = src.AllOf
	}
	if src.AnyOf != nil {
		dest.AnyOf = src.AnyOf
	}
	if src.OneOf != nil {
		dest.OneOf = src.OneOf
	}
	if src.Not != nil {
		dest.Not = src.Not
	}

	// Merge 'enum' field (assuming that maintaining order doesn't matter)
	dest.Enum = append(dest.Enum, src.Enum...)

	// Recursive calls for nested structures
	if src.Properties != nil {
		if dest.Properties == nil {
			dest.Properties = make(map[string]*Schema)
		}
		for propName, srcPropSchema := range src.Properties {
			if destPropSchema, exists := dest.Properties[propName]; exists {
				dest.Properties[propName] = mergeSchemas(destPropSchema, srcPropSchema)
			} else {
				dest.Properties[propName] = srcPropSchema
			}
		}
	}

	// 'required' array is combined uniquely
	dest.Required = uniqueStringAppend(dest.Required, src.Required...)

	// Merge 'items' if they exist (assuming they're not arrays)
	if src.Items != nil {
		dest.Items = mergeSchemas(dest.Items, src.Items)
	}

	return dest
}

func convertSchemaToMap(schema *Schema, noAdditionalProperties bool) (map[string]interface{}, error) {
	return convertSchemaToMapRec(schema, make(map[uintptr]bool), noAdditionalProperties)
}

func convertSchemaToMapRec(schema *Schema, visited map[uintptr]bool, noAdditionalProperties bool) (map[string]interface{}, error) {
	if schema == nil {
		return nil, nil
	}
	// Get the uintptr for the current schema pointer to identify it uniquely
	ptr := reflect.ValueOf(schema).Pointer()

	// If we've already visited this schema, we've found a circular reference
	if visited[ptr] {
		return nil, errors.New("circular reference detected in schema")
	}

	// Mark the current schema as visited
	visited[ptr] = true

	schemaMap := make(map[string]interface{})

	// Scalars
	if schema.Type != "" {
		schemaMap["type"] = schema.Type
	}
	if schema.MultipleOf != nil {
		schemaMap["multipleOf"] = *schema.MultipleOf
	}
	if schema.Maximum != nil {
		schemaMap["maximum"] = *schema.Maximum
	}
	if schema.Minimum != nil {
		schemaMap["minimum"] = *schema.Minimum
	}
	if schema.MaxLength != nil {
		schemaMap["maxLength"] = *schema.MaxLength
	}
	if schema.MinLength != nil {
		schemaMap["minLength"] = *schema.MinLength
	}
	if schema.Pattern != "" {
		schemaMap["pattern"] = schema.Pattern
	}
	if schema.MaxItems != nil {
		schemaMap["maxItems"] = *schema.MaxItems
	}
	if schema.MinItems != nil {
		schemaMap["minItems"] = *schema.MinItems
	}
	if schema.UniqueItems {
		schemaMap["uniqueItems"] = schema.UniqueItems
	}
	if schema.MaxProperties != nil {
		schemaMap["maxProperties"] = *schema.MaxProperties
	}
	if schema.MinProperties != nil {
		schemaMap["minProperties"] = *schema.MinProperties
	}
	if schema.Title != "" {
		schemaMap["title"] = schema.Title
	}
	if schema.Description != "" {
		schemaMap["description"] = schema.Description
	}
	if schema.ReadOnly {
		schemaMap["readOnly"] = schema.ReadOnly
	}
	if schema.Default != nil {
		schemaMap["default"] = schema.Default
	}
	if schema.AdditionalProperties != nil {
		schemaMap["additionalProperties"] = *schema.AdditionalProperties
	} else if noAdditionalProperties && schema.Type == "object" {
		schemaMap["additionalProperties"] = false
	}
	if schema.UnevaluatedProperties != nil {
		schemaMap["unevaluatedProperties"] = *schema.UnevaluatedProperties
	}
	if schema.ID != "" {
		schemaMap["$id"] = schema.ID
	}
	if schema.Ref != "" {
		schemaMap["$ref"] = schema.Ref
	}
	if schema.AllOf != nil {
		delete(schemaMap, "type")
		schemaMap["allOf"] = schema.AllOf
	}
	if schema.AnyOf != nil {
		delete(schemaMap, "type")
		schemaMap["anyOf"] = schema.AnyOf
	}
	if schema.OneOf != nil {
		delete(schemaMap, "type")
		schemaMap["oneOf"] = schema.OneOf
	}
	if schema.Not != nil {
		delete(schemaMap, "type")
		schemaMap["not"] = schema.Not
	}

	// Arrays
	if len(schema.Required) > 0 {
		schemaMap["required"] = schema.Required
	}
	if schema.Enum != nil {
		schemaMap["enum"] = schema.Enum
	}

	// Nested Schemas
	if schema.Items != nil {
		itemsMap, err := convertSchemaToMapRec(schema.Items, visited, noAdditionalProperties)
		if err != nil {
			return nil, err
		}
		schemaMap["items"] = itemsMap
	}
	if schema.Properties != nil {
		propertiesMap := make(map[string]interface{})
		for propName, propSchema := range schema.Properties {
			propMap, err := convertSchemaToMapRec(propSchema, visited, noAdditionalProperties)
			if err != nil {
				return nil, err
			}
			propertiesMap[propName] = propMap
		}
		schemaMap["properties"] = propertiesMap
	}

	if schema.PatternProperties != nil {
		patternPropertiesMap := make(map[string]interface{})
		for propName, propSchema := range schema.PatternProperties {
			propMap, err := convertSchemaToMapRec(propSchema, visited, noAdditionalProperties)
			if err != nil {
				return nil, err
			}
			patternPropertiesMap[propName] = propMap
		}
		schemaMap["patternProperties"] = patternPropertiesMap
	}

	delete(visited, ptr)

	return schemaMap, nil
}
