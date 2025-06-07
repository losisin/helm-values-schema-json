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
	"io/fs"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RootFS is a replacement for [os.Root.FS] that intentionally doesn't call
// [fs.ValidPath] as that messes up the error messages and doesn't add any
// security guarantees as all the security is implemented in [os.Root] already.
type RootFS os.Root

var _ fs.FS = &RootFS{}

// Open implements [fs.FS].
func (r *RootFS) Open(name string) (fs.File, error) {
	return ((*os.Root)(r)).Open(name)
}

func NewDefaultLoader(client *http.Client, bundleFS fs.FS, basePath string) Loader {
	fileLoader := NewFileLoader(bundleFS, basePath)
	httpLoader := NewHTTPLoader(client, NewHTTPCache())
	return NewCacheLoader(URLSchemeLoader{
		"http":  httpLoader,
		"https": httpLoader,
		"file":  fileLoader, // Used for "file:///some/abs/path"
		"":      fileLoader, // Used for "./foobar.json" or "/some/abs/path"
	})
}

// Load uses a bundle [Loader] to resolve a schema "$ref".
// Depending on the loader implementation, it may read from cache,
// read files from disk, or fetch files from the web using HTTP.
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
	fs       fs.FS
	basePath string
}

func NewFileLoader(fs fs.FS, basePath string) FileLoader {
	return FileLoader{
		fs:       fs,
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
	f, err := loader.fs.Open(path)
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
			errors.ErrUnsupported, ref.Redacted(), strings.Join(slices.Collect(maps.Keys(loader)), ","))
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
	cache  *HTTPCache
}

func NewHTTPLoader(client *http.Client, cache *HTTPCache) HTTPLoader {
	return HTTPLoader{client: client, cache: cache}
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

	fmt.Println("Loading", req.URL.Redacted())

	cached, schema, err := loader.LoadCache(req)
	if err != nil {
		fmt.Println("Error loading from cache:", err)
	} else if schema != nil {
		return schema, nil
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

	if cached.ETag != "" {
		req.Header.Add("If-None-Match", cached.ETag)
	}

	start := time.Now()
	resp, err := loader.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w", ref.Redacted(), err)
	}

	if cached.ETag != "" && resp.StatusCode == http.StatusNotModified {
		schema, err := loader.LoadCacheETag(req, resp, cached)
		if err == nil {
			return schema, nil
		}
		fmt.Println("Error using etag cache:", err)
		// Redo the request, but without the etag this time
		newReq, err := http.NewRequestWithContext(ctx, http.MethodGet, refClone.String(), nil)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header.Clone()
		newReq.Header.Del("If-None-Match")
		newResp, err := loader.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request $ref=%q over HTTP: %w", ref.Redacted(), err)
		}
		req = newReq
		resp = newResp
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request $ref=%q over HTTP: got non-2xx status code: %s", ref.Redacted(), resp.Status)
	}
	defer closeIgnoreError(resp.Body)

	reader := resp.Body
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		r, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("request $ref=%q over HTTP: create gzip reader: %w", ref.Redacted(), err)
		}
		reader = r
	case "":
		// Do nothing
	default:
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w: unsupported content encoding: %q", ref.Redacted(), errors.ErrUnsupported, resp.Header.Get("Content-Encoding"))
	}

	var isYAML bool
	if mediatype, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil {
		switch strings.ToLower(params["charset"]) {
		case "", "utf-8", "utf8":
			// OK
		default:
			return nil, fmt.Errorf("request $ref=%q over HTTP: %w: unsupported response charset: %q", ref.Redacted(), errors.ErrUnsupported, params["charset"])
		}

		if yamlMediaTypeRegexp.MatchString(mediatype) {
			isYAML = true
		}
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w", ref.Redacted(), err)
	}

	if err := loader.SaveCache(req, resp, b); err != nil {
		fmt.Println("Error saving response cache:", err)
	}

	duration := time.Since(start)
	fmt.Printf("=> got %s in %s\n", formatSizeBytes(len(b)), duration.Truncate(time.Millisecond))

	if isYAML {
		var schema Schema
		if err := yaml.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q YAML: %w", ref.Redacted(), err)
		}
		return &schema, nil
	} else {
		var schema Schema
		if err := json.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q JSON: %w", ref.Redacted(), err)
		}
		return &schema, nil
	}
}

func (loader HTTPLoader) SaveCache(req *http.Request, resp *http.Response, body []byte) error {
	if loader.cache == nil {
		return nil
	}
	cached, err := loader.cache.SaveCache(req, resp, body)
	if err != nil {
		return err
	}
	fmt.Printf("=> cached (expires in %s)\n",
		cached.MaxAge.Truncate(time.Second))
	return nil
}

func (loader HTTPLoader) LoadCacheETag(req *http.Request, resp *http.Response, cached CachedResponse) (*Schema, error) {
	if loader.cache == nil {
		return nil, nil
	}
	renewedCache, err := loader.cache.SaveCache(req, resp, cached.Data)
	if err != nil {
		return nil, err
	}
	var schema Schema
	if err := yaml.Unmarshal(renewedCache.Data, &schema); err != nil {
		return nil, fmt.Errorf("parse cached YAML: %w", err)
	}
	fmt.Printf("=> got %s from cache (renewed etag, expires in %s)\n",
		formatSizeBytes(len(renewedCache.Data)),
		renewedCache.MaxAge.Truncate(time.Second))
	return &schema, nil
}

func (loader HTTPLoader) LoadCache(req *http.Request) (CachedResponse, *Schema, error) {
	if loader.cache == nil {
		return CachedResponse{}, nil, nil
	}
	cached, err := loader.cache.LoadCache(req)
	if err != nil {
		return CachedResponse{}, nil, err
	}
	if cached.Expired() {
		return cached, nil, nil
	}
	var schema Schema
	if err := yaml.Unmarshal(cached.Data, &schema); err != nil {
		return CachedResponse{}, nil, fmt.Errorf("parse cached YAML: %w", err)
	}
	fmt.Printf("=> got %s from cache (expires in %s)\n",
		formatSizeBytes(len(cached.Data)),
		time.Until(cached.Expiry()).Truncate(time.Second))
	return cached, &schema, nil
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
