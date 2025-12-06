package pkg

import (
	"bytes"
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
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

	dest.Schema = cmp.Or(src.Schema, dest.Schema)
	dest.ID = cmp.Or(src.ID, dest.ID)
	dest.Vocabulary = mergeMap(dest.Vocabulary, src.Vocabulary)
	dest.Anchor = cmp.Or(src.Anchor, dest.Anchor)
	dest.DynamicAnchor = cmp.Or(src.DynamicAnchor, dest.DynamicAnchor)
	dest.RecursiveAnchor = cmp.Or(src.RecursiveAnchor, dest.RecursiveAnchor)
	dest.Title = cmp.Or(src.Title, dest.Title)
	dest.Description = cmp.Or(src.Description, dest.Description)
	dest.Comment = cmp.Or(src.Comment, dest.Comment)
	if src.Examples != nil {
		dest.Examples = src.Examples
	}
	dest.Deprecated = dest.Deprecated || src.Deprecated
	dest.ReadOnly = dest.ReadOnly || src.ReadOnly
	dest.WriteOnly = dest.WriteOnly || src.WriteOnly
	if src.Default != nil {
		dest.Default = src.Default
	}
	if src.Ref != "" {
		dest.Ref = src.Ref
		dest.RefReferrer = src.RefReferrer
	}
	if src.DynamicRef != "" {
		dest.DynamicRef = src.DynamicRef
		dest.DynamicRefReferrer = src.DynamicRefReferrer
	}
	dest.RecursiveRef = cmp.Or(src.RecursiveRef, dest.RecursiveRef)
	dest.Type = cmp.Or(src.Type, dest.Type)
	dest.Const = cmp.Or(src.Const, dest.Const)
	dest.Enum = mergeEnum(dest.Enum, src.Enum)
	if src.AllOf != nil {
		dest.AllOf = src.AllOf
	}
	if src.AnyOf != nil {
		dest.AnyOf = src.AnyOf
	}
	if src.OneOf != nil {
		dest.OneOf = src.OneOf
	}
	dest.Not = cmp.Or(src.Not, dest.Not)
	dest.If = cmp.Or(src.If, dest.If)
	dest.Then = cmp.Or(src.Then, dest.Then)
	dest.Else = cmp.Or(src.Else, dest.Else)
	dest.ExclusiveMaximum = cmp.Or(src.ExclusiveMaximum, dest.ExclusiveMaximum)
	dest.Maximum = cmp.Or(src.Maximum, dest.Maximum)
	dest.ExclusiveMinimum = cmp.Or(src.ExclusiveMinimum, dest.ExclusiveMinimum)
	dest.Minimum = cmp.Or(src.Minimum, dest.Minimum)
	dest.MultipleOf = cmp.Or(src.MultipleOf, dest.MultipleOf)
	dest.Pattern = cmp.Or(src.Pattern, dest.Pattern)
	dest.Format = cmp.Or(src.Format, dest.Format)
	dest.MaxLength = cmp.Or(src.MaxLength, dest.MaxLength)
	dest.MinLength = cmp.Or(src.MinLength, dest.MinLength)
	dest.ContentEncoding = cmp.Or(src.ContentEncoding, dest.ContentEncoding)
	dest.ContentMediaType = cmp.Or(src.ContentMediaType, dest.ContentMediaType)
	dest.ContentSchema = cmp.Or(src.ContentSchema, dest.ContentSchema)
	dest.MaxItems = cmp.Or(src.MaxItems, dest.MaxItems)
	dest.MinItems = cmp.Or(src.MinItems, dest.MinItems)
	dest.UniqueItems = dest.UniqueItems || src.UniqueItems
	dest.MaxContains = cmp.Or(src.MaxContains, dest.MaxContains)
	dest.MinContains = cmp.Or(src.MinContains, dest.MinContains)
	dest.Contains = cmp.Or(src.Contains, dest.Contains)
	if src.PrefixItems != nil {
		dest.PrefixItems = src.PrefixItems
	}
	dest.Items = mergeSchemas(dest.Items, src.Items)
	dest.AdditionalItems = mergeSchemas(dest.AdditionalItems, src.AdditionalItems)
	dest.UnevaluatedItems = mergeSchemas(dest.UnevaluatedItems, src.UnevaluatedItems)
	dest.Required = uniqueStringAppend(dest.Required, src.Required)
	dest.MaxProperties = cmp.Or(src.MaxProperties, dest.MaxProperties)
	dest.MinProperties = cmp.Or(src.MinProperties, dest.MinProperties)
	dest.PropertyNames = cmp.Or(src.PropertyNames, dest.PropertyNames)
	dest.Properties = mergeSchemasMap(dest.Properties, src.Properties)
	dest.PatternProperties = mergeSchemasMap(dest.PatternProperties, src.PatternProperties)
	dest.AdditionalProperties = mergeSchemas(dest.AdditionalProperties, src.AdditionalProperties)
	dest.UnevaluatedProperties = mergeSchemas(dest.UnevaluatedProperties, src.UnevaluatedProperties)
	dest.DependentRequired = mergeMap(dest.DependentRequired, src.DependentRequired)
	dest.Dependencies = cmp.Or(src.Dependencies, dest.Dependencies)
	dest.DependentSchemas = mergeSchemasMap(dest.DependentSchemas, src.DependentSchemas)
	dest.Defs = mergeSchemasMap(dest.Defs, src.Defs)
	dest.Definitions = mergeSchemasMap(dest.Definitions, src.Definitions)

	dest.RequiredByParent = dest.RequiredByParent || src.RequiredByParent
	return dest
}

