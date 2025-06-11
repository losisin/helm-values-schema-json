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
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
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
//
// The basePathForIDs is an absolute path used to change the resulting
// $ref & $id absolute paths of bundled local files to relative paths.
// It is only used cosmetically and has no impact of how files are loaded.
func Load(ctx context.Context, loader Loader, ref *url.URL, basePathForIDs string) (*Schema, error) {
	if loader == nil {
		return nil, fmt.Errorf("nil loader")
	}
	if ref == nil || *ref == (url.URL{}) {
		return nil, fmt.Errorf("cannot load empty $ref")
	}
	schema, err := loader.Load(ctx, ref)
	if err != nil || schema == nil {
		return nil, err
	}

	// It's fine to modify the $id here, as it is not used any more times
	// after this. So changing it is solely a cosmetic change.
	schema.ID = trimFragmentURL(refRelativeToBasePath(ref, basePathForIDs))
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
	fs         fs.FS
	fsRootPath string
}

// NewFileLoader returns a new file loader.
//
// The fsRootPath parameter is used to convert absolute paths into relative
// paths when loading the files, and should be the absolute path of the
// file system's root directory. This value mostly matters when using [os.Root].
//
// This is only a cosmetic change as it will make error messages have more
// readable relative paths instead of absolute paths.
func NewFileLoader(fs fs.FS, fsRootPath string) FileLoader {
	return FileLoader{
		fs:         fs,
		fsRootPath: fsRootPath,
	}
}

var _ Loader = FileLoader{}

// Load implements [Loader].
func (loader FileLoader) Load(ctx context.Context, ref *url.URL) (*Schema, error) {
	logger := LoggerFromContext(ctx)

	if ref.Scheme != "file" && ref.Scheme != "" {
		return nil, fmt.Errorf(`file url in $ref=%q must start with "file://", "./", or "/"`, ref)
	}
	refFile, err := ParseRefFileURL(ref)
	if err != nil {
		return nil, fmt.Errorf("parse file url: %w", err)
	}
	if refFile.Path == "" {
		return nil, fmt.Errorf("file url in $ref=%q must contain a path", ref)
	}
	pathAbs := filepath.FromSlash(refFile.Path)

	path := pathAbs
	if loader.fsRootPath != "" && filepath.IsAbs(pathAbs) {
		rel, err := filepath.Rel(loader.fsRootPath, path)
		if err != nil {
			return nil, fmt.Errorf("get relative path from bundle root: %w", err)
		}
		path = rel
	}

	logger.Log("Loading file", path)
	f, err := loader.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer closeIgnoreError(f)
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	logger.Logf("=> got %s", formatSizeBytes(len(b)))

	var schema Schema
	switch filepath.Ext(path) {
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse YAML file: %w", err)
		}
	default:
		if err := json.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse JSON file: %w", err)
		}
	}

	schema.SetReferrer(ReferrerDir(filepath.Dir(pathAbs)))
	return &schema, nil
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
	urlString := trimFragmentURL(ref)
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
	cache  HTTPCache

	SizeLimit int64
	UserAgent string
}

func NewHTTPLoader(client *http.Client, cache HTTPCache) HTTPLoader {
	return HTTPLoader{
		client:    client,
		cache:     cache,
		SizeLimit: 200 * 1000 * 1000, // arbitrary limit, but prevents CLI from eating all RAM
		UserAgent: HTTPLoaderDefaultUserAgent,
	}
}

var _ Loader = HTTPLoader{}

var HTTPLoaderDefaultUserAgent = ""

var yamlMediaTypeRegexp = regexp.MustCompile(`^application/(.*\+)?yaml$`)

// Flag is only used in testing to achieve better test coverage
var failHTTPLoaderNewRequest bool

