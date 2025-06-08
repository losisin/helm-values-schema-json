package pkg

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// BundleSchema will use the [Loader] to load any "$ref" references and
// store them in "$defs".
//
// This function will update the schema in-place.
func BundleSchema(ctx context.Context, loader Loader, schema *Schema, basePath string) error {
	if loader == nil {
		return fmt.Errorf("nil loader")
	}
	if schema == nil {
		return fmt.Errorf("nil schema")
	}
	return bundleSchemaRec(ctx, nil, loader, schema, schema, basePath)
}

func bundleSchemaRec(ctx context.Context, ptr Ptr, loader Loader, root, schema *Schema, basePath string) error {
	for path, subSchema := range schema.Subschemas() {
		ptr := ptr.Add(path)
		if err := bundleSchemaRec(ctx, ptr, loader, root, subSchema, basePath); err != nil {
			return err
		}
	}

	if schema.Ref == "" || strings.HasPrefix(schema.Ref, "#") {
		// Nothing to bundle
		return nil
	}
	for _, def := range root.Defs {
		if def.ID == bundleRefToID(schema.Ref) {
			// Already bundled
			return nil
		}
	}
	if schema.ID != "" {
		ctx = ContextWithLoaderReferrer(ctx, schema.ID)
	}
	ref, err := schema.ParseRef()
	if err != nil {
		return fmt.Errorf("%s: %w", ptr.Prop("$ref"), err)
	}

	// Make sure schema $ref corresponds with the corrected path
	if ref.Scheme == "" && ref.Path != "" {
		rel, err := filepath.Rel(basePath, ref.Path)
		if err != nil {
			return fmt.Errorf("%s: %w", ptr.Prop("$ref"), err)
		}
		schema.Ref = RefFile{Path: filepath.ToSlash(filepath.Clean(rel)), Fragment: ref.Fragment}.String()
	} else {
		schema.Ref = ref.String()
	}

	loaded, err := Load(ctx, loader, ref, basePath)
	if err != nil {
		return fmt.Errorf("%s: %w", ptr.Prop("$ref"), err)
	}
	if root.Defs == nil {
		root.Defs = map[string]*Schema{}
	}

	// Copy over $defs
	moveDefToRoot(root, &loaded.Defs)
	moveDefToRoot(root, &loaded.Definitions)

	// Add the value itself
	root.Defs[generateBundledName(loaded.ID, root.Defs)] = loaded

	return bundleSchemaRec(ctx, ptr, loader, root, loaded, basePath)
}

func moveDefToRoot(root *Schema, defs *map[string]*Schema) {
	for key, def := range *defs {
		if def.ID == "" {
			// Only move items that are referenced by $id.
			continue
		}
		root.Defs[generateBundledName(def.ID, root.Defs)] = def
		delete(*defs, key)
	}
	if len(*defs) == 0 {
		*defs = nil
	}
}

func generateBundledName(id string, defs map[string]*Schema) string {
	if id == "" {
		return ""
	}
	for name, def := range defs {
		if def.ID == id {
			return name
		}
	}
	baseName := path.Base(id)
	name := baseName
	i := 1
	for defs[name] != nil {
		i++
		name = fmt.Sprintf("%s_%d", baseName, i)
	}
	return name
}

// BundleRemoveIDs removes "$id" references to "$defs" and updates the "$ref"
// to point to the "$defs" elements directly inside the same document.
// This is non-standard behavior, but helps adding compatibility with
// non-compliant implementations such as the JSON & YAML language servers
// found in Visual Studio Code: https://github.com/microsoft/vscode-json-languageservice/issues/224
//
// For example, before:
//
//	{
//	  "$schema": "https://json-schema.org/draft/2020-12/schema",
//	  "properties": {
//	    "foo": {
//	      "$ref": "https://example.com/schema.json",
//	    }
//	  },
//	  "$defs": {
//	    "values.schema.json": {
//	      "$id": "https://example.com/schema.json"
//	    }
//	  }
//	}
//
// After:
//
//	{
//	  "$schema": "https://json-schema.org/draft/2020-12/schema",
//	  "properties": {
//	    "foo": {
//	      "$ref": "#/$defs/values.schema.json",
//	    }
//	  },
//	  "$defs": {
//	    "values.schema.json": {
//	    }
//	  }
//	}
//
// This function will update the schema in-place.
func BundleRemoveIDs(schema *Schema) error {
	if schema == nil {
		return fmt.Errorf("nil schema")
	}
	if err := bundleChangeRefsRec(nil, nil, schema, schema); err != nil {
		return err
	}
	for _, def := range schema.Defs {
		def.ID = ""
	}
	return nil
}

