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

// compareSide is one side of a comparison.
type compareSide struct {
	Label          string                `json:"label"`
	BBox           []float64             `json:"bbox,omitempty"`
	Start          *time.Time            `json:"start,omitempty"`
	End            *time.Time            `json:"end,omitempty"`
	Scenes         int                   `json:"scenes"`
	MeanCloudCover *float64              `json:"mean_cloud_cover,omitempty"`
	BestScene      *provider.Observation `json:"best_scene,omitempty"`
	Index          *provider.IndexSeries `json:"index,omitempty"`
}

// compareResult is the JSON/human shape for `earth compare`.
type compareResult struct {
	Collection string      `json:"collection"`
	Provider   string      `json:"provider"`
	Index      string      `json:"index,omitempty"`
	A          compareSide `json:"a"`
	B          compareSide `json:"b"`
	Delta      *indexDelta `json:"index_delta,omitempty"`
}

// indexDelta is the difference between the two sides' mean index.
type indexDelta struct {
	Index    string   `json:"index"`
	MeanA    *float64 `json:"mean_a,omitempty"`
	MeanB    *float64 `json:"mean_b,omitempty"`
	Absolute *float64 `json:"absolute,omitempty"`
	Relative *float64 `json:"relative,omitempty"`
}

func newCompareCommand(env Environment) *cobra.Command {
	var (
		collection string
		// Side A
		areaA  string
		bboxA  string
		fromA  string
		toA    string
		sinceA string
		// Side B
		areaB  string
		bboxB  string
		fromB  string
		toB    string
		sinceB string

		limit      int
		resolution float64
		interval   string
		indexName  string
	)

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare two areas or two time windows",
		Long: `Compare two observations.

Compare two points in time (same area, two windows):

  earth compare --collection sentinel-2-l2a --bbox -70.8,-33.6,-70.4,-33.3 \
    --since 30d --against-since 180d

Compare two areas (same window):

  earth compare --collection sentinel-2-l2a \
    --area north.geojson --against-bbox -70.8,-33.6,-70.4,-33.3 --since 90d

When credentials are configured and a time window is present, a spectral index
(default NDVI) is computed for each side and the difference is reported.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(collection) == "" {
				return apperr.Usage("--collection is required")
			}

			now := nowUTC()
			sideA, err := buildSide("A", bboxA, areaA, fromA, toA, sinceA, now)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			sideB, err := buildSide("B", bboxB, areaB, fromB, toB, sinceB, now)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			if sideA.BBox == nil && sideB.BBox == nil {
				return apperr.Usage("provide an area for at least one side (--bbox/--area or --against-bbox/--against-area)")
			}
			// Fill in what each side did not specify. This supports both
			// one-area/two-windows and two-areas/one-window comparisons.
			if sideB.BBox == nil {
				sideB.BBox = sideA.BBox
			}
			if sideA.BBox == nil {
				sideA.BBox = sideB.BBox
			}
			if sideB.Start == nil && sideB.End == nil {
				sideB.Start, sideB.End = sideA.Start, sideA.End
			}
			if sideA.Start == nil && sideA.End == nil {
				sideA.Start, sideA.End = sideB.Start, sideB.End
			}
			if sideA.Start == nil && sideA.End == nil && sideB.Start == nil && sideB.End == nil {
				return apperr.Usage("provide a time window for at least one side (--since/--from/--to)")
			}

			if limit <= 0 {
				limit = 200
			}

			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			result := &compareResult{
				Collection: collection,
				Provider:   p.Name(),
			}

			for _, side := range []*compareSide{&sideA, &sideB} {
				if err := runCompareSide(cmd, p, collection, side, limit, resolution, interval, indexName); err != nil {
					return err
				}
			}
			result.A = sideA
			result.B = sideB
			result.Index = indexOr(indexName, sideA.Index, sideB.Index)
			result.Delta = computeDelta(sideA.Index, sideB.Index)

			if session.json {
				return session.printer.JSONValue(result)
			}
			printCompare(session.printer, result)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&collection, "collection", "", "collection id to compare (required)")

	flags.StringVar(&bboxA, "bbox", "", "side A bounding box minLon,minLat,maxLon,maxLat")
	flags.StringVar(&areaA, "area", "", "side A GeoJSON file")
	flags.StringVar(&fromA, "from", "", "side A start date")
	flags.StringVar(&toA, "to", "", "side A end date")
	flags.StringVar(&sinceA, "since", "", "side A relative window, for example 30d")

	flags.StringVar(&bboxB, "against-bbox", "", "side B bounding box minLon,minLat,maxLon,maxLat")
	flags.StringVar(&areaB, "against-area", "", "side B GeoJSON file")
	flags.StringVar(&fromB, "against-from", "", "side B start date")
	flags.StringVar(&toB, "against-to", "", "side B end date")
	flags.StringVar(&sinceB, "against-since", "", "side B relative window, for example 180d")

	flags.IntVar(&limit, "limit", 200, "maximum number of scenes per side")
	flags.Float64Var(&resolution, "resolution", 10, "ground sample distance in metres for the index")
	flags.StringVar(&interval, "interval", "P10D", "ISO8601 aggregation interval for the index")
	flags.StringVar(&indexName, "index", "ndvi", "spectral index to compare")
	return cmd
}

func buildSide(label, bboxRaw, area, from, to, since string, now time.Time) (compareSide, error) {
	side := compareSide{Label: label}

	bbox, err := resolveArea(bboxRaw, area)
	if err != nil {
		return compareSide{}, err
	}
	if bbox != nil {
		side.BBox = bbox.Slice()
	}

	start, end, _, err := parseTimeWindow(from, to, since, now)
	if err != nil {
		return compareSide{}, err
	}
	side.Start = start
	side.End = end
	return side, nil
}

// identicalWindow reports whether two sides describe the same time window.
func identicalWindow(a, b compareSide) bool {
	return sameTime(a.Start, b.Start) && sameTime(a.End, b.End)
}

func sameTime(a, b *time.Time) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Equal(*b)
	}
}

func runCompareSide(cmd *cobra.Command, p provider.Provider, collection string, side *compareSide, limit int, resolution float64, interval, indexName string) error {
	var bbox *geometry.BBox
	if len(side.BBox) == 4 {
		bbox = &geometry.BBox{
			MinLon: side.BBox[0], MinLat: side.BBox[1],
			MaxLon: side.BBox[2], MaxLat: side.BBox[3],
		}
	}

	observations, err := p.Search(cmd.Context(), provider.SearchRequest{
		Collection: collection,
		BBox:       bbox,
		Start:      side.Start,
		End:        side.End,
		Limit:      limit,
	})
	if err != nil {
		return err
	}
	side.Scenes = len(observations)
	side.MeanCloudCover = meanCloudCover(observations)
	side.BestScene = bestObservation(observations)

	if bbox != nil && side.Start != nil && side.End != nil {
		if indexer, ok := p.(provider.IndexProvider); ok && indexer.SupportsIndex() && strings.TrimSpace(indexName) != "" {
			series, err := indexer.IndexSeries(cmd.Context(), provider.IndexRequest{
				Collection: collection,
				Index:      indexName,
				BBox:       bbox,
				Start:      side.Start,
				End:        side.End,
				Interval:   interval,
				Resolution: resolution,
			})
			if err != nil {
				return err
			}
			side.Index = &series
		}
	}
	return nil
}

func meanCloudCover(observations []provider.Observation) *float64 {
	var sum float64
	var count int
	for _, observation := range observations {
		if observation.CloudCover != nil {
			sum += *observation.CloudCover
			count++
		}
	}
	if count == 0 {
		return nil
	}
	mean := sum / float64(count)
	return &mean
}

func bestObservation(observations []provider.Observation) *provider.Observation {
	var best *provider.Observation
	for i := range observations {
		candidate := &observations[i]
		if best == nil || lessCloudyOrNewer(candidate, best) {
			best = candidate
		}
	}
	return best
}

func lessCloudyOrNewer(candidate, current *provider.Observation) bool {
	if candidate.CloudCover != nil && current.CloudCover != nil {
		if *candidate.CloudCover != *current.CloudCover {
			return *candidate.CloudCover < *current.CloudCover
		}
	} else if candidate.CloudCover != nil {
		return true
	} else if current.CloudCover != nil {
		return false
	}
	if candidate.DateTime != nil && current.DateTime != nil {
		return candidate.DateTime.After(*current.DateTime)
	}
	return false
}

// meanIndex returns the sample-weighted mean of an index across intervals.
func meanIndex(series *provider.IndexSeries) *float64 {
	if series == nil {
		return nil
	}
	var weightedSum float64
	var weight float64
	for _, interval := range series.Intervals {
		if interval.Mean == nil {
			continue
		}
		w := 1.0
		if interval.SampleCount != nil && *interval.SampleCount > 0 {
			w = float64(*interval.SampleCount)
		}
		weightedSum += *interval.Mean * w
		weight += w
	}
	if weight == 0 {
		return nil
	}
	mean := weightedSum / weight
	return &mean
}

func computeDelta(a, b *provider.IndexSeries) *indexDelta {
	meanA := meanIndex(a)
	meanB := meanIndex(b)
	if meanA == nil || meanB == nil {
		return nil
	}
	delta := &indexDelta{
		Index: indexOr("", a, b),
		MeanA: meanA,
		MeanB: meanB,
	}
	absolute := *meanB - *meanA
	delta.Absolute = &absolute
	if *meanA != 0 {
		relative := absolute / *meanA
		delta.Relative = &relative
	}
	return delta
}

func indexOr(explicit string, a, b *provider.IndexSeries) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.ToLower(strings.TrimSpace(explicit))
	}
	if a != nil && a.Index != "" {
		return a.Index
	}
	if b != nil && b.Index != "" {
		return b.Index
	}
	return ""
}

func printCompare(printer interface {
	Field(string, string)
	Line(string, ...any)
}, result *compareResult) {
	printer.Field("Collection", result.Collection)
	printer.Field("Provider", result.Provider)
	if result.Index != "" {
		printer.Field("Index", strings.ToUpper(result.Index))
	}

	for _, side := range []compareSide{result.A, result.B} {
		printer.Line("")
		printer.Line("Side %s", side.Label)
		printer.Field("  Period", describeWindow(side))
		if len(side.BBox) == 4 {
			printer.Field("  Area", fmt.Sprintf("%g,%g,%g,%g", side.BBox[0], side.BBox[1], side.BBox[2], side.BBox[3]))
		}
		printer.Field("  Scenes", fmt.Sprintf("%d", side.Scenes))
		if side.MeanCloudCover != nil {
			printer.Field("  Mean cloud", fmt.Sprintf("%.1f%%", *side.MeanCloudCover))
		}
		if mean := meanIndex(side.Index); mean != nil {
			printer.Field("  Mean index", fmt.Sprintf("%.3f", *mean))
		}
	}

	if result.Delta != nil {
		printer.Line("")
		printer.Line("Difference (B − A)")
		if result.Delta.Absolute != nil {
			printer.Field("Absolute", fmt.Sprintf("%+.3f", *result.Delta.Absolute))
		}
		if result.Delta.Relative != nil {
			printer.Field("Relative", fmt.Sprintf("%+.1f%%", *result.Delta.Relative*100))
		}
	}
}

func describeWindow(side compareSide) string {
	switch {
	case side.Start != nil && side.End != nil:
		return side.Start.UTC().Format(dateLayout) + " to " + side.End.UTC().Format(dateLayout)
	case side.Start != nil:
		return "since " + side.Start.UTC().Format(dateLayout)
	case side.End != nil:
		return "until " + side.End.UTC().Format(dateLayout)
	default:
		return "all time"
	}
}
