package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/losisin/helm-values-schema-json/pkg"
)

func main() {
	conf, output, err := pkg.ParseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Println(output)
		return
	} else if err != nil {
		fmt.Println("Error:", output)
		return
	}

	err = pkg.GenerateJsonSchema(conf)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
