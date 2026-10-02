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

// soilMoistureEvalscript reads the CLMS Surface Soil Moisture band, which is
// already expressed as percent saturation.
func soilMoistureEvalscript() string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["SSM", "dataMask"] }],
    output: [
      { id: "ssm", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  return { ssm: [sample.SSM], dataMask: [sample.dataMask] };
}`
}

// landCoverEvalscript reads the CLMS discrete land cover classification.
// Percentiles of a categorical raster give the dominant class.
func landCoverEvalscript() string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["Discrete_Classification", "dataMask"] }],
    output: [
      { id: "class", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  return { class: [sample.Discrete_Classification], dataMask: [sample.dataMask] };
}`
}

// landCoverClasses maps CLMS land cover codes to readable names.
// Source: CLMS Global Land Cover 100 m (v3) class legend.
func landCoverClasses() map[int]string {
	return map[int]string{
		0:   "No data",
		20:  "Shrubs",
		30:  "Herbaceous vegetation",
		40:  "Cropland",
		50:  "Urban / built-up",
		60:  "Bare / sparse vegetation",
		70:  "Snow and ice",
		80:  "Permanent water bodies",
		90:  "Herbaceous wetland",
		100: "Moss and lichen",
		111: "Closed forest, evergreen needle leaf",
		112: "Closed forest, evergreen broad leaf",
		113: "Closed forest, deciduous needle leaf",
		114: "Closed forest, deciduous broad leaf",
		115: "Closed forest, mixed",
		116: "Closed forest, unknown",
		121: "Open forest, evergreen needle leaf",
		122: "Open forest, evergreen broad leaf",
		123: "Open forest, deciduous needle leaf",
		124: "Open forest, deciduous broad leaf",
		125: "Open forest, mixed",
		126: "Open forest, unknown",
		200: "Open sea",
	}
}

// waterQualityEvalscript reads the CLMS trophic state index (TSI), a measure
// of lake eutrophication.
func waterQualityEvalscript() string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["TSI", "dataMask"] }],
    output: [
      { id: "tsi", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  return { tsi: [sample.TSI], dataMask: [sample.dataMask] };
}`
}

// chlorophyllEvalscript reads the Sentinel-3 OLCI chlorophyll-a band computed
// with the OC4Me algorithm.
func chlorophyllEvalscript() string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["CHL_OC4ME", "dataMask"] }],
    output: [
      { id: "chl", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  return { chl: [sample.CHL_OC4ME], dataMask: [sample.dataMask] };
}`
}

// aerosolEvalscript reads the Sentinel-5P Absorbing Aerosol Index band.
func aerosolEvalscript() string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["AER_AI_340_380", "dataMask"] }],
    output: [
      { id: "ai", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  return { ai: [sample.AER_AI_340_380], dataMask: [sample.dataMask] };
}`
}

// waterTemperatureEvalscript reads the CLMS Lake Surface Water Temperature.
// The stored value is in hundredths of Kelvin, so it is scaled and converted
// to degrees Celsius.
func waterTemperatureEvalscript() string {
	return `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["LSWT", "dataMask"] }],
    output: [
      { id: "lswt", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  var celsius = sample.LSWT * 0.01 + 273.15 - 273.15;
  return { lswt: [celsius], dataMask: [sample.dataMask] };
}`
}
