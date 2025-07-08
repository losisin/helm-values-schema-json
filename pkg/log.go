package pkg

import (
	"context"
	"fmt"
	"io"
	"os"
)

type loggerContextKey int

var loggerContextValue = loggerContextKey(1)

// Logger is an interface that allows writing log messages to the user.
//
// The interface is intentionally shaped to also allow [testing.T] as a logger.
type Logger interface {
	// Prints message as a line.
	// No need to suffix with a newline.
	Log(a ...any)
	// Prints formatted message as a line.
	// No need to suffix with a newline.
	Logf(format string, a ...any)
}

// ContextWithLogger returns a derived context that sets the current logger.
// This logger is later retrieved using [LoggerFromContext].
func ContextWithLogger(parent context.Context, logger Logger) context.Context {
	return context.WithValue(parent, loggerContextValue, logger)
}

// LoggerFromContext tries to find the current logger from the context,
// and if none set then will return a new logger that uses [os.Stdout].
func LoggerFromContext(ctx context.Context) Logger {
	logger := ctx.Value(loggerContextValue)
	if logger == nil {
		return NewLogger(os.Stderr)
	}
	return logger.(Logger)
}

// NewLogger returns a new logger using the provided writer.
func NewLogger(output io.Writer) WriterLogger {
	return WriterLogger{output}
}

// WriterLogger is a [Logger] implementation that writes to a [io.Writer].
type WriterLogger struct {
	Output io.Writer
}

// ensures it implements the interface
var _ Logger = WriterLogger{}

func (logger WriterLogger) Log(a ...any) {
	_, _ = fmt.Fprintln(logger.Output, a...)
}

func (logger WriterLogger) Logf(format string, a ...any) {
	_, _ = fmt.Fprintf(logger.Output, format+"\n", a...)
}
