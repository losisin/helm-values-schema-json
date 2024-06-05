package pkg

import (
	"errors"
	"reflect"
	"testing"
)

func TestMultiStringFlagString(t *testing.T) {
	tests := []struct {
		input    multiStringFlag
		expected string
	}{
		{
			input:    multiStringFlag{},
			expected: "",
		},
		{
			input:    multiStringFlag{"value1"},
			expected: "value1",
		},
		{
			input:    multiStringFlag{"value1", "value2", "value3"},
			expected: "value1, value2, value3",
		},
	}

	for i, test := range tests {
		result := test.input.String()
		if result != test.expected {
			t.Errorf("Test case %d: Expected %q, but got %q", i+1, test.expected, result)
		}
	}
}

func TestMultiStringFlagSet(t *testing.T) {
	tests := []struct {
		input     string
		initial   multiStringFlag
		expected  multiStringFlag
		errorFlag bool
	}{
		{
			input:     "value1,value2,value3",
			initial:   multiStringFlag{},
			expected:  multiStringFlag{"value1", "value2", "value3"},
			errorFlag: false,
		},
		{
			input:     "",
			initial:   multiStringFlag{"existingValue"},
			expected:  multiStringFlag{"existingValue"},
			errorFlag: false,
		},
		{
			input:     "value1, value2, value3",
			initial:   multiStringFlag{},
			expected:  multiStringFlag{"value1", "value2", "value3"},
			errorFlag: false,
		},
	}

	for i, test := range tests {
		err := test.initial.Set(test.input)
		if err != nil && !test.errorFlag {
			t.Errorf("Test case %d: Expected no error, but got: %v", i+1, err)
		} else if err == nil && test.errorFlag {
			t.Errorf("Test case %d: Expected an error, but got none", i+1)
		}
	}
}

func TestUniqueStringAppend(t *testing.T) {
	tests := []struct {
		name string
		dest []string
		src  []string
		want []string
	}{
		{
			name: "empty slices",
			dest: []string{},
			src:  []string{},
			want: []string{},
		},
		{
			name: "unique items",
			dest: []string{"a", "b"},
			src:  []string{"c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "duplicate items",
			dest: []string{"a", "b"},
			src:  []string{"b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "all duplicate items",
			dest: []string{"a", "b"},
			src:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "src only has duplicates",
			dest: []string{"c", "d"},
			src:  []string{"c", "c", "d", "d"},
			want: []string{"c", "d"},
		},
		{
			name: "empty dest slice",
			dest: []string{},
			src:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "empty src slice",
			dest: []string{"a", "b"},
			src:  []string{},
			want: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the 'dest' slice to preserve the original.
			destCopy := make([]string, len(tt.dest))
			copy(destCopy, tt.dest)

			// Call the function with the copy.
			got := uniqueStringAppend(destCopy, tt.src...)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uniqueStringAppend() = %v, want %v", got, tt.want)
			}

			// Verify that the original 'dest' slice is not modified.
			if !reflect.DeepEqual(tt.dest, destCopy) {
				t.Errorf("uniqueStringAppend() unexpectedly modified the original dest slice")
			}
		})
	}
}

func TestBoolFlag_String(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		value    bool
		expected string
	}{
		{"Unset flag", false, false, "not set"},
		{"Set flag to true", true, true, "true"},
		{"Set flag to false", true, false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BoolFlag{
				set:   tt.set,
				value: tt.value,
			}
			if got := b.String(); got != tt.expected {
				t.Errorf("BoolFlag.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBoolFlag_Set(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedSet   bool
		expectedValue bool
		expectedErr   error
	}{
		{"Set true", "true", true, true, nil},
		{"Set false", "false", true, false, nil},
		{"Set invalid", "invalid", false, false, errors.New("invalid boolean value")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BoolFlag{}
			err := b.Set(tt.input)
			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("BoolFlag.Set() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			if err != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("BoolFlag.Set() error = %v, expectedErr %v", err, tt.expectedErr)
			}
			if b.set != tt.expectedSet {
				t.Errorf("BoolFlag.set = %v, expected %v", b.set, tt.expectedSet)
			}
			if b.value != tt.expectedValue {
				t.Errorf("BoolFlag.value = %v, expected %v", b.value, tt.expectedValue)
			}
		})
	}
}

func TestBoolFlag_IsSet(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		expected bool
	}{
		{"Flag is not set", false, false},
		{"Flag is set", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BoolFlag{
				set: tt.set,
			}
			if got := b.IsSet(); got != tt.expected {
				t.Errorf("BoolFlag.IsSet() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBoolFlag_GetValue(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected bool
	}{
		{"Flag value is false", false, false},
		{"Flag value is true", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BoolFlag{
				value: tt.value,
			}
			if got := b.Value(); got != tt.expected {
				t.Errorf("BoolFlag.GetValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}
