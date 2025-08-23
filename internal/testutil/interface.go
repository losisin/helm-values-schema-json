package testutil

// T is a smaller interface than [testing.T] to only provide the minimum methods
// needed, so it is easier to mock out in meta-testing.
type T interface {
	Helper()
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}

// FakeT is a simple overridable interface implementation of [T],
// allowing for custom implementations during meta-testing.
type FakeT struct {
	HelperFunc func()
	FatalfFunc func(format string, args ...any)
	ErrorfFunc func(format string, args ...any)
}

var _ T = FakeT{}

// Errorf implements [T].
func (f FakeT) Errorf(format string, args ...any) { f.ErrorfFunc(format, args...) }

// Fatalf implements [T].
func (f FakeT) Fatalf(format string, args ...any) { f.FatalfFunc(format, args...) }

// Helper implements [T].
func (f FakeT) Helper() { f.HelperFunc() }
