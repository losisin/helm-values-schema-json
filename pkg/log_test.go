package pkg

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterLogger(t *testing.T) {
	t.Run("log", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf)
		logger.Log("hello", "there")
		assert.Equal(t, "hello there\n", buf.String())
	})

	t.Run("logf", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf)
		logger.Logf("hello %q", "there")
		assert.Equal(t, "hello \"there\"\n", buf.String())
	})

	t.Run("default logger from empty context", func(t *testing.T) {
		logger := LoggerFromContext(context.Background())
		require.IsType(t, WriterLogger{}, logger)
		assert.Same(t, os.Stderr, logger.(WriterLogger).Output)
	})

	t.Run("logger from context", func(t *testing.T) {
		ctx := ContextWithLogger(context.Background(), NewLogger(nil))
		logger := LoggerFromContext(ctx)
		require.IsType(t, WriterLogger{}, logger)
		assert.Nil(t, logger.(WriterLogger).Output)
	})
}
