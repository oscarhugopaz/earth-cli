package cli

import (
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestCollapseWeightedMeanAndPercentiles(t *testing.T) {
	series := &provider.IndexSeries{
		Index: "ndvi",
		Intervals: []provider.IndexInterval{
			{
				Mean:        floatPtr(0.2),
				SampleCount: intPtr(100),
				Min:         floatPtr(0.1),
				Max:         floatPtr(0.4),
				Percentiles: map[string]float64{"10.0": 0.1, "50.0": 0.2, "90.0": 0.4},
			},
			{
				Mean:        floatPtr(0.6),
				SampleCount: intPtr(300),
				Min:         floatPtr(0.05),
				Max:         floatPtr(0.9),
				Percentiles: map[string]float64{"10.0": 0.2, "50.0": 0.6, "90.0": 0.8},
			},
		},
	}

	stats := collapse(changeStats{}, series)
	if stats.Mean == nil || *stats.Mean != 0.5 {
		t.Fatalf("mean = %v, want 0.5", stats.Mean)
	}
	if stats.SampleCount == nil || *stats.SampleCount != 400 {
		t.Fatalf("sample count = %v", stats.SampleCount)
	}
	if stats.Min == nil || *stats.Min != 0.05 {
		t.Fatalf("min = %v", stats.Min)
	}
	if stats.Max == nil || *stats.Max != 0.9 {
		t.Fatalf("max = %v", stats.Max)
	}
	if stats.P10 == nil || *stats.P10 < 0.149 || *stats.P10 > 0.151 {
		t.Fatalf("p10 = %v, want 0.15", stats.P10)
	}
	if stats.P50 == nil || *stats.P50 < 0.399 || *stats.P50 > 0.401 {
		t.Fatalf("p50 = %v, want 0.4", stats.P50)
	}
}

func TestCollapseHandlesMissingStats(t *testing.T) {
	// Intervals without samples must not produce values.
	series := &provider.IndexSeries{
		Intervals: []provider.IndexInterval{{Mean: nil, Min: nil, Max: nil}},
	}
	stats := collapse(changeStats{Scenes: 9}, series)
	if stats.Mean != nil || stats.P50 != nil {
		t.Fatalf("stats = %+v, want no values", stats)
	}
	if stats.Scenes != 9 {
		t.Fatalf("base fields should be preserved: %+v", stats)
	}
	if collapse(changeStats{}, nil).Mean != nil {
		t.Fatal("nil series should produce no mean")
	}
}

func TestAveragePercentileKeySpelling(t *testing.T) {
	buckets := map[string][]float64{"10.0": {0.2, 0.4}}
	got := averagePercentile(buckets, "10")
	if got == nil || *got < 0.299 || *got > 0.301 {
		t.Fatalf("averagePercentile = %v, want ~0.3", got)
	}
	if averagePercentile(buckets, "90") != nil {
		t.Fatal("missing percentile should be nil")
	}
}

func TestDiff(t *testing.T) {
	if diff(nil, floatPtr(1)) != nil || diff(floatPtr(1), nil) != nil {
		t.Fatal("diff with a missing side should be nil")
	}
	got := diff(floatPtr(0.2), floatPtr(0.5))
	if got == nil || *got != 0.3 {
		t.Fatalf("diff = %v, want 0.3", got)
	}
}

func TestChangeCommandRegistered(t *testing.T) {
	code, stdout, _ := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !contains(stdout, "change") {
		t.Fatalf("help missing change:\n%s", stdout)
	}
}

func TestChangeRequiresArea(t *testing.T) {
	code, _, stderr := runCLI(t, "change",
		"--before-from", "2026-05-01", "--before-to", "2026-06-01",
		"--after-from", "2026-08-01", "--after-to", "2026-09-01")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !contains(stderr, "requires an area of interest") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestChangeRequiresFullWindows(t *testing.T) {
	code, _, stderr := runCLI(t, "change", "--bbox", "-70.8,-33.6,-70.4,-33.3",
		"--before-from", "2026-05-01", "--after-from", "2026-08-01")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !contains(stderr, "full window") {
		t.Fatalf("stderr = %q", stderr)
	}
}

var _ = time.Now
