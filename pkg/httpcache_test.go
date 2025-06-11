package pkg

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/losisin/helm-values-schema-json/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPCache_CacheDir(t *testing.T) {
	testutil.ResetEnvAfterTest(t)
	os.Clearenv()
	require.NoError(t, os.Setenv("HOME", "/foo/bar"))

	cache := NewHTTPCache()

	dir := cache.cacheDirFunc()
	require.Contains(t, dir, "/helm-values-schema-json/httploader")
}

func TestHTTPCache_CacheDir_Error(t *testing.T) {
	testutil.ResetEnvAfterTest(t)
	// Remove $HOME and $XDG_CONFIG_DIR, which makes [os.UserCacheDir] fail
	os.Clearenv()

	cache := NewHTTPCache()
	dir := cache.cacheDirFunc()
	assert.Equal(t, "/tmp/helm-values-schema-json/httploader", dir)
}

func TestLoadCache(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		setup     func(t *testing.T, dir string)
		want      CachedResponse
		wantErr   string
		wantErrIs error
	}{
		{
			name:      "empty url",
			url:       "",
			setup:     func(t *testing.T, dir string) {},
			wantErrIs: os.ErrNotExist,
		},
		{
			name:      "file not found",
			url:       "http://example.com/file.txt",
			setup:     func(t *testing.T, dir string) {},
			wantErrIs: os.ErrNotExist,
		},
		{
			name: "dir",
			url:  "http://example.com/file.txt",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "http", "example.com", "file.txt.cbor.gz"), 0755))
			},
			wantErr: "file.txt.cbor.gz: is a directory",
		},
		{
			name: "invalid gzip",
			url:  "http://example.com/file.txt",
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "http", "example.com")
				require.NoError(t, os.MkdirAll(subdir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.txt.cbor.gz"), []byte("this is invalid gzip"), 0644))
			},
			wantErr: "gzip: invalid header",
		},
		{
			name: "invalid cbor",
			url:  "http://example.com/file.txt",
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "http", "example.com")

				require.NoError(t, os.MkdirAll(subdir, 0755))
				file, err := os.Create(filepath.Join(subdir, "file.txt.cbor.gz"))
				require.NoError(t, err)
				defer func() { assert.NoError(t, file.Close()) }()

				w := gzip.NewWriter(file)
				defer func() { assert.NoError(t, w.Close()) }()

				_, err = io.WriteString(w, "this is invalid cbor")
				require.NoError(t, err)
			},
			wantErr: "decode cached response: ",
		},
		{
			name: "valid",
			url:  "http://example.com/file.txt",
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "http", "example.com")

				require.NoError(t, os.MkdirAll(subdir, 0755))
				file, err := os.Create(filepath.Join(subdir, "file.txt.cbor.gz"))
				require.NoError(t, err)
				defer func() { assert.NoError(t, file.Close()) }()

				w := gzip.NewWriter(file)
				defer func() { assert.NoError(t, w.Close()) }()

				// Copy the type here, so that we can detect regressions in case the real struct is changed
				type CachedResponse struct {
					CachedAt time.Time
					MaxAge   time.Duration
					ETag     string
					Data     []byte
				}

				enc := cbor.NewEncoder(w)
				require.NoError(t, enc.Encode(CachedResponse{
					CachedAt: time.Date(2025, 6, 8, 12, 0, 0, 0, time.UTC),
					MaxAge:   time.Hour,
					ETag:     "lorem ipsum",
					Data:     []byte(`{"foo":"bar"}`),
				}))
			},
			want: CachedResponse{
				CachedAt: time.Date(2025, 6, 8, 12, 0, 0, 0, time.UTC),
				MaxAge:   time.Hour,
				ETag:     "lorem ipsum",
				Data:     []byte(`{"foo":"bar"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewHTTPCache()
			dir, err := os.MkdirTemp("", "schema-httpcache-*")
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, os.RemoveAll(dir))
			})
			cache.cacheDirFunc = func() string { return dir }

			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)

			tt.setup(t, dir)

			cached, err := cache.LoadCache(req)
			switch {
			case tt.wantErrIs != nil:
				require.ErrorIs(t, err, tt.wantErrIs)
			case tt.wantErr != "":
				assert.ErrorContains(t, err, tt.wantErr)
			default:
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, cached)
		})
	}
}

func TestSaveCache(t *testing.T) {
	now := time.Date(2025, 6, 9, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		url  string
		resp *http.Response
		body []byte
		want CachedResponse
	}{
		{
			name: "no cache control",
			url:  "http://example.com",
			resp: &http.Response{
				Header: http.Header{
					http.CanonicalHeaderKey("ETag"): []string{"myETag"},
				},
			},
			body: []byte("foo"),
			want: CachedResponse{},
		},
		{
			name: "no max age in cache control",
			url:  "http://example.com",
			resp: &http.Response{
				Header: http.Header{
					http.CanonicalHeaderKey("Cache-Control"): []string{"foobar"},
					http.CanonicalHeaderKey("ETag"):          []string{"myETag"},
				},
			},
			body: []byte("foo"),
			want: CachedResponse{},
		},
		{
			name: "no etag",
			url:  "http://example.com",
			resp: &http.Response{
				Header: http.Header{
					http.CanonicalHeaderKey("Cache-Control"): []string{"max-age=100"},
				},
			},
			body: []byte("foo"),
			want: CachedResponse{
				CachedAt: now,
				MaxAge:   100 * time.Second,
				Data:     []byte("foo"),
			},
		},
		{
			name: "full save",
			url:  "http://example.com",
			resp: &http.Response{
				Header: http.Header{
					http.CanonicalHeaderKey("Cache-Control"): []string{"max-age=100"},
					http.CanonicalHeaderKey("ETag"):          []string{"myETag"},
				},
			},
			body: []byte("foo"),
			want: CachedResponse{
				CachedAt: now,
				MaxAge:   100 * time.Second,
				Data:     []byte("foo"),
				ETag:     "myETag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewHTTPCache()
			dir := testutil.CreateTempDir(t, "schema-httpcache-*")
			cache.cacheDirFunc = func() string { return dir }
			cache.now = func() time.Time { return now }

			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)

			cached, err := cache.SaveCache(req, tt.resp, tt.body)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cached)

			loaded, err := cache.LoadCache(req)
			if !os.IsNotExist(err) {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, loaded)
		})
	}
}

func TestSaveCache_Error(t *testing.T) {
	t.Run("mkdir", func(t *testing.T) {
		cache := NewHTTPCache()
		file := testutil.CreateTempFile(t, "schema-httpcache-*")
		cache.cacheDirFunc = func() string { return file.Name() }

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		require.NoError(t, err)

		_, err = cache.SaveCache(req, &http.Response{
			Header: http.Header{
				http.CanonicalHeaderKey("Cache-Control"): []string{"max-age=100"},
				http.CanonicalHeaderKey("ETag"):          []string{"myETag"},
			},
		}, nil)
		assert.ErrorContains(t, err, "mkdir:")
	})

	t.Run("create file", func(t *testing.T) {
		cache := NewHTTPCache()
		dir := testutil.CreateTempDir(t, "schema-httpcache-*")
		cache.cacheDirFunc = func() string { return dir }

		require.NoError(t, os.MkdirAll(filepath.Join(dir, "http", "example.com", "schema.json.cbor.gz"), 0755))

		req, err := http.NewRequest(http.MethodGet, "http://example.com/schema.json", nil)
		require.NoError(t, err)

		_, err = cache.SaveCache(req, &http.Response{
			Header: http.Header{
				http.CanonicalHeaderKey("Cache-Control"): []string{"max-age=100"},
				http.CanonicalHeaderKey("ETag"):          []string{"myETag"},
			},
		}, nil)
		assert.ErrorContains(t, err, "create cache file:")
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
		name      string
		url       *url.URL
		wantParts []string
	}{
		{
			name:      "nil",
			url:       nil,
			wantParts: nil,
		},
		{
			name:      "with invalid path characters",
			url:       urlWithInvalidPath,
			wantParts: []string{"https", "example.com", "AA"},
		},
		{
			name:      "no scheme",
			url:       mustParseURL("//example.com"),
			wantParts: []string{"no-scheme", "example.com", "_index"},
		},
		{
			name:      "port",
			url:       mustParseURL("https://example.com:80"),
			wantParts: []string{"https", "example.com", "80", "_index"},
		},
		{
			name:      "example.com no path",
			url:       mustParseURL("https://example.com/"),
			wantParts: []string{"https", "example.com", "_index"},
		},
		{
			name:      "example.com with path",
			url:       mustParseURL("https://example.com/index.html"),
			wantParts: []string{"https", "example.com", "index.html"},
		},
		{
			name:      "remove userinfo",
			url:       mustParseURL("https://foo:bar@example.com/index.html"),
			wantParts: []string{"https", "example.com", "index.html"},
		},
		{
			name:      "remove fragment",
			url:       mustParseURL("https://example.com/index.html#foobar"),
			wantParts: []string{"https", "example.com", "index.html"},
		},
		{
			name:      "remove query",
			url:       mustParseURL("https://example.com/index.html?foo=bar"),
			wantParts: []string{"https", "example.com", "index.html"},
		},
		{
			name:      "multiple slashes",
			url:       mustParseURL("https://example.com//subdir///index.html"),
			wantParts: []string{"https", "example.com", "subdir", "index.html"},
		},
		{
			name:      "dots",
			url:       mustParseURL("https://example.com/."),
			wantParts: []string{"https", "example.com", "_dot"},
		},
		{
			name:      "folder escape",
			url:       mustParseURL("https://example.com/../../foo/../../index.html"),
			wantParts: []string{"https", "example.com", "_up", "_up", "foo", "_up", "_up", "index.html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlToCachePath(tt.url)
			var want string
			if tt.wantParts != nil {
				want = filepath.Join(tt.wantParts...)
			}

			if want != got {
				t.Errorf("wrong result\nurl:  %q\nwant: %q\ngot:  %q", tt.url, want, got)
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