func bundleChangeRefsRec(parentDefPtr, ptr Ptr, root, schema *Schema) error {
	if schema.ID != "" {
		parentDefPtr = ptr
	}

	for subPath, subSchema := range schema.Subschemas() {
		ptr := ptr.Add(subPath)
		if err := bundleChangeRefsRec(parentDefPtr, ptr, root, subSchema); err != nil {
			return fmt.Errorf("%s: %w", ptr, err)
		}
	}

	if schema.Ref == "" || strings.HasPrefix(schema.Ref, "#") {
		if schema.Ref != "" && len(parentDefPtr) > 0 {
			// Update inline refs
			schema.Ref = fmt.Sprintf("#%s%s", parentDefPtr, strings.TrimPrefix(schema.Ref, "#"))
		}

		return nil
	}

	ref, err := url.Parse(schema.Ref)
	if err != nil {
		return fmt.Errorf("parse $ref=%q as URL: %w", schema.Ref, err)
	}

	name, ok := findDefNameByRef(root.Defs, ref)
	if !ok {
		return fmt.Errorf("no $defs found that matches $ref=%q", schema.Ref)
	}

	if ref.Fragment != "" {
		schema.Ref = fmt.Sprintf("#%s%s", NewPtr("$defs", name), ref.Fragment)
	} else {
		schema.Ref = fmt.Sprintf("#%s", NewPtr("$defs", name))
	}

	return nil
}

func findDefNameByRef(defs map[string]*Schema, ref *url.URL) (string, bool) {
	for name, def := range defs {
		if def.ID == trimFragment(ref) {
			return name, true
		}
	}
	return "", false
}

// RemoveUnusedDefs will try clean up all unused $defs to reduce the size of the
// final bundled schema.
func RemoveUnusedDefs(schema *Schema) {
	refCounts := map[*Schema]int{}
	for {
		clear(refCounts)
		findUnusedDefs(nil, schema, schema, refCounts)
		deletedCount := removeUnusedDefs(schema, refCounts)
		if deletedCount == 0 {
			break
		}
	}
}

func removeUnusedDefs(schema *Schema, refCounts map[*Schema]int) int {
	deletedCount := 0

	for _, def := range schema.Subschemas() {
		deletedCount += removeUnusedDefs(def, refCounts)
	}

	for name, def := range schema.Defs {
		if refCounts[def] == 0 {
			delete(schema.Defs, name)
			deletedCount++
		}
	}
	if len(schema.Defs) == 0 {
		schema.Defs = nil
	}

	for name, def := range schema.Definitions {
		if refCounts[def] == 0 {
			delete(schema.Definitions, name)
			deletedCount++
		}
	}
	if len(schema.Definitions) == 0 {
		schema.Definitions = nil
	}
	return deletedCount
}

func findUnusedDefs(ptr Ptr, root, schema *Schema, refCounts map[*Schema]int) {
	for path, def := range schema.Subschemas() {
		findUnusedDefs(ptr.Add(path), root, def, refCounts)
	}

	if schema.Ref == "" {
		return
	}

	if strings.HasPrefix(schema.Ref, "#/") {
		refPtr := ParsePtr(schema.Ref)
		if len(refPtr) > 0 && ptr.HasPrefix(refPtr) {
			// Ignore self-referential
			// E.g "#/$defs/foo.json/properties/moo" has $ref to "#/$defs/foo.json"
			return
		}
		for _, def := range resolvePtr(root, refPtr) {
			refCounts[def]++
		}
		return
	}

	ref, err := url.Parse(schema.Ref)
	if err != nil {
		return
	}

	if name, ok := findDefNameByRef(root.Defs, ref); ok {
		refCounts[root.Defs[name]]++
	}
}

func resolvePtr(schema *Schema, ptr Ptr) []*Schema {
	if schema == nil {
		return nil
	}
	if len(ptr) == 0 {
		return []*Schema{schema}
	}
	if len(ptr) < 2 {
		return []*Schema{schema}
	}
	switch ptr[0] {
	case "$defs":
		return append([]*Schema{schema}, resolvePtr(schema.Defs[ptr[1]], ptr[2:])...)
	case "definitions":
		return append([]*Schema{schema}, resolvePtr(schema.Definitions[ptr[1]], ptr[2:])...)
	default:
		return []*Schema{schema}
	}
}

func bundleRefToID(ref string) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return trimFragment(refURL)
}

func trimFragment(ref *url.URL) string {
	refClone := *ref
	refClone.Fragment = ""
	return refClone.String()
}
