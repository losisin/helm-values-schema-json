package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/losisin/helm-values-schema-json/pkg"
)

func main() {
	// Load configuration from a YAML file
	fileConfig, err := pkg.LoadConfig("schema.yaml")
	if err != nil {
		fmt.Println("Error loading config file:", err)
	}

	// Parse CLI flags
	flagConfig, output, err := pkg.ParseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Println(output)
		return
	} else if err != nil {
		fmt.Println("Error parsing flags:", output)
		return
	}

	// Merge configurations, giving precedence to CLI flags
	var finalConfig *pkg.Config
	if fileConfig != nil {
		finalConfig = pkg.MergeConfig(fileConfig, flagConfig)
	} else {
		finalConfig = flagConfig
	}

	err = pkg.GenerateJsonSchema(finalConfig)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
