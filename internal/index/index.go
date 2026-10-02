// Package index defines spectral indices computed from Sentinel-2 bands.
//
// Each index declares the bands it needs, a human description and an
// evalscript that produces a single float output plus a dataMask. Definitions
// live here so they are provider-neutral and testable.
package index

import (
	"fmt"
	"sort"
	"strings"
)

// Definition describes a computable spectral index.
type Definition struct {
	// Name is the lowercase identifier used on the command line and in JSON.
	Name string
	// Title is a short human label.
	Title string
	// Description explains what the index measures.
	Description string
	// Unit is the output unit (usually "index").
	Unit string
	// Bands are the input bands requested from the collection.
	Bands []string
	// Formula is the human-readable expression.
	Formula string
	// OutputID is the evalscript output id used for statistics.
	OutputID string
	// Evalscript is a full Sentinel-2 evalscript implementing the index.
	Evalscript string
}

// dataMaskLine is shared by every evalscript: mark clouds, cirrus, cloud
// shadows and snow as invalid so they are excluded from statistics.
const dataMaskLine = `var valid = sample.dataMask === 1 && [3, 8, 9, 10, 11].indexOf(sample.SCL) === -1;`

func evalscript(bands []string, outputID, formula string) string {
	inputBands := append(append([]string{}, bands...), "SCL", "dataMask")
	quoted := make([]string, len(inputBands))
	for i, band := range inputBands {
		quoted[i] = `"` + band + `"`
	}
	// The stored Formula is human-readable (B08); the evalscript needs the
	// sample-prefixed form (sample.B08).
	expression := formula
	for _, band := range bands {
		expression = strings.ReplaceAll(expression, band, "sample."+band)
	}
	return fmt.Sprintf(`//VERSION=3
function setup() {
  return {
    input: [{ bands: [%s] }],
    output: [
      { id: "%s", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  %s
  var value = %s;
  return { "%s": [value], dataMask: [valid ? 1 : 0] };
}`,
		strings.Join(quoted, ", "),
		outputID,
		dataMaskLine,
		expression,
		outputID,
	)
}

var definitions = map[string]Definition{}

func register(def Definition) {
	def.Name = strings.ToLower(def.Name)
	if def.Unit == "" {
		def.Unit = "index"
	}
	if def.OutputID == "" {
		def.OutputID = def.Name
	}
	if def.Evalscript == "" {
		def.Evalscript = evalscript(def.Bands, def.OutputID, def.Formula)
	}
	definitions[def.Name] = def
}

func init() {
	register(Definition{
		Name:        "ndvi",
		Title:       "NDVI",
		Description: "Normalized Difference Vegetation Index; vegetation vigor and green cover.",
		Bands:       []string{"B04", "B08"},
		Formula:     "(B08 - B04) / (B08 + B04)",
	})

	register(Definition{
		Name:        "ndwi",
		Title:       "NDWI",
		Description: "Normalized Difference Water Index (McFeeters); open water detection.",
		Bands:       []string{"B03", "B08"},
		Formula:     "(B03 - B08) / (B03 + B08)",
	})

	register(Definition{
		Name:        "mndwi",
		Title:       "MNDWI",
		Description: "Modified Normalized Difference Water Index; water in urban and built-up areas.",
		Bands:       []string{"B03", "B11"},
		Formula:     "(B03 - B11) / (B03 + B11)",
	})

	register(Definition{
		Name:        "ndmi",
		Title:       "NDMI",
		Description: "Normalized Difference Moisture Index; vegetation and soil water content.",
		Bands:       []string{"B08", "B11"},
		Formula:     "(B08 - B11) / (B08 + B11)",
	})

	register(Definition{
		Name:        "nbr",
		Title:       "NBR",
		Description: "Normalized Burn Ratio; burned area severity (lower NBR means more severe burn).",
		Bands:       []string{"B08", "B12"},
		Formula:     "(B08 - B12) / (B08 + B12)",
	})

	register(Definition{
		Name:        "ndbi",
		Title:       "NDBI",
		Description: "Normalized Difference Built-up Index; impervious and built-up surfaces.",
		Bands:       []string{"B11", "B08"},
		Formula:     "(B11 - B08) / (B11 + B08)",
	})

	register(Definition{
		Name:        "evi",
		Title:       "EVI",
		Description: "Enhanced Vegetation Index; reduces atmospheric and canopy background noise.",
		Bands:       []string{"B02", "B04", "B08"},
		Formula:     "2.5 * (B08 - B04) / (B08 + 6 * B04 - 7.5 * B02 + 1)",
	})

	register(Definition{
		Name:        "savi",
		Title:       "SAVI",
		Description: "Soil Adjusted Vegetation Index; vegetation in areas with exposed soil.",
		Bands:       []string{"B04", "B08"},
		Formula:     "((B08 - B04) / (B08 + B04 + 0.5)) * (1 + 0.5)",
	})

	register(Definition{
		Name:        "ndre",
		Title:       "NDRE",
		Description: "Normalized Difference Red Edge; chlorophyll content in dense canopies.",
		Bands:       []string{"B05", "B08"},
		Formula:     "(B08 - B05) / (B08 + B05)",
	})

	register(Definition{
		Name:        "ndsi",
		Title:       "NDSI",
		Description: "Normalized Difference Snow Index; snow and ice extent.",
		Bands:       []string{"B03", "B11"},
		Formula:     "(B03 - B11) / (B03 + B11)",
	})

	register(Definition{
		Name:        "gndvi",
		Title:       "GNDVI",
		Description: "Green Normalized Difference Vegetation Index; chlorophyll-sensitive vigor.",
		Bands:       []string{"B03", "B08"},
		Formula:     "(B08 - B03) / (B08 + B03)",
	})

	register(Definition{
		Name:        "nbr2",
		Title:       "NBR2",
		Description: "Normalized Burn Ratio 2; burn severity using short-wave infrared bands.",
		Bands:       []string{"B11", "B12"},
		Formula:     "(B11 - B12) / (B11 + B12)",
	})
}

// Lookup returns the definition for a name.
func Lookup(name string) (Definition, bool) {
	def, ok := definitions[strings.ToLower(strings.TrimSpace(name))]
	return def, ok
}

// Names lists every index name in stable order.
func Names() []string {
	names := make([]string, 0, len(definitions))
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// All returns every definition in stable order.
func All() []Definition {
	names := Names()
	out := make([]Definition, 0, len(names))
	for _, name := range names {
		out = append(out, definitions[name])
	}
	return out
}

// UnknownError builds an actionable error for an unknown index name.
func UnknownError(name string) error {
	return fmt.Errorf("unknown index %q\n\nAvailable indices:\n  %s",
		name, strings.Join(Names(), "\n  "))
}
