package pkg

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// Bundle will use default loader settings to bundle all $ref into $defs
// using [BundleSchema] and optionally remove any IDs using [BundleRemoveIDs].
//
// This function will update the schema in-place.
//
// The paths, outputDir & bundleRoot, are only used to change absolute paths
// into relative paths in a solely cosmetic way.
func Bundle(ctx context.Context, httpLoader Loader, schema *Schema, outputDir, bundleRoot string, withoutIDs bool) error {
	absOutputDir, err := filepath.Abs(filepath.Dir(filepath.FromSlash(outputDir)))
	if err != nil {
		return fmt.Errorf("output %s: get absolute path: %w", outputDir, err)
	}
	loader, root, err := NewDefaultLoader(bundleRoot, httpLoader)
	if err != nil {
		return err
	}
	defer root.Close()
	return bundleWithLoader(ctx, loader, schema, absOutputDir, withoutIDs)
}

func bundleWithLoader(ctx context.Context, loader Loader, schema *Schema, absOutputDir string, withoutIDs bool) error {
	if err := BundleSchema(ctx, loader, schema, absOutputDir); err != nil {
		return fmt.Errorf("bundle schemas: %w", err)
	}

	if withoutIDs {
		if err := BundleRemoveIDs(schema); err != nil {
			return fmt.Errorf("remove bundled $id: %w", err)
		}
	}

	// Cleanup unused $defs after all other bundling tasks
	RemoveUnusedDefs(schema)
	return nil
}

// BundleSchema will use the [Loader] to load any "$ref" references and
// store them in "$defs".
//
// This function will update the schema in-place.
//
// The basePathForIDs is an absolute path used to change the resulting
// $ref & $id absolute paths of bundled local files to relative paths.
// It is only used cosmetically and has no impact of how files are loaded.
func BundleSchema(ctx context.Context, loader Loader, schema *Schema, basePathForIDs string) error {
	if loader == nil {
		return fmt.Errorf("nil loader")
	}
	if schema == nil {
		return fmt.Errorf("nil schema")
	}
	return bundleSchemaRec(ctx, nil, loader, schema, schema, basePathForIDs)
}

func bundleSchemaRec(ctx context.Context, ptr Ptr, loader Loader, root, schema *Schema, basePathForIDs string) error {
	for path, subSchema := range schema.Subschemas() {
		ptr := ptr.Add(path)
		if err := bundleSchemaRec(ctx, ptr, loader, root, subSchema, basePathForIDs); err != nil {
			return err
		}
	}

	if schema.Ref == "" || strings.HasPrefix(schema.Ref, "#") {
		// Nothing to bundle
		return nil
	}
	for _, def := range root.Defs {
		if def.ID == trimFragment(schema.Ref) {
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
	//
	// It's fine to modify the $ref here, as it is not used any more times
	// after this. So changing it is solely a cosmetic change.
	schema.Ref = refRelativeToBasePath(ref, basePathForIDs).String()

	loaded, err := Load(ctx, loader, ref, basePathForIDs)
	if err != nil {
		return fmt.Errorf("%s: %w", ptr.Prop("$ref"), err)
	}
	if loaded == nil {
		return nil
	}
	if root.Defs == nil {
		root.Defs = map[string]*Schema{}
	}

	if newRef, ok := refRelativeToNearestID(ParsePtr(ref.Fragment), loaded); ok {
		schema.Ref = newRef
	}

	// Copy over $defs
	moveDefToRoot(root, &loaded.Defs)
	moveDefToRoot(root, &loaded.Definitions)

	// Add the value itself
	root.Defs[generateBundledName(loaded.ID, root.Defs)] = loaded

	return bundleSchemaRec(ctx, ptr, loader, root, loaded, basePathForIDs)
}

// refRelativeToNearestID takes in a [Ptr], tries to resolve it on the target schema,
// and outputs a $ref that points to the same target but from the nearest subschema
// that has an $id.
//
// Example:
//
//	refRelativeToNearestID(ParsePtr("foo.json#/definitions/bar/items"), &Schema{...})
//	// (where "/definitions/bar" has "$id": "hello")
//	// => "hello#/items"
func refRelativeToNearestID(ptr Ptr, loaded *Schema) (string, bool) {
	matches := ptr.Resolve(loaded)

	var newRootUsingID *Schema
	var suffix Ptr
	for _, match := range matches {
		if match.Schema.ID != "" {
			newRootUsingID = match.Schema
			// ignoring "ok" return because we already used this ptr when resolving
			// the matches, so it's guaranteed that [ptr] starts with
			suffix = ptr[len(match.Ptr):]
		}
	}

	if newRootUsingID == nil {
		return "", false
	}

	if len(suffix) == 0 {
		return newRootUsingID.ID, true
	}
	return fmt.Sprintf("%s#%s", newRootUsingID.ID, suffix), true
}

func refRelativeToBasePath(ref *url.URL, basePathForIDs string) *url.URL {
	refFile, err := ParseRefFileURLAllowAbs(ref)
	pathFromSlash := filepath.FromSlash(refFile.Path)
	if err != nil || refFile.Path == "" || !filepath.IsAbs(pathFromSlash) {
		return ref
	}
	rel, err := filepath.Rel(basePathForIDs, pathFromSlash)
	if err != nil {
		return ref
	}
	return &url.URL{
		Path:     filepath.ToSlash(filepath.Clean(rel)),
		Fragment: ref.Fragment,
	}
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
		if err := bundleChangeRefsRec(parentDefPtr, ptr.Add(subPath), root, subSchema); err != nil {
			return err
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
		return fmt.Errorf("%s: parse $ref as URL: %w", ptr.Prop("$ref"), err)
	}

	name, ok := findDefNameByRef(root.Defs, ref)
	if !ok {
		return fmt.Errorf("%s: no $defs found that matches $ref=%q", ptr.Prop("$ref"), ref.Redacted())
	}

	if ref.Fragment != "" {
		schema.Ref = fmt.Sprintf("#%s/%s", NewPtr("$defs", name), strings.TrimPrefix(ref.Fragment, "/"))
	} else {
		schema.Ref = fmt.Sprintf("#%s", NewPtr("$defs", name))
	}

	return nil
}

func findDefNameByRef(defs map[string]*Schema, ref *url.URL) (string, bool) {
	for name, def := range defs {
		if def.ID == trimFragmentURL(ref) {
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
		for _, match := range refPtr.Resolve(root) {
			refCounts[match.Schema]++
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

func trimFragment(ref string) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return trimFragmentURL(refURL)
}

func trimFragmentURL(ref *url.URL) string {
	refClone := *ref
	refClone.Fragment = ""
	return refClone.String()
}
