package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/apperr"
	"github.com/oscarhugopaz/earth-cli/internal/observation"
	"github.com/oscarhugopaz/earth-cli/internal/output"
)

func newObserveCommand(env Environment) *cobra.Command {
	var (
		bboxRaw    string
		area       string
		from       string
		to         string
		since      string
		limit      int
		resolution float64
		interval   string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "observe <observation>",
		Short: "Resolve a high-level observation (for example vegetation)",
		Long: `Resolve a high-level observation such as "vegetation" into source scenes.

The Earth Engine maps the observation to an appropriate provider collection and
returns what it can truthfully resolve. Derived metrics that require processing
APIs are not fabricated.

Examples:

  earth observe vegetation --area vineyard.geojson --since 90d
  earth observe vegetation --bbox -70.8,-33.6,-70.4,-33.3 --since 90d`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return apperr.Usage("observation name is required\n\nAvailable observations:\n  %s",
					strings.Join(observation.Names(), "\n  "))
			}
			if len(args) > 1 {
				return apperr.Usage("expected a single observation name, got %d", len(args))
			}

			name := strings.TrimSpace(args[0])
			if _, ok := observation.Lookup(name); !ok {
				return apperr.Usage("unknown observation %q\n\nAvailable observations:\n  %s",
					name, strings.Join(observation.Names(), "\n  "))
			}

			bbox, err := resolveArea(bboxRaw, area)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			if bbox == nil {
				return apperr.Usage("observe requires an area of interest: provide --area <file.geojson> or --bbox minLon,minLat,maxLon,maxLat")
			}

			start, end, label, err := parseTimeWindow(from, to, since, nowUTC())
			if err != nil {
				return apperr.Usage("%s", err)
			}
			if limit <= 0 {
				limit = 100
			}

			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}

			result, err := session.engine.Observe(cmd.Context(), name, session.provider, observation.Request{
				BBox:        bbox,
				Start:       start,
				End:         end,
				PeriodLabel: label,
				Limit:       limit,
				Resolution:  resolution,
				Interval:    interval,
				DryRun:      dryRun,
			})
			if err != nil {
				return err
			}

			if session.json {
				return session.printer.JSONValue(result)
			}
			printObservation(session.printer, result)
			return nil
		},
	}

	cmd.Flags().StringVar(&area, "area", "", "GeoJSON file (Polygon or MultiPolygon) of interest")
	cmd.Flags().StringVar(&bboxRaw, "bbox", "", "bounding box minLon,minLat,maxLon,maxLat")
	cmd.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&since, "since", "", "relative window, for example 90d")
	cmd.Flags().IntVar(&limit, "limit", 100, "maximum number of source scenes to consider")
	cmd.Flags().Float64Var(&resolution, "resolution", 10, "ground sample distance in metres for derived indices")
	cmd.Flags().StringVar(&interval, "interval", "P10D", "ISO8601 aggregation interval for derived indices (for example P10D, P30D)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan the derived index and estimate cost without spending quota")
	return cmd
}

func printObservation(printer *output.Printer, result *observation.Result) {
	printer.Field("Observation", result.Observation)
	printer.Field("Provider", result.Provider)
	printer.Field("Source", result.Source)
	if result.Collection != "" {
		printer.Field("Collection", result.Collection)
	}
	if result.Period != "" {
		printer.Field("Period", result.Period)
	}
	if len(result.BBox) == 4 {
		printer.Field("Area", fmt.Sprintf("%g,%g,%g,%g", result.BBox[0], result.BBox[1], result.BBox[2], result.BBox[3]))
	}
	printer.Field("Scenes", fmt.Sprintf("%d", result.Scenes))
	if result.MeanCloudCover != nil {
		printer.Field("Mean cloud", fmt.Sprintf("%.1f%%", *result.MeanCloudCover))
	}
	if result.BestScene != nil {
		printer.Field("Best scene", formatDate(result.BestScene.DateTime))
		if result.BestScene.CloudCover != nil {
			printer.Field("Cloud cover", fmt.Sprintf("%.1f%%", *result.BestScene.CloudCover))
		}
		if result.BestScene.ID != "" {
			printer.Field("Scene ID", result.BestScene.ID)
		}
	}
	if len(result.Bands) > 0 {
		printer.Field("Bands", strings.Join(result.Bands, ", "))
	}
	if result.Formula != "" {
		printer.Field("NDVI formula", result.Formula)
	}
	if result.NDVI != nil && len(result.NDVI.Intervals) > 0 {
		printer.Line("")
		printer.Line("NDVI series (%s, %s)", result.NDVI.Collection, result.NDVI.Interval)
		rows := make([][]string, 0, len(result.NDVI.Intervals))
		for _, interval := range result.NDVI.Intervals {
			rows = append(rows, []string{
				interval.From.UTC().Format(dateLayout),
				interval.To.UTC().Format(dateLayout),
				formatFloat(interval.Mean, 3),
				formatFloat(interval.Min, 3),
				formatFloat(interval.Max, 3),
				formatInt(interval.SampleCount),
			})
		}
		printer.Table([]string{"FROM", "TO", "MEAN", "MIN", "MAX", "SAMPLES"}, rows)
	}
	if result.NDVIPlan != nil {
		plan := result.NDVIPlan
		printer.Line("")
		printer.Line("NDVI plan (dry run)")
		printer.Field("Index", strings.ToUpper(plan.Index))
		printer.Field("Collection", plan.Collection)
		printer.Field("Resolution", fmt.Sprintf("%gm", plan.ResolutionM))
		printer.Field("Interval", plan.Interval)
		if len(plan.Bands) > 0 {
			printer.Field("Bands", strings.Join(plan.Bands, ", "))
		}
		printer.Field("Estimated cost", fmt.Sprintf("~%.2f PU (±%.0f%%)", plan.EstimatedPU, 25.0))
	}
	if result.Note != "" {
		printer.Line("")
		printer.Line("%s", result.Note)
	}
}

func formatFloat(value *float64, precision int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.*f", precision, *value)
}

func formatInt(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}
