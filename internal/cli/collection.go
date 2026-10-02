package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newCollectionCommand(env Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "collection <collection-id>",
		Short: "Show details for a single collection",
		Long: `Show normalized metadata for one collection, when the STAC API provides it.

  earth collection sentinel-2-l2a`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			collection, err := p.Collection(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if session.json {
				return session.printer.JSONValue(collection)
			}
			printCollection(session.printer, collection)
			return nil
		},
	}
}

func printCollection(printer *output.Printer, collection provider.Collection) {
	printer.Field("ID", collection.ID)
	if collection.Title != "" {
		printer.Field("Title", collection.Title)
	}
	if collection.Description != "" {
		printer.Field("Description", collection.Description)
	}
	if collection.License != "" {
		printer.Field("License", collection.License)
	}
	if len(collection.Keywords) > 0 {
		printer.Field("Keywords", strings.Join(collection.Keywords, ", "))
	}
	if collection.Extent != nil {
		if len(collection.Extent.Spatial) > 0 {
			printer.Field("Spatial", formatBBox(collection.Extent.Spatial[0]))
		}
		if len(collection.Extent.Temporal) > 0 {
			printer.Field("Temporal", formatInterval(collection.Extent.Temporal[0]))
		}
	}
	if collection.ItemURL != "" {
		printer.Field("Item URL", collection.ItemURL)
	}
	if collection.QueryablesURL != "" {
		printer.Field("Queryables", collection.QueryablesURL)
	}
}

func formatBBox(bbox []float64) string {
	if len(bbox) < 4 {
		return fmt.Sprintf("%v", bbox)
	}
	return fmt.Sprintf("%g,%g,%g,%g", bbox[0], bbox[1], bbox[2], bbox[3])
}

func formatInterval(interval []*string) string {
	start, end := "open", "present"
	if len(interval) > 0 && interval[0] != nil && *interval[0] != "" {
		start = *interval[0]
	}
	if len(interval) > 1 && interval[1] != nil && *interval[1] != "" {
		end = *interval[1]
	}
	return start + " .. " + end
}
