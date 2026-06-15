package pkg

import (
	"cmp"
	"compress/gzip"
	"encoding/base32"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
)

type HTTPCache interface {
	LoadCache(req *http.Request) (CachedResponse, error)
	SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error)
}

type DummyHTTPCache struct {
	LoadCacheFunc func(req *http.Request) (CachedResponse, error)
	SaveCacheFunc func(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error)
}

var _ HTTPCache = DummyHTTPCache{}

func (d DummyHTTPCache) LoadCache(req *http.Request) (CachedResponse, error) {
	return d.LoadCacheFunc(req)
}

func (d DummyHTTPCache) SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error) {
	return d.SaveCacheFunc(req, resp, body)
}

func NewHTTPMemoryCache() *HTTPMemoryCache {
	return &HTTPMemoryCache{
		Map: map[string]CachedResponse{},
		Now: time.Now,
	}
}

type HTTPMemoryCache struct {
	Map map[string]CachedResponse
	Now func() time.Time

	// MinCacheDuration mirrors [HTTPFileCache.MinCacheDuration].
	MinCacheDuration time.Duration
}

var _ HTTPCache = &HTTPMemoryCache{}

func (h *HTTPMemoryCache) LoadCache(req *http.Request) (CachedResponse, error) {
	if cached, ok := h.Map[req.URL.String()]; ok {
		return cached, nil
	}
	return CachedResponse{}, os.ErrNotExist
}

func (h *HTTPMemoryCache) SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error) {
	maxAge := getCacheControlMaxAge(resp.Header.Get("Cache-Control"))
	if maxAge <= 0 {
		// Response doesn't want to be cached.
		return CachedResponse{}, nil
	}
	maxAge = applyMinCacheDuration(maxAge, h.MinCacheDuration)
	cached := CachedResponse{
		Data:     body,
		CachedAt: h.Now(),
		MaxAge:   maxAge,
		ETag:     resp.Header.Get("ETag"),
	}
	h.Map[req.URL.String()] = cached
	return cached, nil
}

type HTTPFileCache struct {
	cacheDirFunc func() string
	now          func() time.Time

	// MinCacheDuration, when greater than zero, raises a cacheable response's
	// effective max-age to at least this value. It lets users keep downloaded
	// schemas cached longer than the short max-age that many schema stores
	// return. Responses the server marks as uncacheable are still not cached.
	MinCacheDuration time.Duration
}

func NewHTTPCache(minCacheDuration time.Duration) *HTTPFileCache {
	return &HTTPFileCache{
		cacheDirFunc: sync.OnceValue(func() string {
			dir, err := os.UserCacheDir()
			if err != nil {
				// Default to /tmp if there's no user $HOME
				dir = os.TempDir()
			}
			return filepath.Join(dir, "helm-values-schema-json", "httploader")
		}),
		now:              time.Now,
		MinCacheDuration: minCacheDuration,
	}
}

var _ HTTPCache = &HTTPFileCache{}

func (h *HTTPFileCache) LoadCache(req *http.Request) (CachedResponse, error) {
	path := filepath.Join(h.cacheDirFunc(), urlToCachePath(req.URL)+".cbor.gz")

	file, err := os.Open(path) // #nosec G304 -- path is known to be safe thanks to [urlToCachePath]
	if err != nil {
		return CachedResponse{}, err
	}
	defer closeIgnoreError(file)
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return CachedResponse{}, err
	}
	defer closeIgnoreError(gzipReader)
	cborDecoder := cbor.NewDecoder(gzipReader)
	var cached CachedResponse
	if err := cborDecoder.Decode(&cached); err != nil {
		return CachedResponse{}, fmt.Errorf("decode cached response: %w", err)
	}
	cached.CachedAt = cached.CachedAt.UTC()
	return cached, nil
}

