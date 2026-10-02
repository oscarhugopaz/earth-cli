package cli

import (
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/observation"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }

func TestMeanIndexWeighted(t *testing.T) {
	series := &provider.IndexSeries{
		Index: "ndvi",
		Intervals: []provider.IndexInterval{
			{Mean: floatPtr(0.2), SampleCount: intPtr(100)},
			{Mean: nil, SampleCount: intPtr(100)}, // no samples: skipped
			{Mean: floatPtr(0.6), SampleCount: intPtr(300)},
		},
	}
	mean := meanIndex(series)
	if mean == nil {
		t.Fatal("meanIndex returned nil")
	}
	// (0.2*100 + 0.6*300) / 400 = 0.5
	if *mean != 0.5 {
		t.Fatalf("mean = %v, want 0.5", *mean)
	}
}

func TestMeanIndexEmpty(t *testing.T) {
	if meanIndex(nil) != nil {
		t.Fatal("nil series should have no mean")
	}
	if meanIndex(&provider.IndexSeries{}) != nil {
		t.Fatal("series without intervals should have no mean")
	}
}

func TestComputeDelta(t *testing.T) {
	a := &provider.IndexSeries{Index: "ndvi", Intervals: []provider.IndexInterval{{Mean: floatPtr(0.2)}}}
	b := &provider.IndexSeries{Index: "ndvi", Intervals: []provider.IndexInterval{{Mean: floatPtr(0.3)}}}

	delta := computeDelta(a, b)
	if delta == nil {
		t.Fatal("delta is nil")
	}
	if delta.Absolute == nil || *delta.Absolute < 0.0999 || *delta.Absolute > 0.1001 {
		t.Fatalf("absolute = %v", delta.Absolute)
	}
	if delta.Relative == nil || *delta.Relative < 0.499 || *delta.Relative > 0.501 {
		t.Fatalf("relative = %v", delta.Relative)
	}
	if delta.Index != "ndvi" {
		t.Fatalf("index = %q", delta.Index)
	}
}

func TestComputeDeltaMissingSides(t *testing.T) {
	a := &provider.IndexSeries{Index: "ndvi", Intervals: []provider.IndexInterval{{Mean: floatPtr(0.2)}}}
	if computeDelta(a, nil) != nil {
		t.Fatal("delta with a missing side should be nil")
	}
	if computeDelta(nil, nil) != nil {
		t.Fatal("delta with both sides missing should be nil")
	}
}

func TestComputeDeltaZeroBaseline(t *testing.T) {
	a := &provider.IndexSeries{Index: "ndvi", Intervals: []provider.IndexInterval{{Mean: floatPtr(0)}}}
	b := &provider.IndexSeries{Index: "ndvi", Intervals: []provider.IndexInterval{{Mean: floatPtr(0.5)}}}

	delta := computeDelta(a, b)
	if delta == nil || delta.Absolute == nil {
		t.Fatal("delta should exist")
	}
	if delta.Relative != nil {
		t.Fatal("relative change is undefined for a zero baseline")
	}
}

func TestBuildSideRejectsBothAreaAndBBox(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	_, err := buildSide("A", "-70,−33,-69,-32", "area.geojson", "", "", "", now)
	if err == nil {
		t.Fatal("expected an error when both bbox and area are given")
	}
}

func TestBuildSideWindow(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	side, err := buildSide("A", "", "", "", "", "30d", now)
	if err != nil {
		t.Fatalf("buildSide: %v", err)
	}
	if side.Start == nil || side.End == nil {
		t.Fatalf("side = %+v", side)
	}
}

func TestDescribeWindow(t *testing.T) {
	start := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	if got := describeWindow(compareSide{Start: &start, End: &end}); got != "2026-07-04 to 2026-10-02" {
		t.Fatalf("describeWindow = %q", got)
	}
	if got := describeWindow(compareSide{Start: &start}); got != "since 2026-07-04" {
		t.Fatalf("describeWindow = %q", got)
	}
	if got := describeWindow(compareSide{}); got != "all time" {
		t.Fatalf("describeWindow = %q", got)
	}
}

// The compare command must appear in the root help.
func TestCompareCommandRegistered(t *testing.T) {
	code, stdout, _ := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !contains(stdout, "compare") {
		t.Fatalf("help missing compare:\n%s", stdout)
	}
	if !contains(stdout, "indices") {
		t.Fatalf("help missing indices:\n%s", stdout)
	}
}

func TestCompareRequiresCollection(t *testing.T) {
	code, _, stderr := runCLI(t, "compare", "--bbox", "-70.8,-33.6,-70.4,-33.3", "--since", "30d")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !contains(stderr, "--collection is required") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestCompareRequiresTimeWindow(t *testing.T) {
	code, _, stderr := runCLI(t, "compare", "--collection", "sentinel-2-l2a", "--bbox", "-70.8,-33.6,-70.4,-33.3")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !contains(stderr, "time window") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

var _ = observation.Result{}
