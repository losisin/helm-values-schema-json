package pkg

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		loader  Loader
		ref     *url.URL
		wantErr string
	}{
		{
			name:    "nil loader",
			loader:  nil,
			ref:     mustParseURL(""),
			wantErr: "nil loader",
		},
		{
			name:    "nil ref",
			loader:  DummyLoader{},
			ref:     nil,
			wantErr: "cannot load empty $ref",
		},
		{
			name:    "empty ref",
			loader:  DummyLoader{},
			ref:     mustParseURL(""),
			wantErr: "cannot load empty $ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(t.Context(), tt.loader, tt.ref, "/")
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestFileLoader_Error(t *testing.T) {
	tests := []struct {
		name       string
		url        *url.URL
		fsRootPath string
		wantErr    string
	}{
		{
			name:    "invalid scheme",
			url:     mustParseURL("ftp:///foo"),
			wantErr: `file url in $ref="ftp:///foo" must start with "file://", "./", or "/"`,
		},
		{
			name:    "invalid path",
			url:     mustParseURL("file://localhost"),
			wantErr: `file url in $ref="file://localhost" must contain a path`,
		},
		{
			name:    "path escapes parent",
			url:     mustParseURL("file:///file/that/does/not/exist"),
			wantErr: "path escapes from parent",
		},
		{
			name:    "path not found",
			url:     mustParseURL("./local/file/that/does/not/exist"),
			wantErr: "no such file or directory",
		},
		{
			name:    "invalid JSON",
			url:     mustParseURL("./invalid-schema.json"),
			wantErr: `parse JSON file: invalid character 'h' in literal true`,
		},
		{
			name:    "invalid YAML",
			url:     mustParseURL("./invalid-schema.yaml"),
			wantErr: `parse YAML file: yaml: did not find expected key`,
		},
		{
			name:       "fail to get relative path from fsRootPath",
			url:        mustParseURL("/foo/bar"),
			fsRootPath: "some/relative/path",
			wantErr:    `get relative path from bundle root: Rel: can't make /foo/bar relative to some/relative/path`,
		},
	}

	root, err := os.OpenRoot("../testdata/bundle")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, root.Close())
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewFileLoader((*RootFS)(root), tt.fsRootPath)
			_, err := loader.Load(t.Context(), tt.url)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

type DummyFS struct {
	OpenFunc func(name string) (fs.File, error)
}

var _ fs.FS = DummyFS{}

// Open implements [fs.FS].
func (d DummyFS) Open(name string) (fs.File, error) {
	return d.OpenFunc(name)
}

type DummyFile struct {
	StatFunc  func() (fs.FileInfo, error)
	ReadFunc  func([]byte) (int, error)
	CloseFunc func() error
}

var _ fs.File = DummyFile{}

// Close implements [fs.File].
func (d DummyFile) Close() error {
	return d.CloseFunc()
}

// Read implements [fs.File].
func (d DummyFile) Read(b []byte) (int, error) {
	return d.ReadFunc(b)
}

// Stat implements [fs.File].
func (d DummyFile) Stat() (fs.FileInfo, error) {
	return d.StatFunc()
}

func TestFileLoader_TestFSError(t *testing.T) {
	root := DummyFS{
		OpenFunc: func(name string) (fs.File, error) {
			return DummyFile{
				CloseFunc: func() error { return nil },
				ReadFunc: func([]byte) (int, error) {
					return 0, fmt.Errorf("dummy error")
				},
			}, nil
		},
	}

	loader := NewFileLoader(root, "")
	_, err := loader.Load(t.Context(), mustParseURL("./some-fake-file.txt"))
	assert.ErrorContains(t, err, "dummy error")
}

func TestURLSchemeLoader_Error(t *testing.T) {
	tests := []struct {
		name    string
		url     *url.URL
		loader  URLSchemeLoader
		wantErr string
	}{
		{
			name: "invalid scheme",
			url:  mustParseURL("bar:///foo"),
			loader: URLSchemeLoader{
				"foo": DummyLoader{},
			},
			wantErr: `unsupported operation: cannot load schema from $ref="bar:///foo", supported schemes: foo`,
		},
		{
			name: "loader returns error",
			url:  mustParseURL("foo://"),
			loader: URLSchemeLoader{
				"foo": DummyLoader{
					LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
						return nil, fmt.Errorf("test error")
					},
				},
			},
			wantErr: `test error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.loader.Load(t.Context(), tt.url)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCacheLoader(t *testing.T) {
	counter := 0
	loader := NewCacheLoader(DummyLoader{
		LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
			counter++
			return &Schema{
				Enum: []any{counter},
			}, nil
		},
	})

	schema1, err := loader.Load(t.Context(), mustParseURL("foo://"))
	require.NoError(t, err)
	schema2, err := loader.Load(t.Context(), mustParseURL("foo://"))
	require.NoError(t, err)
	schema3, err := loader.Load(t.Context(), mustParseURL("foo://"))
	require.NoError(t, err)

	assert.Same(t, schema1, schema2)
	assert.Same(t, schema2, schema3)
	assert.Same(t, schema3, schema1)
	assert.Equal(t, 1, schema1.Enum[0], "schema1")
	assert.Equal(t, 1, schema2.Enum[0], "schema2")
	assert.Equal(t, 1, schema3.Enum[0], "schema3")
}

func TestHTTPLoader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		response     string
		responseType string
		want         *Schema
		wantFunc     func(serverURL string) *Schema
	}{
		{
			name:     "empty object",
			response: `{"$comment": "hello"}`,
			want:     &Schema{Comment: "hello"},
		},
		{
			name:         "JSON content",
			response:     `{"$comment": "hello"}`,
			responseType: "application/json",
			want:         &Schema{Comment: "hello"},
		},
		{
			name:         "charset",
			response:     `{"$comment": "hello"}`,
			responseType: "application/json; charset=utf8",
			want:         &Schema{Comment: "hello"},
		},
		{
			name:         "JSON with YAML content type",
			response:     `{"$comment": "hello"}`,
			responseType: "application/yaml",
			want:         &Schema{Comment: "hello"},
		},
		{
			name:         "YAML content",
			response:     `$comment: hello`,
			responseType: "application/yaml",
			want:         &Schema{Comment: "hello"},
		},
		{
			name:     "with ref",
			response: `{"$ref": "foo.json"}`,
			wantFunc: func(serverURL string) *Schema {
				return &Schema{
					Ref:         "foo.json",
					RefReferrer: ReferrerURL(mustParseURL(serverURL)),
				}
			},
		},
		{
			name:     "with ref subdir",
			response: `{"$ref": "subdir/foo.json"}`,
			wantFunc: func(serverURL string) *Schema {
				return &Schema{
					Ref:         "subdir/foo.json",
					RefReferrer: ReferrerURL(mustParseURL(serverURL)),
				}
			},
		},
		{
			name:     "with ref subdir fragment",
			response: `{"$ref": "subdir/foo.json#/properties/foo"}`,
			wantFunc: func(serverURL string) *Schema {
				return &Schema{
					Ref:         "subdir/foo.json#/properties/foo",
					RefReferrer: ReferrerURL(mustParseURL(serverURL)),
				}
			},
		},
		{
			name:     "invalid ref isnt checked here",
			response: `{"$ref": "::"}`,
			wantFunc: func(serverURL string) *Schema {
				return &Schema{
					Ref:         "::",
					RefReferrer: ReferrerURL(mustParseURL(serverURL)),
				}
			},
		},
	}

	responseTypes := []struct {
		name  string
		write func(t *testing.T, w http.ResponseWriter, msg, contentType string)
	}{
		{
			name: "uncompressed",
			write: func(t *testing.T, w http.ResponseWriter, msg, contentType string) {
				if contentType != "" {
					w.Header().Add("Content-Type", contentType)
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(msg))
				require.NoError(t, err)
			},
		},
		{
			name: "gzip",
			write: func(t *testing.T, w http.ResponseWriter, msg, contentType string) {
				if contentType != "" {
					w.Header().Add("Content-Type", contentType)
				}
				w.Header().Add("Content-Encoding", "gzip")
				w.WriteHeader(http.StatusOK)
				gzipper := gzip.NewWriter(w)
				defer func() {
					assert.NoError(t, gzipper.Close())
				}()
				_, err := gzipper.Write([]byte(msg))
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		for _, writer := range responseTypes {
			t.Run(tt.name+"/"+writer.name, func(t *testing.T) {
				t.Parallel()
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					writer.write(t, w, tt.response, tt.responseType)
				}))
				defer server.Close()

				ctx := t.Context()

				loader := NewHTTPLoader(server.Client())
				schema, err := loader.Load(ctx, mustParseURL(server.URL))
				require.NoError(t, err)

				want := tt.want
				if tt.wantFunc != nil {
					want = tt.wantFunc(server.URL)
				}

				assert.Equal(t, want, schema, "Schema")
			})
		}

		t.Run(tt.name+"/link", func(t *testing.T) {
			t.Parallel()
			var linkHeader string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				linkHeader = req.Header.Get("Link")
				if tt.responseType != "" {
					w.Header().Add("Content-Type", tt.responseType)
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(tt.response))
				require.NoError(t, err)
			}))
			defer server.Close()

			ctx := t.Context()
			ctx = ContextWithLoaderReferrer(ctx, "http://some-referrerer")

			loader := NewHTTPLoader(server.Client())
			_, err := loader.Load(ctx, mustParseURL(server.URL))
			require.NoError(t, err)
			assert.Equal(t, `<http://some-referrerer>; rel="describedby"`, linkHeader, "Link header")
		})
	}
}

