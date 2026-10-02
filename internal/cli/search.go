package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/apperr"
	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newSearchCommand(env Environment) *cobra.Command {
	var (
		collection string
		bboxRaw    string
		area       string
		from       string
		to         string
		since      string
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search Earth observation items",
		Long: `Search Earth observation items through the selected provider's STAC API.

Examples:

  earth search --collection sentinel-2-l2a \
    --bbox -70.8,-33.6,-70.4,-33.3 --since 30d

  earth search --collection sentinel-2-l2a --area vineyard.geojson \
    --from 2026-09-01 --to 2026-10-01`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(collection) == "" {
				return apperr.Usage("--collection is required")
			}

			bbox, err := resolveArea(bboxRaw, area)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			start, end, _, err := parseTimeWindow(from, to, since, time.Now().UTC())
			if err != nil {
				return apperr.Usage("%s", err)
			}
			if limit <= 0 {
				limit = 10
			}

			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			observations, err := p.Search(cmd.Context(), provider.SearchRequest{
				Collection: collection,
				BBox:       bbox,
				Start:      start,
				End:        end,
				Limit:      limit,
			})
			if err != nil {
				return err
			}

			if session.json {
				return session.printer.JSONValue(observations)
			}
			if len(observations) == 0 {
				session.printer.Line("No items found.")
				return nil
			}

			rows := make([][]string, 0, len(observations))
			for _, observation := range observations {
				rows = append(rows, []string{
					formatDate(observation.DateTime),
					observation.Collection,
					formatCloudCover(observation.CloudCover),
					observation.ID,
				})
			}
			session.printer.Table([]string{"DATE", "COLLECTION", "CLOUD", "ID"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&collection, "collection", "", "collection id to search (required)")
	cmd.Flags().StringVar(&bboxRaw, "bbox", "", "bounding box minLon,minLat,maxLon,maxLat")
	cmd.Flags().StringVar(&area, "area", "", "GeoJSON file (Polygon or MultiPolygon) to search within")
	cmd.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&since, "since", "", "relative window, for example 30d, 12h or 2w")
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of items to return")
	return cmd
}

// resolveArea parses --bbox or --area, rejecting both at once.
func resolveArea(bboxRaw, area string) (*geometry.BBox, error) {
	bboxRaw = strings.TrimSpace(bboxRaw)
	area = strings.TrimSpace(area)

	if bboxRaw != "" && area != "" {
		return nil, fmt.Errorf("--bbox and --area are mutually exclusive; provide only one")
	}
	if bboxRaw != "" {
		bbox, err := geometry.ParseBBox(bboxRaw)
		if err != nil {
			return nil, err
		}
		return &bbox, nil
	}
	if area != "" {
		bbox, err := geometry.LoadArea(area)
		if err != nil {
			return nil, err
		}
		return &bbox, nil
	}
	return nil, nil
}

func formatDate(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.UTC().Format(dateLayout)
}

func formatCloudCover(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", *value)
}
