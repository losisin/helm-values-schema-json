/*
	This file contains some modified versions of the
	santhosh-tekuri/jsonschema loader code, licensed
	under the Apache License, Version 2.0

	Based on code from:
	- https://github.com/santhosh-tekuri/jsonschema/blob/87df339550a7b2440ff7da286bd34ece7d74039b/loader.go
	- https://github.com/santhosh-tekuri/jsonschema/blob/87df339550a7b2440ff7da286bd34ece7d74039b/cmd/jv/loader.go
*/

package pkg

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func NewDefaultLoader(client *http.Client, root *os.Root, basePath string) Loader {
	fileLoader := NewFileLoader(root, basePath)
	httpLoader := NewHTTPLoader(client)
	return NewCacheLoader(URLSchemeLoader{
		"http":  httpLoader,
		"https": httpLoader,
		"file":  fileLoader, // Used for "file:///some/abs/path"
		"":      fileLoader, // Used for "./foobar.json" or "/some/abs/path"
	})
}

// BundleSchema will use the [Loader] to load any "$ref" references and
// store them in "$defs".
//
// This function will update the schema in-place.
func BundleSchema(ctx context.Context, loader Loader, schema *Schema) error {
	if loader == nil {
		return fmt.Errorf("nil loader")
	}
	if schema == nil {
		return fmt.Errorf("nil schema")
	}
	return bundleSchemaRec(ctx, nil, loader, schema, schema)
}

func bundleSchemaRec(ctx context.Context, ptr Ptr, loader Loader, root, schema *Schema) error {
	for path, subSchema := range schema.Subschemas() {
		ptr := ptr.Add(path)
		if err := bundleSchemaRec(ctx, ptr, loader, root, subSchema); err != nil {
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
	loaded, err := Load(ctx, loader, schema.Ref)
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

	return bundleSchemaRec(ctx, ptr, loader, root, loaded)
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
		if def.ID == bundleRefURLToID(ref) {
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

func Load(ctx context.Context, loader Loader, ref string) (*Schema, error) {
	if loader == nil {
		return nil, fmt.Errorf("nil loader")
	}
	if ref == "" {
		return nil, fmt.Errorf("cannot load empty $ref")
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("parse $ref as URL: %w", err)
	}
	schema, err := loader.Load(ctx, refURL)
	if err != nil {
		return nil, err
	}

	schema.ID = bundleRefURLToID(refURL)
	return schema, nil
}

func bundleRefToID(ref string) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return bundleRefURLToID(refURL)
}

func bundleRefURLToID(ref *url.URL) string {
	refClone := *ref
	refClone.Fragment = ""
	return refClone.String()
}

func FixRootSchemaRef(rootSchemaRef, filePath string) string {
	if rootSchemaRef == "" {
		return ""
	}
	parsed, err := url.Parse(pathWindowsFix(rootSchemaRef))
	if err != nil || parsed.Scheme != "" {
		return rootSchemaRef
	}
	relPath, err := filepath.Rel(filepath.Dir(filePath), parsed.Path)
	if err != nil {
		err := fmt.Errorf("tried to fix root schema $ref path for bundling: get relative path from file %q to schema root ref %q: %w", filePath, rootSchemaRef, err)
		fmt.Println("Warning:", err)
		return rootSchemaRef
	}
	relPath = filepath.ToSlash(relPath)
	if !strings.HasPrefix(relPath, "./") && !strings.HasPrefix(relPath, "../") {
		relPath = "./" + relPath
	}
	return relPath
}

type Loader interface {
	Load(ctx context.Context, ref *url.URL) (*Schema, error)
}

// DummyLoader is a dummy implementation of [Loader] meant to be
// used in tests.
type DummyLoader struct {
	LoadFunc func(ctx context.Context, ref *url.URL) (*Schema, error)
}

var _ Loader = DummyLoader{}

// Load implements [Loader].
func (loader DummyLoader) Load(ctx context.Context, ref *url.URL) (*Schema, error) {
	return loader.LoadFunc(ctx, ref)
}

// FileLoader loads a schema from a "$ref: file:/some/path" reference
// from the local file-system.
type FileLoader struct {
	root     *os.Root
	basePath string
}

func NewFileLoader(root *os.Root, basePath string) FileLoader {
	return FileLoader{
		root:     root,
		basePath: basePath,
	}
}

var _ Loader = FileLoader{}

// Load implements [Loader].
func (loader FileLoader) Load(_ context.Context, ref *url.URL) (*Schema, error) {
	if ref.Scheme != "file" && ref.Scheme != "" {
		return nil, fmt.Errorf(`file url in $ref=%q must start with "file://", "./", or "/"`, ref)
	}
	if ref.Path == "" {
		return nil, fmt.Errorf(`file url in $ref=%q must contain a path`, ref)
	}
	path := pathWindowsFix(ref.Path)
	if loader.basePath != "" && !filepath.IsAbs(path) {
		path = filepath.Join(loader.basePath, path)
	}

	fmt.Println("Loading file", path)
	f, err := loader.root.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open $ref=%q file: %w", ref, err)
	}
	defer closeIgnoreError(f)
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read $ref=%q file: %w", ref, err)
	}

	fmt.Printf("=> got %s\n", formatSizeBytes(len(b)))

	switch filepath.Ext(path) {
	case ".yml", ".yaml":
		var schema Schema
		if err := yaml.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q YAML file: %w", ref, err)
		}
		return &schema, nil
	default:
		var schema Schema
		if err := json.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q JSON file: %w", ref, err)
		}
		return &schema, nil
	}
}