func TestHTTPLoader_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		response     string
		responseType string
		responseCode int
		wantErr      string
	}{
		{
			name:         "invalid undefined content",
			response:     `foobar`,
			responseType: "",
			responseCode: http.StatusOK,
			wantErr:      "JSON: invalid character 'o'",
		},
		{
			name:         "invalid JSON content",
			response:     `foobar`,
			responseType: "application/json",
			responseCode: http.StatusOK,
			wantErr:      "JSON: invalid character 'o'",
		},
		{
			name: "valid JSON but invalid YAML content",
			// YAML doesn't allow tabs
			response:     "\t{\"$comment\":\"hello\"}",
			responseType: "application/yaml",
			responseCode: http.StatusOK,
			wantErr:      "YAML: yaml: found character that cannot start any token",
		},
		{
			name:         "invalid YAML content",
			response:     `: foo:`,
			responseType: "application/yaml",
			responseCode: http.StatusOK,
			wantErr:      "YAML: yaml: did not find expected key",
		},
		{
			name:         "invalid charset",
			response:     "{\"$comment\":\"hello\"}",
			responseType: "application/json; charset=iso-whatever-123456",
			responseCode: http.StatusOK,
			wantErr:      "unsupported response charset",
		},
		{
			name:         "invalid response code",
			response:     `{}`,
			responseCode: http.StatusGone,
			wantErr:      "got non-2xx status code: 410 Gone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if tt.responseType != "" {
					w.Header().Add("Content-Type", tt.responseType)
				}
				w.WriteHeader(tt.responseCode)
				_, err := w.Write([]byte(tt.response))
				require.NoError(t, err)
			}))
			defer server.Close()

			ctx := t.Context()

			loader := NewHTTPLoader(server.Client())
			_, err := loader.Load(ctx, mustParseURL(server.URL))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	t.Run("invalid encoding", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Content-Encoding", "foobar")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
		}))
		defer server.Close()

		ctx := t.Context()

		loader := NewHTTPLoader(server.Client())
		_, err := loader.Load(ctx, mustParseURL(server.URL))
		assert.ErrorContains(t, err, `unsupported content encoding: "foobar"`)
	})

	t.Run("size limit", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(bytes.Repeat([]byte(" "), 1000))
			require.NoError(t, err)
		}))
		defer server.Close()

		ctx := t.Context()

		loader := NewHTTPLoader(server.Client())
		loader.SizeLimit = 20
		_, err := loader.Load(ctx, mustParseURL(server.URL))
		assert.ErrorContains(t, err, `aborted request after reading more than 20B`)
	})

	t.Run("shutdown server", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Content-Encoding", "foobar")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
		}))
		server.Close() // close now already

		ctx := t.Context()

		loader := NewHTTPLoader(server.Client())
		_, err := loader.Load(ctx, mustParseURL(server.URL))
		assert.ErrorContains(t, err, `connect: connection refused`)
	})

	t.Run("invalid gzip", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
		}))
		defer server.Close()

		ctx := t.Context()

		loader := NewHTTPLoader(server.Client())
		_, err := loader.Load(ctx, mustParseURL(server.URL))
		assert.ErrorContains(t, err, `create gzip reader: unexpected EOF`)
	})
}

func TestHTTPLoader_NewRequestError(t *testing.T) {
	failHTTPLoaderNewRequest = true
	defer func() { failHTTPLoaderNewRequest = false }()
	loader := NewHTTPLoader(http.DefaultClient)
	_, err := loader.Load(t.Context(), mustParseURL("file://localhost"))
	assert.ErrorContains(t, err, `create request: `)
}

func TestFormatSizeBytes(t *testing.T) {
	tests := []struct {
		name string
		size int
		want string
	}{
		{size: 0, want: "0B"},
		{size: 1000, want: "1000B"},
		{size: 1999, want: "1999B"},
		{size: 2000, want: "2KB"},
		{size: 10_000, want: "10KB"},
		{size: 1_000_000, want: "1000KB"},
		{size: 1_999_999, want: "1999KB"},
		{size: 2_000_000, want: "2MB"},
		{size: 10_000_000, want: "10MB"},
		{size: 3_000_000_000_000, want: "3000000MB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSizeBytes(tt.size)
			if got != tt.want {
				t.Errorf("wrong result\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}
