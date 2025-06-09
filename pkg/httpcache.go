package pkg

import (
	"cmp"
	"compress/gzip"
	"encoding/base32"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
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
}

func NewHTTPCache() *HTTPFileCache {
	return &HTTPFileCache{
		cacheDirFunc: sync.OnceValue(func() string {
			dir, err := os.UserCacheDir()
			if err != nil {
				// Default to /tmp if there's no user $HOME
				dir = os.TempDir()
			}
			return filepath.Join(dir, "helm-values-schema-json", "httploader")
		}),
		now: time.Now,
	}
}

var _ HTTPCache = &HTTPFileCache{}

func (h *HTTPFileCache) LoadCache(req *http.Request) (CachedResponse, error) {
	path := filepath.Join(h.cacheDirFunc(), urlToCachePath(req.URL)+".gob.gz")

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
	gobDecoder := gob.NewDecoder(gzipReader)
	var cached CachedResponse
	if err := gobDecoder.Decode(&cached); err != nil {
		return CachedResponse{}, fmt.Errorf("decode cached response: %w", err)
	}
	return cached, nil
}

func (h *HTTPFileCache) SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error) {
	maxAge := getCacheControlMaxAge(resp.Header.Get("Cache-Control"))
	if maxAge <= 0 {
		// Response doesn't want to be cached.
		return CachedResponse{}, nil
	}

	cached := CachedResponse{
		Data:     body,
		CachedAt: h.now(),
		MaxAge:   maxAge,
		ETag:     resp.Header.Get("ETag"),
	}
	path := filepath.Join(h.cacheDirFunc(), urlToCachePath(req.URL)+".gob.gz")

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
	gobEncoder := gob.NewEncoder(gzipWriter)
	return cached, gobEncoder.Encode(cached)
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

// urlToCachePath returns a relative path that can be used as a file path
// when storing files in a cache, keyed by their URL.
//
// This function is lossy and one-way. It is not meant to be reversable.
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
	CachedAt time.Time
	MaxAge   time.Duration
	ETag     string
	Data     []byte
}

func (c CachedResponse) Expiry() time.Time {
	return c.CachedAt.Add(c.MaxAge)
}

func (c CachedResponse) Expired() bool {
	return time.Since(c.CachedAt) > c.MaxAge
}
