package pkg

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPCache_CacheDir(t *testing.T) {
	resetEnvAfterTest(t)
	os.Clearenv()
	os.Setenv("HOME", "/foo/bar")

	cache := NewHTTPCache()

	dir, err := cache.cacheDirFunc()
	require.NoError(t, err)
	assert.Equal(t, "/foo/bar/.cache/helm-values-schema-json/httploader", dir)
}

func TestHTTPCache_CacheDir_Error(t *testing.T) {
	resetEnvAfterTest(t)
	// Remove $HOME and $XDG_CONFIG_DIR, which makes [os.UserCacheDir] fail
	os.Clearenv()

	cache := NewHTTPCache()
	_, err := cache.cacheDirFunc()
	assert.Error(t, err)
}

func resetEnvAfterTest(t *testing.T) {
	envs := os.Environ()
	t.Cleanup(func() {
		for _, env := range envs {
			k, v, _ := strings.Cut(env, "=")
			os.Setenv(k, v)
		}
	})
}

func TestGetCacheControlMaxAge(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{
			name:   "empty",
			header: "",
			want:   0,
		},
		{
			name:   "only max-age",
			header: "max-age=123",
			want:   123 * time.Second,
		},
		{
			name:   "only no max-age value",
			header: "max-age",
			want:   0,
		},
		{
			name:   "only invalid max-age value",
			header: "max-age=foo",
			want:   0,
		},
		{
			name:   "valid max-age and max-age with no value",
			header: "max-age=123, max-age",
			want:   123 * time.Second,
		},
		{
			name:   "valid max-age and invalid max-age value",
			header: "max-age=123, max-age=foo",
			want:   123 * time.Second,
		},
		{
			name:   "max-age with other settings",
			header: "foo, bar, moo, max-age=123, doo, lorem",
			want:   123 * time.Second,
		},
		{
			name:   "max-age before no-cache",
			header: "max-age=123, no-cache",
			want:   0,
		},
		{
			name:   "max-age after no-cache",
			header: "no-cache, max-age=123",
			want:   0,
		},
		{
			name:   "no-cache with value",
			header: "no-cache=lorem, max-age=123",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCacheControlMaxAge(tt.header)
			if tt.want != got {
				t.Errorf("wrong result\nwant: %s\ngot:  %s", tt.want, got)
			}
		})
	}
}

func TestURLToCachePath(t *testing.T) {
	urlWithInvalidPath := mustParseURL("https://example.com/")
	// This is contrived on Linux, as Linux only invalidates null byte
	// But on Windows there are way more ways of making invalid paths
	urlWithInvalidPath.Path = "\x00"

	tests := []struct {
		name string
		url  *url.URL
		want string
	}{
		{
			name: "nil",
			url:  nil,
			want: "",
		},
		{
			name: "with invalid path characters",
			url:  urlWithInvalidPath,
			want: "https/example.com/AA",
		},
		{
			name: "no scheme",
			url:  mustParseURL("//example.com"),
			want: "no-scheme/example.com/_index",
		},
		{
			name: "port",
			url:  mustParseURL("https://example.com:80"),
			want: "https/example.com/80/_index",
		},
		{
			name: "example.com no path",
			url:  mustParseURL("https://example.com/"),
			want: "https/example.com/_index",
		},
		{
			name: "example.com with path",
			url:  mustParseURL("https://example.com/index.html"),
			want: "https/example.com/index.html",
		},
		{
			name: "remove userinfo",
			url:  mustParseURL("https://foo:bar@example.com/index.html"),
			want: "https/example.com/index.html",
		},
		{
			name: "remove fragment",
			url:  mustParseURL("https://example.com/index.html#foobar"),
			want: "https/example.com/index.html",
		},
		{
			name: "remove query",
			url:  mustParseURL("https://example.com/index.html?foo=bar"),
			want: "https/example.com/index.html",
		},
		{
			name: "multiple slashes",
			url:  mustParseURL("https://example.com//subdir///index.html"),
			want: "https/example.com/subdir/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlToCachePath(tt.url)
			if tt.want != got {
				t.Errorf("wrong result\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestCachedSchema_Expiry(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		time   time.Time
		maxAge time.Duration
		want   bool
	}{
		{
			name:   "zero time zero max age",
			time:   time.Time{},
			maxAge: 0,
			want:   true,
		},
		{
			name:   "too old",
			time:   now.Add(-1 * time.Hour),
			maxAge: 5 * time.Minute,
			want:   true,
		},
		{
			name:   "not expired",
			time:   now.Add(-1 * time.Minute),
			maxAge: 5 * time.Hour,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := CachedResponse{CachedAt: tt.time, MaxAge: tt.maxAge}
			assert.Equal(t, tt.want, cached.Expired())
			assert.WithinDuration(t, tt.time.Add(tt.maxAge), cached.Expiry(), time.Second)
		})
	}
}
