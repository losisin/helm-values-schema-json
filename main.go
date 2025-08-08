package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/losisin/helm-values-schema-json/v2/pkg"
)

// This is set by goreleaser using ldflags
var Version string

// Using variable to allow mocking this during tests
var osExit = os.Exit

func main() {
	cmd := pkg.NewCmd()
	cmd.Version = getVersion()
	pkg.HTTPLoaderDefaultUserAgent = fmt.Sprintf("helm-values-schema-json/%s", cmd.Version)

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	if err := cmd.Execute(); err != nil {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			osExit(1)
		}
	}
}

func getVersion() string {
	if Version != "" {
		return "v" + strings.TrimPrefix(Version, "v")
	}
	return getVersionFromBuildInfo(debug.ReadBuildInfo())
}

func getVersionFromBuildInfo(info *debug.BuildInfo, ok bool) string {
	if !ok {
		return "(devel)" // same string used by Go when running `go run .`
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	var revision string
	var dirty bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if revision != "" {
		if dirty {
			return revision + "-dirty"
		}
		return revision
	}
	return "(devel)"
}
