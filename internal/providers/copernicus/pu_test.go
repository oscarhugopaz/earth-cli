package copernicus

import (
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestEstimatePU(t *testing.T) {
	// A tiny parcel whose pixel count is below the 512x512 denominator must be
	// clamped to the 0.01 PU area factor. ~200 m at 10 m is 20x20 px.
	small := &geometry.BBox{MinLon: -70.8000, MinLat: -33.6000, MaxLon: -70.7978, MaxLat: -33.5982}
	got := EstimatePU(PUInputs{BBox: small, ResolutionM: 10, Bands: 2, Samples: 1})
	// areaFactor clamps to 0.01 and the band factor is 2/3 -> 0.00667, which is
	// then raised to the Statistical API minimum of 0.01 PU.
	if got != 0.01 {
		t.Fatalf("small parcel PU = %v, want the 0.01 minimum", got)
	}

	// Larger area with more samples must cost strictly more.
	big := &geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	one := EstimatePU(PUInputs{BBox: big, ResolutionM: 10, Bands: 3, Samples: 1})
	thirty := EstimatePU(PUInputs{BBox: big, ResolutionM: 10, Bands: 3, Samples: 30})
	if thirty <= one {
		t.Fatalf("more samples should cost more: %v vs %v", thirty, one)
	}

	// Coarser resolution must cost less.
	coarse := EstimatePU(PUInputs{BBox: big, ResolutionM: 20, Bands: 3, Samples: 30})
	if coarse >= thirty {
		t.Fatalf("coarser resolution should cost less: %v vs %v", coarse, thirty)
	}

	// Degenerate inputs.
	if EstimatePU(PUInputs{ResolutionM: 10, Samples: 1}) != 0 {
		t.Fatal("nil bbox should estimate 0")
	}
	if EstimatePU(PUInputs{BBox: big, ResolutionM: 10, Samples: 0}) != 0 {
		t.Fatal("zero samples should estimate 0")
	}
}

func TestPlanIndex(t *testing.T) {
	p := New("")
	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	plan := p.PlanIndex(provider.IndexRequest{
		Collection: "sentinel-2-l2a",
		Index:      "ndvi",
		BBox:       &bbox,
		Start:      &start,
		End:        &end,
		Resolution: 20,
	})
	if plan.Index != "ndvi" || plan.Collection != "sentinel-2-l2a" {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.ResolutionM != 20 || plan.Interval != "P10D" {
		t.Fatalf("plan resolution/interval = %+v", plan)
	}
	if len(plan.Bands) != 3 {
		t.Fatalf("bands = %v", plan.Bands)
	}
	if plan.EstimatedPU <= 0 {
		t.Fatalf("estimated PU = %v", plan.EstimatedPU)
	}
}

func TestPlanIndexDefaults(t *testing.T) {
	p := New("")
	plan := p.PlanIndex(provider.IndexRequest{})
	if plan.Collection != "sentinel-2-l2a" || plan.Index != "ndvi" {
		t.Fatalf("plan defaults = %+v", plan)
	}
	if plan.ResolutionM != 10 || plan.Interval != "P10D" {
		t.Fatalf("plan resolution/interval = %+v", plan)
	}
	if plan.EstimatedPU != 0 {
		t.Fatalf("plan without window should not estimate PU: %v", plan.EstimatedPU)
	}
}