func pathWindowsFix(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/")
		path = filepath.FromSlash(path)
	}
	return path
}

// URLSchemeLoader delegates to other [Loader] implementations
// based on the [url.URL] scheme.
type URLSchemeLoader map[string]Loader

var _ Loader = URLSchemeLoader{}

// Load implements [Loader].
func (loader URLSchemeLoader) Load(ctx context.Context, ref *url.URL) (*Schema, error) {
	loaderForScheme, ok := loader[ref.Scheme]
	if !ok {
		return nil, fmt.Errorf("%w: cannot load schema from $ref=%q, supported schemes: %v",
			errors.ErrUnsupported, ref, strings.Join(slices.Collect(maps.Keys(loader)), ","))
	}
	return loaderForScheme.Load(ctx, ref)
}

// CacheLoader stores loaded schemas in memory and reuses (or "memoizes", if you will)
// calls to the underlying [Loader].
type CacheLoader struct {
	schemas   map[string]*Schema
	subLoader Loader
}

func NewCacheLoader(loader Loader) *CacheLoader {
	return &CacheLoader{
		schemas:   map[string]*Schema{},
		subLoader: loader,
	}
}

var _ Loader = CacheLoader{}

// Load implements [Loader].
func (loader CacheLoader) Load(ctx context.Context, ref *url.URL) (*Schema, error) {
	urlString := bundleRefURLToID(ref)
	if schema := loader.schemas[urlString]; schema != nil {
		return schema, nil
	}
	schema, err := loader.subLoader.Load(ctx, ref)
	if err != nil {
		return nil, err
	}
	loader.schemas[urlString] = schema
	return schema, nil
}

type HTTPLoader struct {
	client *http.Client
}

func NewHTTPLoader(client *http.Client) HTTPLoader {
	return HTTPLoader{client: client}
}

var _ Loader = HTTPLoader{}

var yamlMediaTypeRegexp = regexp.MustCompile(`^application/(.*\+)?yaml$`)

// Load implements [Loader].
func (loader HTTPLoader) Load(ctx context.Context, ref *url.URL) (*Schema, error) {
	// Hardcoding a higher limit so CI/CD pipelines don't get stuck
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	refClone := *ref
	refClone.Fragment = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, refClone.String(), nil)
	if err != nil {
		return nil, err
	}
	// YAML now has a proper media type since Feb 2024 :D
	// https://datatracker.ietf.org/doc/rfc9512/
	req.Header.Add("Accept", "application/schema+json,application/json,application/schema+yaml,application/yaml,text/plain; charset=utf-8")
	req.Header.Add("Accept-Encoding", "gzip")
	if referrer, ok := ctx.Value(loaderContextReferrer).(string); ok {
		if strings.HasPrefix(referrer, "http://") || strings.HasPrefix(referrer, "https://") {
			req.Header.Add("Link", fmt.Sprintf(`<%s>; rel="describedby"`, referrer))
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "helm-values-schema-json/1")
	}

	start := time.Now()
	fmt.Println("Loading", req.URL.Redacted())

	resp, err := loader.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w", ref, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request $ref=%q over HTTP: got non-2xx status code: %s", ref, resp.Status)
	}
	defer closeIgnoreError(resp.Body)

	reader := resp.Body
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		r, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("request $ref=%q over HTTP: create gzip reader: %w", ref, err)
		}
		reader = r
	case "":
		// Do nothing
	default:
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w: unsupported content encoding: %q", ref, errors.ErrUnsupported, resp.Header.Get("Content-Encoding"))
	}

	var isYAML bool
	if mediatype, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil {
		switch strings.ToLower(params["charset"]) {
		case "", "utf-8", "utf8":
			// OK
		default:
			return nil, fmt.Errorf("request $ref=%q over HTTP: %w: unsupported response charset: %q", ref, errors.ErrUnsupported, params["charset"])
		}

		if yamlMediaTypeRegexp.MatchString(mediatype) {
			isYAML = true
		}
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w", ref, err)
	}

	duration := time.Since(start)
	fmt.Printf("=> got %s in %s\n", formatSizeBytes(len(b)), duration.Truncate(time.Millisecond))

	if isYAML {
		var schema Schema
		if err := yaml.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q YAML: %w", ref, err)
		}
		return &schema, nil
	} else {
		var schema Schema
		if err := json.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q JSON: %w", ref, err)
		}
		return &schema, nil
	}
}

type loaderContextKey int

var loaderContextReferrer = loaderContextKey(1)

func ContextWithLoaderReferrer(parent context.Context, referrer string) context.Context {
	return context.WithValue(parent, loaderContextReferrer, referrer)
}

func formatSizeBytes(size int) string {
	switch {
	case size < 2_000:
		return fmt.Sprintf("%dB", size)
	case size < 2_000_000:
		return fmt.Sprintf("%dKB", size/1_000)
	default:
		return fmt.Sprintf("%dMB", size/1_000_000)
	}
}
