package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newItemCommand(env Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "item <collection> <item-id>",
		Short: "Show a single item and its assets",
		Long: `Show a single STAC item, including its assets and their URLs.

  earth item sentinel-2-l2a S2B_MSIL2A_20260914T143739_N0512_R096_T19HCC_20260914T195453`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			item, err := p.Item(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}

			if session.json {
				return session.printer.JSONValue(item)
			}
			printItem(session.printer, item)
			return nil
		},
	}
}

func printItem(printer *output.Printer, item provider.Observation) {
	printer.Field("ID", item.ID)
	if item.Collection != "" {
		printer.Field("Collection", item.Collection)
	}
	if item.DateTime != nil {
		printer.Field("Date", item.DateTime.UTC().Format(time.RFC3339))
	}
	if item.CloudCover != nil {
		printer.Field("Cloud cover", fmt.Sprintf("%.1f%%", *item.CloudCover))
	}
	if item.ItemURL != "" {
		printer.Field("Item URL", item.ItemURL)
	}

	if len(item.AssetDetails) == 0 {
		return
	}
	printer.Line("")
	printer.Line("Assets (%d)", len(item.AssetDetails))
	rows := make([][]string, 0, len(item.AssetDetails))
	for _, asset := range item.AssetDetails {
		rows = append(rows, []string{asset.Name, asset.Type, asset.Href})
	}
	printer.Table([]string{"NAME", "TYPE", "HREF"}, rows)
}
