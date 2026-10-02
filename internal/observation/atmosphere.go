package observation

// Atmospheric observations use Sentinel-5P Level-2 trace-gas products. Like
// temperature, they need a collection and a band that differ from the
// Sentinel-2 reflectance indices, so the knowledge lives here.

const (
	// AtmosphereCollection is the Copernicus STAC collection for Sentinel-5P
	// NO2 (the default atmospheric product).
	AtmosphereCollection = "sentinel-5p-l2-no2"
	// AtmosphereProcessingCollection is the Sentinel-5P data collection id
	// used by Sentinel Hub.
	AtmosphereProcessingCollection = "sentinel-5p-l2"
	// AtmosphereSource is the human label for that source.
	AtmosphereSource = "Sentinel-5P Level-2"
)

// atmosphereEvalscript reads a trace-gas band and returns it as a float. The
// band name is injected because each gas is a different band in the same
// Sentinel-5P collection.
func atmosphereEvalscript(band string) string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["` + band + `", "dataMask"] }],
    output: [
      { id: "gas", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  return { gas: [sample.` + band + `], dataMask: [sample.dataMask] };
}`
}

// atmosphereFormula names the product behind the default atmosphere band.
const atmosphereFormula = "Sentinel-5P L2 NO2 vertical column"

// AtmosphereEvalscript returns the evalscript for the default atmosphere band.
func AtmosphereEvalscript() string { return atmosphereEvalscript("NO2") }

// AtmosphereFormula returns a human description of the atmosphere computation.
func AtmosphereFormula() string { return atmosphereFormula }

// AtmosphereBandEvalscript returns an evalscript for a specific trace-gas
// band (NO2, O3, SO2, CO, HCHO).
func AtmosphereBandEvalscript(band string) string { return atmosphereEvalscript(band) }
