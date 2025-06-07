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

type HTTPCache struct {
	cacheDirFunc func() (string, error)
}

func NewHTTPCache() *HTTPCache {
	return &HTTPCache{
		cacheDirFunc: sync.OnceValues(func() (string, error) {
			dir, err := os.UserCacheDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(dir, "helm-values-schema-json", "httploader"), nil
		}),
	}
}

func (h *HTTPCache) LoadCache(req *http.Request) (CachedResponse, error) {
	path := urlToCachePath(req.URL) + ".gob.gz"
	dir, err := h.cacheDirFunc()
	if err != nil {
		return CachedResponse{}, err
	}
	path = filepath.Join(dir, path)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CachedResponse{}, nil
		}
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
		return CachedResponse{}, err
	}
	return cached, nil
}

func (h *HTTPCache) SaveCache(req *http.Request, resp *http.Response, body []byte) (CachedResponse, error) {
	maxAge := getCacheControlMaxAge(resp.Header.Get("Cache-Control"))
	if maxAge <= 0 {
		// Response doesn't want to be cached.
		return CachedResponse{}, nil
	}
	cached := CachedResponse{
		Data:     body,
		CachedAt: time.Now(),
		MaxAge:   maxAge,
		ETag:     resp.Header.Get("ETag"),
	}

	path := urlToCachePath(req.URL) + ".gob.gz"
	dir, err := h.cacheDirFunc()
	if err != nil {
		return CachedResponse{}, err
	}
	path = filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return CachedResponse{}, fmt.Errorf("mkdir: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return CachedResponse{}, err
	}
	gzipWriter := gzip.NewWriter(file)
	defer closeIgnoreError(gzipWriter)
	gobEncoder := gob.NewEncoder(gzipWriter)
	if err := gobEncoder.Encode(cached); err != nil {
		return CachedResponse{}, err
	}
	return cached, nil
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
				return 0
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
	urlPath := strings.TrimPrefix(filepath.Clean(u.Path), "/")
	if urlPath == "." {
		urlPath = ""
	}
	if urlPath != "" {
		pathSegments := filepath.SplitList(urlPath)
		for i, seg := range pathSegments {
			if _, err := filepath.Localize(seg); err != nil {
				// convert any invalid path segments into base32
				pathSegments[i] = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(seg))
			}
		}
		urlPath = filepath.Join(pathSegments...)
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
