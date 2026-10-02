package cli

import (
	"github.com/spf13/cobra"
)

type providerInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

func newProvidersCommand(env Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List available Earth observation providers",
		Long: `List the Earth observation providers registered in the Earth Engine.

Copernicus is a provider, not the core architecture: additional STAC-compatible
providers can be added without changing this command.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}

			providers := session.engine.Providers()
			if session.json {
				out := make([]providerInfo, 0, len(providers))
				for _, p := range providers {
					out = append(out, providerInfo{Name: p.Name(), Type: p.Type(), Status: p.Status()})
				}
				return session.printer.JSONValue(out)
			}

			rows := make([][]string, 0, len(providers))
			for _, p := range providers {
				rows = append(rows, []string{p.Name(), p.Type(), p.Status()})
			}
			session.printer.Table([]string{"NAME", "TYPE", "STATUS"}, rows)
			return nil
		},
	}
}
