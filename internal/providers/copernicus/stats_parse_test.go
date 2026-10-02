package copernicus

import (
	"encoding/json"
	"testing"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
)

// Sentinel Hub encodes stats as numbers normally, but uses "NaN"/"Infinity"
// strings when an interval has no valid samples. Both must decode safely.
func TestIndexStatsUnmarshal(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		wantMean    *float64
		wantSamples *int
	}{
		{"numbers", `{"mean":0.5,"min":0.1,"max":0.9,"stDev":0.12,"sampleCount":1234}`, ptr(0.5), ptrInt(1234)},
		{"string numbers", `{"mean":"0.42","min":"0.1","max":"0.8","sampleCount":"100"}`, ptr(0.42), ptrInt(100)},
		{"nan strings", `{"mean":"NaN","min":"NaN","max":"NaN","stDev":"NaN","sampleCount":"0"}`, nil, ptrInt(0)},
		{"mixed", `{"mean":0.3,"min":"NaN","sampleCount":7}`, ptr(0.3), ptrInt(7)},
		{"missing fields", `{}`, nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stats indexStats
			if err := json.Unmarshal([]byte(tc.raw), &stats); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			assertFloatPtr(t, "mean", stats.Mean, tc.wantMean)
			assertIntPtr(t, "sampleCount", stats.SampleCount, tc.wantSamples)
		})
	}
}

func TestMetresToDegrees(t *testing.T) {
	bbox := bboxPtr(0, 0, 1, 1)
	got := metresToDegrees(111320, bbox)
	if got < 0.99 || got > 1.01 {
		t.Fatalf("10m at equator ~1 degree, got %v", got)
	}

	// At higher latitude one degree of longitude covers less ground, so the
	// same ground distance in metres needs more degrees.
	high := bboxPtr(0, 59, 1, 60)
	if metresToDegrees(111320, high) <= got {
		t.Fatalf("expected larger degree size at high latitude")
	}

	// Nil bbox degrades to the equatorial approximation.
	if metresToDegrees(111320, nil) < 0.99 {
		t.Fatalf("nil bbox fallback failed")
	}
}

func ptr(v float64) *float64 { return &v }
func ptrInt(v int) *int      { return &v }

func bboxPtr(minLon, minLat, maxLon, maxLat float64) *geometry.BBox {
	return &geometry.BBox{MinLon: minLon, MinLat: minLat, MaxLon: maxLon, MaxLat: maxLat}
}

func assertFloatPtr(t *testing.T, name string, got, want *float64) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Fatalf("%s = %v, want %v", name, *got, *want)
	}
}

func assertIntPtr(t *testing.T, name string, got, want *int) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Fatalf("%s = %v, want %v", name, *got, *want)
	}
}
