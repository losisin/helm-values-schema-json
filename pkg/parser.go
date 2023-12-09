package pkg

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

func convertSchemaToMap(schema *Schema) (map[string]interface{}, error) {
	if schema == nil {
		return nil, nil
	}

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

	// Arrays
	if len(schema.Required) > 0 {
		schemaMap["required"] = schema.Required
	}
	if schema.Enum != nil {
		schemaMap["enum"] = schema.Enum
	}

	// Nested Schemas
	if schema.Items != nil {
		itemsMap, err := convertSchemaToMap(schema.Items)
		if err != nil {
			return nil, err
		}
		schemaMap["items"] = itemsMap
	}
	if schema.Properties != nil {
		propertiesMap := make(map[string]interface{})
		for propName, propSchema := range schema.Properties {
			propMap, err := convertSchemaToMap(propSchema)
			if err != nil {
				return nil, err
			}
			propertiesMap[propName] = propMap
		}
		schemaMap["properties"] = propertiesMap
	}

	return schemaMap, nil
}
