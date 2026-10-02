package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/apperr"
	"github.com/oscarhugopaz/earth-cli/internal/version"
)

func newRootCommand(env Environment) *cobra.Command {
	root := &cobra.Command{
		Use:   "earth",
		Short: "A developer-friendly CLI for programmable Earth observation",
		Long: `earth is a developer-friendly CLI for programmable Earth observation.

It discovers and searches Earth observation data through provider-oriented
STAC catalogs, starting with the Copernicus Data Space Ecosystem, and exposes
high-level observations such as vegetation without requiring you to know STAC,
satellite band names, or product naming.

Use "earth <command> --json" for stable machine-readable output.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}
	root.SetVersionTemplate("earth version {{.Version}}\n")
	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return apperr.Usage("%s", err)
	})

	flags := root.PersistentFlags()
	flags.Bool("json", false, "write machine-readable JSON to stdout")
	flags.String("provider", "", "provider to query (defaults to the configured provider)")
	flags.Duration("timeout", 30*time.Second, "HTTP timeout per provider request")
	flags.Bool("no-color", false, "disable ANSI colors")

	root.AddCommand(
		newVersionCommand(env),
		newProvidersCommand(env),
		newConfigCommand(env),
		newIndicesCommand(env),
		newCollectionsCommand(env),
		newCollectionCommand(env),
		newItemCommand(env),
		newSearchCommand(env),
		newCompareCommand(env),
		newChangeCommand(env),
		newObserveCommand(env),
	)
	return root
}
