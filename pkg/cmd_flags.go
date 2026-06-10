package pkg

import "github.com/spf13/pflag"

// registerSharedFlags registers the flags shared by the root (generate) command
// and the bundle subcommand, so their names, defaults and usage strings live in
// one place and cannot drift apart. The root command reads them back through
// koanf; the bundle subcommand reads them directly from the flag set.
func registerSharedFlags(fs *pflag.FlagSet) {
	fs.Int("indent", DefaultConfig.Indent, "Indentation spaces (even number)")
	fs.String("bundle-root", "", "Root directory to allow local referenced files to be loaded from (default current working directory)")
	fs.Bool("bundle-without-id", false, "Bundle without using $id to reference bundled schemas, which improves compatibility with e.g the VS Code JSON extension")
	fs.String("bundle-cache-min", "", "Minimum cache duration for downloaded schemas, e.g. 24h or 30m. Raises short server Cache-Control max-age values; empty follows the server")
	fs.String("k8s-schema-url", DefaultConfig.K8sSchemaURL, "URL template used in $ref: $k8s/... alias")
	fs.String("k8s-schema-version", "", "Version used in the --k8s-schema-url template for $ref: $k8s/... alias")
}
