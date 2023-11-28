package pkg

import (
	"flag"
	"reflect"
	"strings"
	"testing"
)

func TestParseFlagsPass(t *testing.T) {
	var tests = []struct {
		args []string
		conf Config
	}{
		{[]string{"-input", "values.yaml"},
			Config{input: multiStringFlag{"values.yaml"}, outputPath: "values.schema.json", draft: 2020, args: []string{}}},

		{[]string{"-input", "values1.yaml values2.yaml"},
			Config{input: multiStringFlag{"values1.yaml values2.yaml"}, outputPath: "values.schema.json", draft: 2020, args: []string{}}},

		{[]string{"-input", "values.yaml", "-output", "my.schema.json", "-draft", "2019"},
			Config{input: multiStringFlag{"values.yaml"}, outputPath: "my.schema.json", draft: 2019, args: []string{}}},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := ParseFlags("schema", tt.args)
			if err != nil {
				t.Errorf("err got %v, want nil", err)
			}
			if output != "" {
				t.Errorf("output got %q, want empty", output)
			}
			if !reflect.DeepEqual(*conf, tt.conf) {
				t.Errorf("conf got %+v, want %+v", *conf, tt.conf)
			}
		})
	}
}

func TestParseFlagsUsage(t *testing.T) {
	var usageArgs = []string{"-help", "-h", "--help"}

	for _, arg := range usageArgs {
		t.Run(arg, func(t *testing.T) {
			conf, output, err := ParseFlags("schema", []string{arg})
			if err != flag.ErrHelp {
				t.Errorf("err got %v, want ErrHelp", err)
			}
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if !strings.Contains(output, "Usage of") {
				t.Errorf("output can't find \"Usage of\": %q", output)
			}
		})
	}
}

func TestParseFlagsFail(t *testing.T) {
	var tests = []struct {
		args   []string
		errStr string
	}{
		{[]string{"-input"}, "flag needs an argument"},
		{[]string{"-draft", "foo"}, "invalid value"},
		{[]string{"-foo"}, "flag provided but not defined"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := ParseFlags("schema", tt.args)
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if !strings.Contains(err.Error(), tt.errStr) {
				t.Errorf("err got %q, want to find %q", err.Error(), tt.errStr)
			}
			if !strings.Contains(output, "Usage of") {
				t.Errorf("output got %q", output)
			}
		})
	}
}
