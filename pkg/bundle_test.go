package pkg

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
					"foo": {
						Ref: "../some/file.json",
					},
				},
			},
			loader: DummyLoader{
				LoadFunc: func(ctx context.Context, ref *url.URL) (*Schema, error) {
					return &Schema{}, nil
				},
			},
			want: &Schema{
				Properties: map[string]*Schema{
					"foo": {
						Ref: "../some/file.json",
					},
				},
				Defs: map[string]*Schema{
					"file.json": {
						ID: "../some/file.json",
					},
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
					return &Schema{
						Comment: "Referred by: " + referrer,
					}, nil
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleSchema(t.Context(), tt.loader, tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.schema)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := BundleSchema(t.Context(), tt.loader, tt.schema)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
