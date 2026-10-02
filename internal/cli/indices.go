package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/index"
)

type indexInfo struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Formula     string   `json:"formula"`
	Bands       []string `json:"bands"`
}

func newIndicesCommand(env Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "indices",
		Short: "List computable spectral indices",
		Long: `List the spectral indices earth can compute from Sentinel-2 L2A.

Computing an index requires Copernicus Sentinel Hub credentials; use
"earth observe vegetation --index <name>" to compute one over an area.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}

			definitions := index.All()
			if session.json {
				out := make([]indexInfo, 0, len(definitions))
				for _, def := range definitions {
					out = append(out, indexInfo{
						Name:        def.Name,
						Title:       def.Title,
						Description: def.Description,
						Formula:     def.Formula,
						Bands:       def.Bands,
					})
				}
				return session.printer.JSONValue(out)
			}

			rows := make([][]string, 0, len(definitions))
			for _, def := range definitions {
				rows = append(rows, []string{
					def.Name,
					def.Title,
					strings.Join(def.Bands, ","),
					def.Description,
				})
			}
			session.printer.Table([]string{"NAME", "TITLE", "BANDS", "DESCRIPTION"}, rows)
			return nil
		},
	}
}
