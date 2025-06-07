package pkg

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
