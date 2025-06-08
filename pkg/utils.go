package pkg

import (
	"io"
)

func uniqueStringAppend(dest []string, src ...string) []string {
	existingItems := make(map[string]bool)
	for _, item := range dest {
		existingItems[item] = true
	}

	for _, item := range src {
		if _, exists := existingItems[item]; !exists {
			dest = append(dest, item)
			existingItems[item] = true
		}
	}

	return dest
}

func closeIgnoreError(closer io.Closer) {
	_ = closer.Close()
}

// LimitReaderWithError returns a wrapper around [io.LimitedReader] that
// returns a custom error when the limit is reached instead of [io.EOF].
func LimitReaderWithError(r io.Reader, n int64, err error) LimitedReaderWithError {
	return LimitedReaderWithError{
		Reader: &io.LimitedReader{R: r, N: n},
		Err:    err,
	}
}

// LimitedReaderWithError is a wrapper around [io.LimitedReader] that
// returns a custom error when the limit is reached instead of [io.EOF].
type LimitedReaderWithError struct {
	Reader *io.LimitedReader
	Err    error
}

// Read implements [io.Reader].
func (r LimitedReaderWithError) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	if err == io.EOF && r.Reader.N <= 0 {
		return n, r.Err
	}
	return n, err
}
