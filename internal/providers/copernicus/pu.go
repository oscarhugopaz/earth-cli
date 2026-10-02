package copernicus

import (
	"math"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
)

// Processing Unit (PU) model for the Sentinel Hub Statistical API.
//
// One PU is defined by Sentinel Hub as:
//   - an output of 512 x 512 pixels,
//   - 3 input bands,
//   - 1 data sample per pixel,
//   - <= 16 bits per pixel output,
//   - no extra processing.
//
// The cost of a request is the product of the applicable multiplication
// factors. For the Statistical API the number of data samples equals the
// number of acquisitions in the time window.
//
// Reference:
// https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/Overview/ProcessingUnit.html

// PUInputs describes everything needed to estimate a Statistical API request.
type PUInputs struct {
	BBox        *geometry.BBox
	ResolutionM float64
	Bands       int
	Samples     int // acquisitions in the time window
}

// PUError is the uncertainty of a PU estimate as a ratio (e.g. 0.25 = ±25%).
const PUError = 0.25

// EstimatePU returns a conservative estimate of processing units and the
// components used to derive it. The number of samples is the dominant term and
// is inherently uncertain, so callers should present the estimate as a range.
func EstimatePU(in PUInputs) float64 {
	if in.BBox == nil || in.Samples <= 0 {
		return 0
	}

	res := in.ResolutionM
	if res <= 0 {
		res = 10
	}

	// Area factor: requested pixels / (512 x 512), minimum 0.01.
	widthM := haversine(widthM(in.BBox.MinLon, in.BBox.MaxLon, (in.BBox.MinLat+in.BBox.MaxLat)/2))
	heightM := haversine((in.BBox.MaxLat - in.BBox.MinLat) * 111320)
	pixelsX := widthM / res
	pixelsY := heightM / res
	areaFactor := (pixelsX * pixelsY) / (512 * 512)
	if areaFactor < 0.01 {
		areaFactor = 0.01
	}

	// Band factor: dataMask does not count.
	bandFactor := float64(in.Bands) / 3
	if bandFactor <= 0 {
		bandFactor = 0.01
	}

	// Sample factor: one multiplication per acquisition.
	sampleFactor := float64(in.Samples)

	// Output format factor: 32-bit floats cost 2x.
	const formatFactor = 1.0

	pue := areaFactor * bandFactor * sampleFactor * formatFactor

	// Statistical API enforces a minimum request cost of 0.01 PU.
	if pue < 0.01 {
		pue = 0.01
	}
	return pue
}

// widthM converts a longitude span to metres at the given latitude.
func widthM(minLon, maxLon, latitude float64) float64 {
	return haversine((maxLon - minLon) * math.Cos(latitude*math.Pi/180) * 111320)
}

// haversine is a tiny identity wrapper kept for readability at call sites.
func haversine(metres float64) float64 {
	if metres < 0 {
		return -metres
	}
	return metres
}
