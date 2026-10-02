package cli

import (
	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/version"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func newVersionCommand(env Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			jsonOut, _ := cmd.Flags().GetBool("json")
			printer := output.New(env.Stdout, env.Stderr, jsonOut, false)
			if jsonOut {
				return printer.JSONValue(versionInfo{
					Version: version.Version,
					Commit:  version.Commit,
					Date:    version.Date,
				})
			}
			// Keep this line stable: scripts and the Homebrew test parse it.
			printer.Line("earth version %s", version.Version)
			return nil
		},
	}
}
