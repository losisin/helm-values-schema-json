package main

import (
	"fmt"
	"os"

	"github.com/losisin/helm-values-schema-json/v2/pkg"
)

func main() {
	cmd := pkg.NewCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Println("Error:", err)
	}
}