func (h *HTTPFileCache) SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error) {
	maxAge := getCacheControlMaxAge(resp.Header.Get("Cache-Control"))
	if maxAge <= 0 {
		// Response doesn't want to be cached.
		return CachedResponse{}, nil
	}

	maxAge = applyMinCacheDuration(maxAge, h.MinCacheDuration)
	cached := CachedResponse{
		Data:     body,
		CachedAt: h.now().UTC(),
		MaxAge:   maxAge,
		ETag:     resp.Header.Get("ETag"),
	}
	path := filepath.Join(h.cacheDirFunc(), urlToCachePath(req.URL)+".cbor.gz")

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return CachedResponse{}, fmt.Errorf("mkdir: %w", err)
	}

	file, err := os.Create(path) // #nosec G304 -- path is known to be safe thanks to [urlToCachePath]
	if err != nil {
		return CachedResponse{}, fmt.Errorf("create cache file: %w", err)
	}
	defer closeIgnoreError(file)
	gzipWriter := gzip.NewWriter(file)
	defer closeIgnoreError(gzipWriter)
	cborEncoder := cbor.NewEncoder(gzipWriter)
	return cached, cborEncoder.Encode(cached)
}

func getCacheControlMaxAge(header string) time.Duration {
	var maxAge time.Duration
	for directive := range strings.SplitSeq(header, ",") {
		key, value, _ := strings.Cut(strings.TrimSpace(directive), "=")
		switch key {
		case "no-cache", "no-store":
			return 0
		case "max-age":
			seconds, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			maxAge = time.Duration(seconds) * time.Second
		}
	}
	return maxAge
}

// applyMinCacheDuration raises maxAge to minDuration when minDuration is larger.
// It is only meant to be called for responses that are already cacheable
// (maxAge > 0); responses the server marked as no-store/no-cache are skipped by
// the caller and never extended.
func applyMinCacheDuration(maxAge, minDuration time.Duration) time.Duration {
	if minDuration > maxAge {
		return minDuration
	}
	return maxAge
}

// ParseCacheMinDuration parses a --bundle-cache-min value such as "24h" or
// "30m" into a [time.Duration]. An empty string means "no override" and returns
// zero. It is the single place that interprets the duration string, so any
// future support for extended units (1d/1w/1M/1y) only needs to be added here.
func ParseCacheMinDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("parse bundle cache min duration %q: %w", s, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("bundle cache min duration %q must not be negative", s)
	}
	return d, nil
}

// urlToCachePath returns a relative path that can be used as a file path
// when storing files in a cache, keyed by their URL.
//
// This function is lossy and one-way. It is not meant to be reversible.
// It has potential for collisions, but not during normal usage.
//
// The purpose is to make the file path human readable to allow the user
// to manually clearing part of the cache by deleting correct files
// from their disk.
func urlToCachePath(u *url.URL) string {
	if u == nil {
		return ""
	}
	segments := []string{}
	segments = append(segments, cmp.Or(u.Scheme, "no-scheme"))
	segments = append(segments, cmp.Or(u.Hostname(), "no-host"))
	if port := u.Port(); port != "" {
		segments = append(segments, port)
	}
	urlPath := strings.TrimPrefix(u.Path, "/")
	if urlPath != "" {
		pathSegments := strings.Split(urlPath, "/")
		for i, seg := range pathSegments {
			switch seg {
			case ".":
				pathSegments[i] = "_dot"
			case "..":
				pathSegments[i] = "_up"
			default:
				if _, err := filepath.Localize(seg); err != nil {
					// convert any invalid path segments into base32
					pathSegments[i] = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(seg))
				}
			}
		}
		urlPath = filepath.Clean(filepath.Join(pathSegments...))
	}
	segments = append(segments, cmp.Or(urlPath, "_index"))
	return filepath.Join(segments...)
}

type CachedResponse struct {
	CachedAt time.Time     `cbor:"cachedAt"`
	MaxAge   time.Duration `cbor:"maxAge"`
	ETag     string        `cbor:"etag"`
	Data     []byte        `cbor:"data"`
}

func (c CachedResponse) Expiry() time.Time {
	return c.CachedAt.Add(c.MaxAge)
}

func (c CachedResponse) Expired() bool {
	return time.Since(c.CachedAt) > c.MaxAge
}
