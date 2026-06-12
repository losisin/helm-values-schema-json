package pkg

import (
	"fmt"
	"strings"
)

// ExpandRefs inlines all local $ref values (those starting with "#/") by replacing
// them with the referenced schema content. External $refs are left unchanged.
// Circular references are detected and left in place; their $defs entries are
// preserved so the remaining $ref values stay valid.
//
// This function will update the schema in-place.
//
// To also expand external $refs, first call [Bundle] with withoutIDs=true so that
// external references are converted to local "#/$defs/..." form.
func ExpandRefs(root *Schema) error {
	if root == nil {
		return fmt.Errorf("nil schema")
	}
	if err := expandRefsRec(root, root, nil); err != nil {
		return err
	}
	// Only remove defs that are no longer referenced.
	// Defs pointed to by circular $refs are preserved so those refs remain valid.
	RemoveUnusedDefs(root)
	return nil
}

// expandRefsRec recursively expands local $ref values in schema.
// ancestors tracks the $ref values currently being expanded to detect cycles.
func expandRefsRec(root, schema *Schema, ancestors []string) error {
	// Expand refs in subschemas, skipping $defs/$definitions (treated as
	// an immutable look-up table — we create copies when inlining, not mutate).
	for path, sub := range schema.Subschemas() {
		if len(path) > 0 && (path[0] == "$defs" || path[0] == "definitions") {
			continue
		}
		if err := expandRefsRec(root, sub, ancestors); err != nil {
			return err
		}
	}

	// Handle this schema's own $ref after all children are processed.
	if schema.Ref == "" || !strings.HasPrefix(schema.Ref, "#/") {
		return nil
	}

	// Cycle detection: if we're already expanding this ref, skip it.
	for _, a := range ancestors {
		if a == schema.Ref {
			return nil
		}
	}

	// Resolve the $ref pointer against the root schema.
	refPtr := ParsePtr(schema.Ref)
	matches := refPtr.Resolve(root)
	if len(matches) == 0 || !matches[len(matches)-1].Ptr.Equals(refPtr) {
		return fmt.Errorf("expand $ref %q: not found in schema", schema.Ref)
	}
	target := matches[len(matches)-1].Schema

	// Deep-copy the target to avoid mutating shared $defs entries.
	expanded := target.DeepCopy()
	// Clear identity fields that must not be duplicated inline.
	expanded.ID = ""
	expanded.Anchor = ""
	expanded.DynamicAnchor = ""

	// Recursively expand refs in the copy, adding current ref to ancestors.
	if err := expandRefsRec(root, &expanded, append(ancestors, schema.Ref)); err != nil {
		return err
	}

	// Clear $ref from current schema so mergeSchemas does not re-add it.
	ref := schema.Ref
	schema.Ref = ""
	schema.RefReferrer = Referrer{}

	// Merge: current schema fields win over the referenced content.
	// mergeSchemas(dest, src) — src overwrites dest, so we pass expanded as
	// dest and the (now ref-cleared) schema as src.
	mergeSchemas(&expanded, schema)

	// Replace the current schema's content with the merged result.
	*schema = expanded
	_ = ref
	return nil
}
