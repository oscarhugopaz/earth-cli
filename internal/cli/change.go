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

// changeStats summarizes one window's index statistics.
type changeStats struct {
	Scenes      int      `json:"scenes"`
	Period      string   `json:"period"`
	Mean        *float64 `json:"mean,omitempty"`
	P10         *float64 `json:"p10,omitempty"`
	P50         *float64 `json:"p50,omitempty"`
	P90         *float64 `json:"p90,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	SampleCount *int     `json:"sample_count,omitempty"`
}

// changeResult is the JSON/human shape for `earth change`.
type changeResult struct {
	Observation string      `json:"observation"`
	Provider    string      `json:"provider"`
	Collection  string      `json:"collection"`
	Index       string      `json:"index"`
	Formula     string      `json:"formula,omitempty"`
	BBox        []float64   `json:"bbox,omitempty"`
	Before      changeStats `json:"before"`
	After       changeStats `json:"after"`
	DeltaMean   *float64    `json:"delta_mean,omitempty"`
	DeltaP50    *float64    `json:"delta_p50,omitempty"`
	Note        string      `json:"note,omitempty"`
}

func newChangeCommand(env Environment) *cobra.Command {
	var (
		observationName string
		collection      string
		bboxRaw         string
		area            string
		// Before window
		beforeFrom string
		beforeTo   string
		// After window
		afterFrom string
		afterTo   string

		indexName  string
		interval   string
		resolution float64
		limit      int
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "change",
		Short: "Measure change for an observation between two time windows",
		Long: `Measure change over one area between two time windows.

Reports the index statistics of each window and their difference, using the
spread (p10/p50/p90) rather than only the mean, so localized change is visible
even when the average barely moves.

  earth change --observation burnt-area --area burn.geojson \
    --before-from 2026-01-01 --before-to 2026-02-01 \
    --after-from 2026-02-15 --after-to 2026-03-15`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			bbox, err := resolveArea(bboxRaw, area)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			if bbox == nil {
				return apperr.Usage("change requires an area of interest: provide --area or --bbox")
			}

			beforeStart, beforeEnd, beforeLabel, err := parseTimeWindow(beforeFrom, beforeTo, "", nowUTC())
			if err != nil {
				return apperr.Usage("--before: %s", err)
			}
			afterStart, afterEnd, afterLabel, err := parseTimeWindow(afterFrom, afterTo, "", nowUTC())
			if err != nil {
				return apperr.Usage("--after: %s", err)
			}
			if beforeStart == nil || beforeEnd == nil || afterStart == nil || afterEnd == nil {
				return apperr.Usage("change requires a full window for each side: use --before-from/--before-to and --after-from/--after-to")
			}

			observationName = strings.TrimSpace(observationName)
			if observationName == "" {
				observationName = "vegetation"
			}

			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			// Resolve the observation (and its default collection/index) via the
			// shared engine so change stays consistent with observe/compare.
			resolved, err := session.engine.ResolveObservation(observationName, collection)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			collection = resolved.Collection

			indexName = strings.TrimSpace(indexName)
			if indexName == "" {
				indexName = resolved.Index
			}

			if limit <= 0 {
				limit = 200
			}

			result := &changeResult{
				Observation: resolved.Name,
				Provider:    p.Name(),
				Collection:  collection,
				Index:       indexName,
				BBox:        bbox.Slice(),
			}

			before := window{Start: beforeStart, End: beforeEnd, Label: beforeLabel}
			after := window{Start: afterStart, End: afterEnd, Label: afterLabel}

			beforeStats, beforeSeries, err := measureWindow(cmd, p, collection, bbox, indexName, interval, resolution, limit, before, dryRun)
			if err != nil {
				return err
			}
			afterStats, afterSeries, err := measureWindow(cmd, p, collection, bbox, indexName, interval, resolution, limit, after, dryRun)
			if err != nil {
				return err
			}
			result.Before = beforeStats
			result.After = afterStats

			if dryRun {
				result.Note = "Dry run: no index was computed and no processing units were spent."
				if session.json {
					return session.printer.JSONValue(result)
				}
				printChange(session.printer, result)
				return nil
			}

			// Collapse the per-interval series into one set of stats per window.
			result.Before = collapse(beforeStats, beforeSeries)
			result.After = collapse(afterStats, afterSeries)
			result.Formula = formulaOf(beforeSeries, afterSeries)
			result.DeltaMean = diff(result.Before.Mean, result.After.Mean)
			result.DeltaP50 = diff(result.Before.P50, result.After.P50)

			if session.json {
				return session.printer.JSONValue(result)
			}
			printChange(session.printer, result)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&observationName, "observation", "vegetation", "semantic observation (vegetation, flood, burnt-area, moisture)")
	flags.StringVar(&collection, "collection", "", "override the observation's collection")
	flags.StringVar(&bboxRaw, "bbox", "", "bounding box minLon,minLat,maxLon,maxLat")
	flags.StringVar(&area, "area", "", "GeoJSON file of interest")

	flags.StringVar(&beforeFrom, "before-from", "", "before window start date")
	flags.StringVar(&beforeTo, "before-to", "", "before window end date")
	flags.StringVar(&afterFrom, "after-from", "", "after window start date")
	flags.StringVar(&afterTo, "after-to", "", "after window end date")

	flags.StringVar(&indexName, "index", "", "spectral index (defaults to the observation's)")
	flags.StringVar(&interval, "interval", "P30D", "internal aggregation interval")
	flags.Float64Var(&resolution, "resolution", 20, "ground sample distance in metres")
	flags.IntVar(&limit, "limit", 200, "maximum scenes per window")
	flags.BoolVar(&dryRun, "dry-run", false, "plan both windows without spending quota")
	return cmd
}

type window struct {
	Start *time.Time
	End   *time.Time
	Label string
}

func measureWindow(cmd *cobra.Command, p provider.Provider, collection string, bbox *geometry.BBox, indexName, interval string, resolution float64, limit int, w window, dryRun bool) (changeStats, *provider.IndexSeries, error) {
	stats := changeStats{Period: w.Label}

	observations, err := p.Search(cmd.Context(), provider.SearchRequest{
		Collection: collection,
		BBox:       bbox,
		Start:      w.Start,
		End:        w.End,
		Limit:      limit,
	})
	if err != nil {
		return stats, nil, err
	}
	stats.Scenes = len(observations)

	if dryRun {
		return stats, nil, nil
	}

	indexer, ok := p.(provider.IndexProvider)
	if !ok || !indexer.SupportsIndex() {
		return stats, nil, nil
	}

	series, err := indexer.IndexSeries(cmd.Context(), provider.IndexRequest{
		Collection:  collection,
		Index:       indexName,
		BBox:        bbox,
		Start:       w.Start,
		End:         w.End,
		Interval:    interval,
		Resolution:  resolution,
		Percentiles: []int{10, 50, 90},
	})
	if err != nil {
		return stats, nil, err
	}
	return stats, &series, nil
}

// collapse reduces a per-interval series to a single set of statistics by
// merging intervals: the mean is sample-weighted, percentiles and extremes are
// pooled.
func collapse(base changeStats, series *provider.IndexSeries) changeStats {
	if series == nil {
		return base
	}

	var weightedSum, weight float64
	percentileBuckets := map[string][]float64{}
	var min, max *float64
	var samples int

	for _, interval := range series.Intervals {
		if interval.Mean != nil {
			w := 1.0
			if interval.SampleCount != nil && *interval.SampleCount > 0 {
				w = float64(*interval.SampleCount)
				samples += *interval.SampleCount
			} else {
				samples += 1
			}
			weightedSum += *interval.Mean * w
			weight += w
		}
		for key, value := range interval.Percentiles {
			percentileBuckets[key] = append(percentileBuckets[key], value)
		}
		if interval.Min != nil && (min == nil || *interval.Min < *min) {
			min = interval.Min
		}
		if interval.Max != nil && (max == nil || *interval.Max > *max) {
			max = interval.Max
		}
	}

	if weight > 0 {
		mean := weightedSum / weight
		base.Mean = &mean
	}
	if samples > 0 {
		base.SampleCount = &samples
	}
	base.Min = min
	base.Max = max
	base.P10 = averagePercentile(percentileBuckets, "10")
	base.P50 = averagePercentile(percentileBuckets, "50")
	base.P90 = averagePercentile(percentileBuckets, "90")
	return base
}

// averagePercentile looks up a percentile key, tolerating the "10" vs "10.0"
// spellings Sentinel Hub uses.
func averagePercentile(buckets map[string][]float64, want string) *float64 {
	for key, values := range buckets {
		if !samePercentile(key, want) || len(values) == 0 {
			continue
		}
		var sum float64
		for _, value := range values {
			sum += value
		}
		avg := sum / float64(len(values))
		return &avg
	}
	return nil
}

func samePercentile(key, want string) bool {
	trimmed := strings.TrimSuffix(key, ".0")
	return strings.TrimSuffix(want, ".0") == trimmed
}

func diff(before, after *float64) *float64 {
	if before == nil || after == nil {
		return nil
	}
	value := *after - *before
	return &value
}

func formulaOf(a, b *provider.IndexSeries) string {
	if a != nil && a.Formula != "" {
		return a.Formula
	}
	if b != nil {
		return b.Formula
	}
	return ""
}

func printChange(printer interface {
	Field(string, string)
	Line(string, ...any)
}, result *changeResult) {
	printer.Field("Observation", result.Observation)
	printer.Field("Provider", result.Provider)
	printer.Field("Collection", result.Collection)
	printer.Field("Index", strings.ToUpper(result.Index))
	if result.Formula != "" {
		printer.Field("Formula", result.Formula)
	}
	if len(result.BBox) == 4 {
		printer.Field("Area", fmt.Sprintf("%g,%g,%g,%g", result.BBox[0], result.BBox[1], result.BBox[2], result.BBox[3]))
	}

	printWindow := func(label string, stats changeStats) {
		printer.Line("")
		printer.Line("%s", label)
		printer.Field("  Period", stats.Period)
		printer.Field("  Scenes", fmt.Sprintf("%d", stats.Scenes))
		if stats.SampleCount != nil {
			printer.Field("  Samples", fmt.Sprintf("%d", *stats.SampleCount))
		}
		if stats.Mean != nil {
			printer.Field("  Mean", fmt.Sprintf("%.3f", *stats.Mean))
		}
		if stats.P10 != nil || stats.P50 != nil || stats.P90 != nil {
			printer.Field("  p10/p50/p90", fmt.Sprintf("%s / %s / %s",
				formatOptFloat(stats.P10, 3), formatOptFloat(stats.P50, 3), formatOptFloat(stats.P90, 3)))
		}
	}
	printWindow("Before", result.Before)
	printWindow("After", result.After)

	if result.DeltaMean != nil || result.DeltaP50 != nil {
		printer.Line("")
		printer.Line("Change (After − Before)")
		if result.DeltaMean != nil {
			printer.Field("Mean", fmt.Sprintf("%+.3f", *result.DeltaMean))
		}
		if result.DeltaP50 != nil {
			printer.Field("Median (p50)", fmt.Sprintf("%+.3f", *result.DeltaP50))
		}
	}
	if result.Note != "" {
		printer.Line("")
		printer.Line("%s", result.Note)
	}
}

func formatOptFloat(value *float64, precision int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.*f", precision, *value)
}
