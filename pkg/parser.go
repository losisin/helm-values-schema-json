package pkg

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

func mergeSchemas(dest, src *Schema) *Schema {
	if dest == nil {
		return src
	}
	if src == nil {
		return dest
	}

	dest.SetKind(src.Kind())

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
		dest.AdditionalProperties = mergeSchemas(dest.AdditionalProperties, src.AdditionalProperties)
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
	if src.Schema != "" {
		dest.Schema = src.Schema
	}
	if src.Comment != "" {
		dest.Comment = src.Comment
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
	dest.Properties = mergeSchemasMap(dest.Properties, src.Properties)
	dest.Defs = mergeSchemasMap(dest.Defs, src.Defs)
	dest.Definitions = mergeSchemasMap(dest.Definitions, src.Definitions)

	// 'required' array is combined uniquely
	dest.Required = uniqueStringAppend(dest.Required, src.Required...)

	// Merge 'items' if they exist (assuming they're not arrays)
	if src.Items != nil {
		dest.Items = mergeSchemas(dest.Items, src.Items)
	}
	if src.AdditionalItems != nil {
		dest.AdditionalItems = mergeSchemas(dest.AdditionalItems, src.AdditionalItems)
	}

	return dest
}

func mergeSchemasMap(dest, src map[string]*Schema) map[string]*Schema {
	if src != nil {
		if dest == nil {
			dest = make(map[string]*Schema)
		}
		for defName, srcDefSchema := range src {
			if destDefSchema, exists := dest[defName]; exists {
				dest[defName] = mergeSchemas(destDefSchema, srcDefSchema)
			} else {
				dest[defName] = srcDefSchema
			}
		}
	}
	return dest
}

func ensureCompliant(schema *Schema, noAdditionalProperties bool) error {
	return ensureCompliantRec(nil, schema, map[*Schema]struct{}{}, noAdditionalProperties)
}

func ensureCompliantRec(ptr Ptr, schema *Schema, visited map[*Schema]struct{}, noAdditionalProperties bool) error {
	if schema == nil {
		return nil
	}

	// If we've already visited this schema, we've found a circular reference
	if _, ok := visited[schema]; ok {
		return fmt.Errorf("%s: circular reference detected in schema", ptr)
	}

	// Mark the current schema as visited
	visited[schema] = struct{}{}
	defer delete(visited, schema)

	for path, sub := range schema.Subschemas() {
		// continue recursively
		if err := ensureCompliantRec(ptr.Add(path), sub, visited, noAdditionalProperties); err != nil {
			return err
		}
	}

	if schema.Kind().IsBool() {
		return nil
	}

	if schema.AdditionalProperties == nil && noAdditionalProperties && schema.Type == "object" {
		schema.AdditionalProperties = &SchemaFalse
	}

	switch {
	case len(schema.AllOf) > 0,
		len(schema.AnyOf) > 0,
		len(schema.OneOf) > 0,
		schema.Not != nil:
		// These fields collide with "type"
		schema.Type = nil
	}

	return nil
}

func updateRefK8sAlias(schema *Schema, urlTemplate, version string) error {
	urlFunc := sync.OnceValues(func() (string, error) {
		if version == "" {
			return "", fmt.Errorf(`must set k8sSchemaVersion config when using "$ref: $k8s/...". For example pass --k8sSchemaVersion=v1.33.1 flag`)
		}
		tpl, err := template.New("").Parse(urlTemplate)
		if err != nil {
			return "", fmt.Errorf("parse k8sSchemaURL template: %w", err)
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, struct{ K8sSchemaVersion string }{K8sSchemaVersion: version}); err != nil {
			return "", fmt.Errorf("template k8sSchemaURL: %w", err)
		}
		return buf.String(), nil
	})
	return updateRefK8sAliasRec(nil, schema, urlFunc)
}

func updateRefK8sAliasRec(ptr Ptr, schema *Schema, urlFunc func() (string, error)) error {
	for path, sub := range schema.Subschemas() {
		// continue recursively
		if err := updateRefK8sAliasRec(ptr.Add(path), sub, urlFunc); err != nil {
			return err
		}
	}

	withoutFragment, _, _ := strings.Cut(schema.Ref, "#")
	if withoutFragment == "$k8s" || withoutFragment == "$k8s/" {
		return fmt.Errorf("%s: invalid $k8s schema alias: must have a path but only got %q", ptr, schema.Ref)
	}

	withoutAlias, ok := strings.CutPrefix(schema.Ref, "$k8s/")
	if !ok {
		return nil
	}

	urlPrefix, err := urlFunc()
	if err != nil {
		return fmt.Errorf("%s: %w", ptr, err)
	}

	schema.Ref = fmt.Sprintf("%s/%s", strings.TrimSuffix(urlPrefix, "/"), withoutAlias)
	return nil
}
