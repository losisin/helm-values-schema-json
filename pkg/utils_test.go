package pkg

import "testing"

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
