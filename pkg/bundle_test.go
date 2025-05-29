package pkg

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateBundledName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   string
		defs map[string]*Schema
		want string
	}{
		{
			name: "empty id",
			id:   "",
			defs: map[string]*Schema{},
			want: "",
		},
		{
			name: "new item",
			id:   "foo.json",
			defs: map[string]*Schema{},
			want: "foo.json",
		},
		{
			name: "colliding item",
			id:   "some/path/foo.json",
			defs: map[string]*Schema{
				"foo.json": {
					ID: "some/other/path/foo.json",
				},
			},
			want: "foo.json_2",
		},
		{
			name: "existing item",
			id:   "foo.json",
			defs: map[string]*Schema{
				"foo.json": {
					ID: "foo.json",
				},
			},
			want: "foo.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBundledName(tt.id, tt.defs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBundle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema *Schema
		loader Loader
		want   *Schema
	}{
		{
			name:   "empty schema",
			schema: &Schema{},
			loader: DummyLoader{},
			want:   &Schema{},
		},

		{
			name: "sets $id",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
				},
				Defs: map[string]*Schema{
					"file.json": {ID: "../some/file.json"},
				},
			},
		},

		{
			name: "sets context ID",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						ID:  "some-schema-id",
						Ref: "../some/file.json",
					},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					var referrer string
					if v, ok := ctx.Value(loaderContextReferrer).(string); ok {
						referrer = v
					}
					return &Schema{Comment: "Referred by: " + referrer}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						ID:  "some-schema-id",
						Ref: "../some/file.json",
					},
				},
				Defs: map[string]*Schema{
					"file.json": {
						ID:      "../some/file.json",
						Comment: "Referred by: some-schema-id",
					},
				},
			},
		},

		{
			name: "only bundle once",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
					"bar": {Ref: "../some/file.json"},
					"moo": {Ref: "../some/file.json"},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {Ref: "../some/file.json"},
					"bar": {Ref: "../some/file.json"},
					"moo": {Ref: "../some/file.json"},
				},
				Defs: map[string]*Schema{
					"file.json": {ID: "../some/file.json"},
				},
			},
		},

		{
			name: "already bundled self",
			schema: &Schema{
				Defs: map[string]*Schema{
					"file.json": {
						ID: "../some/file.json",
						Properties: map[string]*Schema{
							"foo": {Ref: "../some/file.json"},
						},
					},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Defs: map[string]*Schema{
					"file.json": {
						ID: "../some/file.json",
						Properties: map[string]*Schema{
							"foo": {Ref: "../some/file.json"},
						},
					},
				},
			},
		},

		{
			name: "subschema is bundled using id",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
			},
			loader: DummyLoader{
				func(ctx context.Context, ref *url.URL) (*Schema, error) {
					switch ref.String() {
					case "foo.json":
						return &Schema{
							Properties: map[string]*Schema{
								"num": {Ref: "bar.json"},
							},
							Defs: map[string]*Schema{
								"bar.json": {
									ID:   "bar.json",
									Type: "number",
								},
							},
						}, nil
					default:
						return nil, fmt.Errorf("undefined test schema: %s", ref)
					}
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID: "foo.json",
						Properties: map[string]*Schema{
							"num": {Ref: "bar.json"},
						},
					},
					"bar.json": {
						ID:   "bar.json",
						Type: "number",
					},
				},
			},
		},

		{
			name: "subschema is bundled without id",
			schema: &Schema{
				Items: &Schema{Ref: "foo.json"},
			},
			loader: DummyLoader{
				func(ctx context.Context, ref *url.URL) (*Schema, error) {
					switch ref.String() {
					case "foo.json":
						return &Schema{
							Properties: map[string]*Schema{
								"num": {Ref: "#/$defs/bar.json"},
							},
							Defs: map[string]*Schema{
								"bar.json": {
									Type: "number",
								},
							},
						}, nil
					default:
						return nil, fmt.Errorf("undefined test schema: %s", ref)
					}
				},
			},
			want: &Schema{
				Items: &Schema{Ref: "foo.json"},
				Defs: map[string]*Schema{
					"foo.json": {
						ID: "foo.json",
						Properties: map[string]*Schema{
							"num": {Ref: "#/$defs/bar.json"},
						},
						Defs: map[string]*Schema{
							"bar.json": {
								Type: "number",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleSchema(t.Context(), tt.loader, tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)

			want, err := yaml.Marshal(tt.want)
			require.NoError(t, err)
			t.Logf("Want:\n%s", string(want))

			got, err := yaml.Marshal(tt.schema)
			require.NoError(t, err)
			t.Logf("Got:\n%s", string(got))
		})
	}
}

func TestBundle_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		schema  *Schema
		loader  Loader
		wantErr string
	}{
		{
			name:    "nil loader",
			schema:  &Schema{},
			loader:  nil,
			wantErr: "nil loader",
		},
		{
			name:    "nil schema",
			schema:  nil,
			loader:  DummyLoader{},
			wantErr: "nil schema",
		},
		{
			name: "invalid URL",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "::",
					},
				},
			},
			loader:  DummyLoader{},
			wantErr: `/properties/foo: parse $ref as URL: parse "::": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleSchema(t.Context(), tt.loader, tt.schema)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestBundleRemoveIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema *Schema
		want   *Schema
	}{
		{
			name:   "empty schema",
			schema: &Schema{},
			want:   &Schema{},
		},

		{
			name: "single subschema has inline ref",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						ID:    "foo.json",
						Items: &Schema{Ref: "#/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
			want: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						Items: &Schema{Ref: "#/$defs/foo/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
		},

		{
			name: "sub-subschema has inline ref",
			schema: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						ID:    "foo.json",
						Items: &Schema{Ref: "#/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Ref: "bar.json"},
						},
					},
					"bar": {
						ID:    "bar.json",
						Items: &Schema{Ref: "#/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
			want: &Schema{
				Defs: map[string]*Schema{
					"foo": {
						Items: &Schema{Ref: "#/$defs/foo/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Ref: "#/$defs/bar"},
						},
					},
					"bar": {
						Items: &Schema{Ref: "#/$defs/bar/$defs/items"},
						Defs: map[string]*Schema{
							"items": {Type: "number"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleRemoveIDs(tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)

			want, err := yaml.Marshal(tt.want)
			require.NoError(t, err)
			t.Logf("Want:\n%s", string(want))

			got, err := yaml.Marshal(tt.schema)
			require.NoError(t, err)
			t.Logf("Got:\n%s", string(got))
		})
	}
}

func TestBundleRemoveIDs_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		schema  *Schema
		wantErr string
	}{
		{
			name:    "nil schema",
			schema:  nil,
			wantErr: "nil schema",
		},
		{
			name: "invalid URL",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "::",
					},
				},
			},
			wantErr: `/properties/foo: parse $ref="::" as URL: parse "::": missing protocol scheme`,
		},
		{
			name: "invalid ref",
			schema: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "./no/$defs/with/this/ref",
					},
				},
			},
			wantErr: `/properties/foo: no $defs found that matches $ref="./no/$defs/with/this/ref"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleRemoveIDs(tt.schema)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestLoad_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		loader  Loader
		ref     string
		wantErr string
	}{
		{
			name:    "nil loader",
			loader:  nil,
			wantErr: "nil loader",
		},
		{
			name:    "empty ref",
			loader:  DummyLoader{},
			wantErr: "cannot load empty $ref",
		},
		{
			name:    "invalid URL",
			loader:  DummyLoader{},
			ref:     "::",
			wantErr: `parse $ref as URL: parse "::": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(t.Context(), tt.loader, tt.ref)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestIterSubschemas_order(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
	}{
		{
			name: "items",
			schema: &Schema{
				Items: &Schema{ID: "a"},
				OneOf: []*Schema{{ID: "b"}, {ID: "c"}},
			},
		},
		{
			name: "properties",
			schema: &Schema{
				Properties: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "patternProperties",
			schema: &Schema{
				PatternProperties: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "defs",
			schema: &Schema{
				Defs: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "definitions",
			schema: &Schema{
				Definitions: map[string]*Schema{"a": {ID: "a"}, "b": {ID: "b"}, "c": {ID: "c"}, "d": {ID: "d"}},
			},
		},
		{
			name: "allOf",
			schema: &Schema{
				AllOf: []*Schema{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			},
		},
		{
			name: "anyOf",
			schema: &Schema{
				AnyOf: []*Schema{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			},
		},
		{
			name: "oneOf",
			schema: &Schema{
				OneOf: []*Schema{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
			},
		},
		{
			name: "not",
			schema: &Schema{
				OneOf: []*Schema{{ID: "a"}, {ID: "b"}},
				Not:   &Schema{ID: "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to ensure we dont get lucky with the ordering
			for range 10 {
				var ids []string
				for _, sub := range iterSubschemas(tt.schema) {
					ids = append(ids, sub.ID)
					if len(ids) == 3 {
						break
					}
				}
				require.Equal(t, "abc", strings.Join(ids, ""))
			}
		})
	}
}

func TestFileLoader_Error(t *testing.T) {
	tests := []struct {
		name    string
		url     *url.URL
		wantErr string
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
			wantErr: `parse $ref="./invalid-schema.json" JSON file: invalid character 'h' in literal true`,
		},
		{
			name:    "invalid YAML",
			url:     mustParseURL("./invalid-schema.yaml"),
			wantErr: `parse $ref="./invalid-schema.yaml" YAML file: yaml: did not find expected key`,
		},
	}

	root, err := os.OpenRoot("../testdata/bundle")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, root.Close())
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewFileLoader(root, "")
			_, err := loader.Load(t.Context(), tt.url)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
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
	}

	for _, tt := range tests {
		t.Run(tt.name+"/simple", func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if tt.responseType != "" {
					w.Header().Add("Content-Type", tt.responseType)
				}
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(tt.response))
				require.NoError(t, err)
			}))
			defer server.Close()

			ctx := t.Context()

			loader := NewHTTPLoader(server.Client())
			schema, err := loader.Load(ctx, mustParseURL(server.URL))
			require.NoError(t, err)
			assert.Equal(t, tt.want, schema, "Schema")
		})

		t.Run(tt.name+"/gzip", func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if tt.responseType != "" {
					w.Header().Add("Content-Type", tt.responseType)
				}
				w.Header().Add("Content-Encoding", "gzip")
				w.WriteHeader(http.StatusOK)
				gzipper := gzip.NewWriter(w)
				defer func() {
					require.NoError(t, gzipper.Close())
				}()
				_, err := gzipper.Write([]byte(tt.response))
				require.NoError(t, err)
			}))
			defer server.Close()

			ctx := t.Context()

			loader := NewHTTPLoader(server.Client())
			schema, err := loader.Load(ctx, mustParseURL(server.URL))
			require.NoError(t, err)
			assert.Equal(t, tt.want, schema, "Schema")
		})

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

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
