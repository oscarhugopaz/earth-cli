package observation

// Thermal observations use a different data source and band contract than the
// Sentinel-2 reflectance indices: Sentinel-3 SLSTR Level-2 exposes a Land
// Surface Temperature product in Kelvin. These helpers keep that knowledge in
// one place, separate from the spectral index catalog.

const (
	// LSTCollection is the Copernicus STAC collection carrying Land Surface
	// Temperature (Sentinel-3 SLSTR Level-2 LST, non-time-critical).
	LSTCollection = "sentinel-3-sl-2-lst-ntc"
	// LSTSource is the human label for that source.
	LSTSource = "Sentinel-3 SLSTR Level-2 LST"

	// lstEvalscript reads the LST band and converts Kelvin to degrees Celsius.
	// dataMask excludes no-data pixels; unlike Sentinel-2 there is no SCL band.
	lstEvalscript = `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["LST", "dataMask"] }],
    output: [
      { id: "lst", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  var celsius = sample.LST - 273.15;
  return { lst: [celsius], dataMask: [sample.dataMask] };
}`

	lstFormula = "LST (K) - 273.15 → °C"
)

// LSTEvalscript returns the evalscript used to compute land surface
// temperature in degrees Celsius.
func LSTEvalscript() string { return lstEvalscript }

// LSTFormula returns a human description of the temperature computation.
func LSTFormula() string { return lstFormula }
