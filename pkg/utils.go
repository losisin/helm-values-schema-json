package pkg

import (
	"cmp"
	"io"
	"iter"
	"maps"
	"net/url"
	"slices"
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

// hasKey returns true when a map contains the given key.
// This utility function allows checking that a key exists in boolean switch statements.
//
// Will always return false if the map is nil.
func hasKey[K comparable, V any](m map[K]V, k K) bool {
	_, ok := m[k]
	return ok
}

func iterMapOrdered[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
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

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

// comparePointer is a helper function for comparing pointer fields
func comparePointer[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func countOccurrencesSlice[T comparable](slice []T, item T) int {
	var count int
	for _, value := range slice {
		if value == item {
			count++
		}
	}
	return count
}

func tryIndex[T any](slice []*T, index int) *T {
	if index < 0 || index >= len(slice) {
		return nil
	}
	return slice[index]
}