// Load implements [Loader].
func (loader HTTPLoader) Load(ctx context.Context, ref *url.URL) (*Schema, error) {
	logger := LoggerFromContext(ctx)
	// Hardcoding a higher limit so CI/CD pipelines don't get stuck
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	refClone := *ref
	refClone.Fragment = ""

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, refClone.String(), nil)
	if err != nil || failHTTPLoaderNewRequest {
		// The [http.NewRequestWithContext] will never fail,
		// so we have to induce a fake failure via [failHTTPLoaderNewRequest]
		return nil, fmt.Errorf("create request: %w", err)
	}

	logger.Log("Loading", req.URL.Redacted())

	cached, cachedSchema, err := loader.LoadCache(req)
	if err != nil {
		logger.Log("Error loading from cache:", err)
	} else if cachedSchema != nil {
		duration := time.Since(start)
		logger.Logf("=> got %s from cache in %s (expires in %s)",
			formatSizeBytes(len(cached.Data)),
			duration.Truncate(time.Millisecond),
			time.Until(cached.Expiry()).Truncate(time.Second))
		return cachedSchema, nil
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
	if loader.UserAgent != "" {
		req.Header.Set("User-Agent", loader.UserAgent)
	}

	if cached.ETag != "" {
		req.Header.Add("If-None-Match", cached.ETag)
	}

	resp, err := loader.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request $ref over HTTP: %w", err)
	}
	defer closeIgnoreError(resp.Body)

	if cached.ETag != "" && resp.StatusCode == http.StatusNotModified {
		cached, schema, err := loader.SaveCacheETag(req, resp, cached)
		if err == nil {
			duration := time.Since(start)
			logger.Logf("=> renewed cache of %s in %s (expires in %s)",
				formatSizeBytes(len(cached.Data)),
				duration.Truncate(time.Millisecond),
				cached.MaxAge.Truncate(time.Second))
			return schema, nil
		}
		logger.Log("Error using etag cache:", err)
		// Redo the request, but without the etag this time
		req.Header.Del("If-None-Match")
		newResp, err := loader.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request $ref over HTTP: %w", err)
		}
		resp = newResp
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request $ref=%q over HTTP: got non-2xx status code: %s", ref.Redacted(), resp.Status)
	}

	var reader io.Reader = resp.Body
	if loader.SizeLimit > 0 {
		reader = LimitReaderWithError(reader, loader.SizeLimit,
			fmt.Errorf("aborted request after reading more than %s", formatSizeBytes(int(loader.SizeLimit))))
	}
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
	if mediaType, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil {
		switch strings.ToLower(params["charset"]) {
		case "", "utf-8", "utf8":
			// OK
		default:
			return nil, fmt.Errorf("request $ref=%q over HTTP: %w: unsupported response charset: %q", ref.Redacted(), errors.ErrUnsupported, params["charset"])
		}

		if yamlMediaTypeRegexp.MatchString(mediaType) {
			isYAML = true
		}
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("request $ref=%q over HTTP: %w", ref.Redacted(), err)
	}

	cached, err = loader.SaveCache(req, resp, b)
	if err != nil {
		logger.Log("Error saving response cache:", err)
	}

	duration := time.Since(start)
	if cached.MaxAge > 0 {
		logger.Logf("=> cached %s in %s (expires in %s)",
			formatSizeBytes(len(b)),
			duration.Truncate(time.Millisecond),
			cached.MaxAge.Truncate(time.Second))
	} else {
		logger.Logf("=> got %s in %s",
			formatSizeBytes(len(b)),
			duration.Truncate(time.Millisecond))
	}

	var schema Schema
	if isYAML {
		if err := yaml.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q YAML: %w", ref.Redacted(), err)
		}
	} else {
		if err := json.Unmarshal(b, &schema); err != nil {
			return nil, fmt.Errorf("parse $ref=%q JSON: %w", ref.Redacted(), err)
		}
	}

	refClone.Path = path.Dir(refClone.Path)
	if ref.Path == "" {
		// path.Dir turns empty path into "." and we don't want that
		refClone.Path = ""
	}
	schema.SetReferrer(ReferrerURL(&refClone))
	return &schema, nil
}

func (loader HTTPLoader) SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error) {
	if loader.cache == nil {
		return CachedResponse{}, nil
	}
	cached, err := loader.cache.SaveCache(req, resp, body)
	if err != nil {
		return CachedResponse{}, err
	}
	return cached, nil
}

func (loader HTTPLoader) SaveCacheETag(req *http.Request, resp *http.Response, cached CachedResponse) (CachedResponse, *Schema, error) {
	if loader.cache == nil {
		return CachedResponse{}, nil, nil
	}
	renewedCache, err := loader.cache.SaveCache(req, resp, cached.Data)
	if err != nil {
		return CachedResponse{}, nil, err
	}
	var schema Schema
	if err := yaml.Unmarshal(renewedCache.Data, &schema); err != nil {
		return CachedResponse{}, nil, fmt.Errorf("parse cached YAML: %w", err)
	}
	return renewedCache, &schema, nil
}

func (loader HTTPLoader) LoadCache(req *http.Request) (CachedResponse, *Schema, error) {
	if loader.cache == nil {
		return CachedResponse{}, nil, nil
	}
	cached, err := loader.cache.LoadCache(req)
	if err != nil {
		if os.IsNotExist(err) {
			return CachedResponse{}, nil, nil
		}
		return CachedResponse{}, nil, err
	}
	if cached.Expired() {
		return cached, nil, nil
	}
	var schema Schema
	if err := yaml.Unmarshal(cached.Data, &schema); err != nil {
		return CachedResponse{}, nil, fmt.Errorf("parse cached YAML: %w", err)
	}
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