func mergeEnum(dest, src []any) []any {
	for _, value := range src {
		if enumContains(dest, value) {
			continue
		}
		dest = append(dest, value)
	}
	return dest
}

func enumContains(enums []any, elem any) bool {
	for _, value := range enums {
		if reflect.DeepEqual(value, elem) {
			return true
		}
	}
	return false
}

func mergeMap[K comparable, V any](dest, src map[K]V) map[K]V {
	if src == nil {
		return dest
	}
	if dest == nil {
		dest = make(map[K]V)
	}
	maps.Copy(dest, src)
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

func ensureCompliant(schema *Schema, noAdditionalProperties, noDefaultGlobal bool, draft int) error {
	if err := ensureCompliantRec(nil, schema, map[*Schema]struct{}{}, noAdditionalProperties, draft); err != nil {
		return err
	}

	if !noDefaultGlobal {
		addMissingGlobalProperty(schema) // only apply to schema root
	}
	return nil
}

func ensureCompliantRec(ptr Ptr, schema *Schema, visited map[*Schema]struct{}, noAdditionalProperties bool, draft int) error {
	if schema == nil {
		return nil
	}

	// If we've already visited this schema, we've found a circular reference
	if hasKey(visited, schema) {
		return fmt.Errorf("%s: circular reference detected in schema", ptr)
	}

	// Mark the current schema as visited
	visited[schema] = struct{}{}
	defer delete(visited, schema)

	for path, sub := range schema.Subschemas() {
		// continue recursively
		if err := ensureCompliantRec(ptr.Add(path), sub, visited, noAdditionalProperties, draft); err != nil {
			return err
		}
	}

	if schema.Kind().IsBool() {
		return nil
	}

	if err := validateType(ptr.Prop("type"), schema.Type); err != nil {
		return err
	}

	if schema.AdditionalProperties == nil && noAdditionalProperties && schema.IsType("object") {
		schema.AdditionalProperties = SchemaFalse()
	}

	switch {
	case len(schema.AllOf) > 0,
		len(schema.AnyOf) > 0,
		len(schema.OneOf) > 0,
		schema.Not != nil,
		schema.Const != nil:
		// These fields collide with "type"
		schema.Type = nil
	}

	if draft <= 7 && schema.Ref != "" {
		schemaClone := *schema
		schemaClone.Ref = ""
		if !schemaClone.IsZero() {
			*schema = Schema{
				AllOf: []*Schema{
					&schemaClone,
					{Ref: schema.Ref},
				},
			}
		}
	}

	return nil
}

func validateType(ptr Ptr, v any) error {
	switch v := v.(type) {
	case []any:
		var types []string
		for i, t := range v {
			ptr := ptr.Item(i)
			switch t := t.(type) {
			case string:
				if !isValidTypeString(t) {
					return fmt.Errorf("%s: invalid type %q, must be one of: array, boolean, integer, null, number, object, string", ptr, t)
				}
				if slices.Contains(types, t) {
					return fmt.Errorf("%s: type list must be unique, but found %q multiple times", ptr, t)
				}
				types = append(types, t)
			default:
				return fmt.Errorf("%s: type list must only contain strings", ptr)
			}
		}
		return nil
	case string:
		if !isValidTypeString(v) {
			return fmt.Errorf("%s: invalid type %q, must be one of: array, boolean, integer, null, number, object, string", ptr, v)
		}
		return nil
	case nil:
		return nil
	default:
		return fmt.Errorf("%s: type only be string or array of strings", ptr)
	}
}

func isValidTypeString(t string) bool {
	switch t {
	case "array", "boolean", "integer", "null", "number", "object", "string":
		return true
	default:
		return false
	}
}

func updateRefK8sAlias(schema *Schema, urlTemplate, version string) error {
	urlFunc := sync.OnceValues(func() (string, error) {
		if version == "" {
			return "", fmt.Errorf(`must set k8sSchemaVersion config when using "$ref: $k8s/...". For example pass --k8s-schema-version=v1.33.1 flag`)
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

// addMissingGlobalProperty adds /properties/global in case
// the resulting does not allow additional object properties
func addMissingGlobalProperty(schema *Schema) {
	switch {
	case
		// ignore invalid argument
		schema == nil,
		// "global" property already set
		hasKey(schema.Properties, "global"),
		// `"additionalProperties": null`
		schema.AdditionalProperties == nil,
		// `"additionalProperties": true`
		schema.AdditionalProperties.Kind() == SchemaKindTrue,
		// `"additionalProperties": {"type": ["object", ...]}`
		schema.AdditionalProperties.Kind() != SchemaKindFalse && schema.AdditionalProperties.IsType("object"),
		// `"additionalProperties": {"type": null}` (aka "allow any type")
		schema.AdditionalProperties.Kind() != SchemaKindFalse && schema.AdditionalProperties.Type == nil:
		return
	}

	if schema.Properties == nil {
		schema.Properties = map[string]*Schema{}
	}
	schema.Properties["global"] = defaultGlobal()
}

// defaultGlobal returns the default "global" property subschema used by [addMissingGlobalProperty].
func defaultGlobal() *Schema {
	return &Schema{
		Type:        []any{"object", "null"},
		Comment:     "Added automatically by 'helm schema' to allow this chart to be used as a Helm dependency, as the `additionalProperties` setting would otherwise collide with Helm's special 'global' values key.",
		Description: "Global values shared between all subcharts",
	}
}
