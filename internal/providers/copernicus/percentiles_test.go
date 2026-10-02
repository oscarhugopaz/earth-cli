package copernicus

import (
	"encoding/json"
	"testing"
)

func TestIndexStatsPercentiles(t *testing.T) {
	raw := `{
		"mean": 0.3,
		"min": -1.0,
		"max": 0.9,
		"sampleCount": 13300,
		"percentiles": {"10.0": 0.149, "50.0": 0.292, "90.0": 0.466}
	}`
	var stats indexStats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(stats.Percentiles) != 3 {
		t.Fatalf("percentiles = %+v", stats.Percentiles)
	}
	if v := stats.Percentiles["50.0"]; v < 0.291 || v > 0.293 {
		t.Fatalf("p50 = %v", v)
	}
}

func TestIndexStatsPercentilesMissing(t *testing.T) {
	var stats indexStats
	if err := json.Unmarshal([]byte(`{"mean":0.1}`), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if stats.Percentiles != nil {
		t.Fatalf("percentiles = %+v, want nil", stats.Percentiles)
	}
}

func TestIndexStatsPercentilesWithNaN(t *testing.T) {
	raw := `{"mean":0.1,"percentiles":{"10.0":"NaN","50.0":0.2}}`
	var stats indexStats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := stats.Percentiles["10.0"]; ok {
		t.Fatal("NaN percentile should be dropped")
	}
	if stats.Percentiles["50.0"] != 0.2 {
		t.Fatalf("p50 = %v", stats.Percentiles["50.0"])
	}
}
