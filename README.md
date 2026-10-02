# earth-cli

A developer-friendly CLI for programmable Earth observation.

`earth` turns Earth observation data into a normal developer primitive. Instead
of learning STAC, Sentinel product naming, satellite bands, GeoTIFF, OData, or
Sentinel Hub internals before you can do anything useful, you ask for what you
want and let the tool figure out the details.

```console
$ earth observe vegetation --bbox -70.8,-33.6,-70.4,-33.3 --since 90d
Observation   vegetation
Provider      copernicus
Source        Sentinel-2 Level-2A
Collection    sentinel-2-l2a
Period        last 90 days
Area          -70.8,-33.6,-70.4,-33.3
Scenes        45
Mean cloud    68.3%
Best scene    2026-09-14
Cloud cover   0.1%
Scene ID      S2B_MSIL2A_20260914T143739_N0512_R096_T19HCC_20260914T195453
Bands         B04, B08
NDVI          NDVI = (B08 - B04) / (B08 + B04)

earth resolves the source scenes but does not compute NDVI in this release: NDVI requires the Sentinel Hub Processing/Statistical APIs or local raster processing, which are not part of the MVP.
```

## What is Earth observation?

Earth observation (EO) is the collection of measurements about the planet taken
by satellites and other remote sensors. A single satellite pass produces
*scenes*: images in several spectral bands (visible, near-infrared, ...) plus
metadata such as acquisition time and cloud cover.

Almost every EO catalog today speaks [STAC](https://stacspec.org/), the
SpatioTemporal Asset Catalog. STAC is a great interchange format but a poor
user interface: to answer "is the vineyard ok?" you first need to know which
collection to use, which bands matter, and how to combine them.

`earth` sits on top of STAC and gives you both:

- low-level, scriptable access to catalogs (`search`, `collections`), and
- high-level, semantic observations (`observe vegetation`).

## Providers

The first provider is the **Copernicus Data Space Ecosystem (CDSE)**, queried
through its public STAC API. Copernicus is a *provider*, not the core
architecture: the CLI talks to an internal Earth Engine that registers
providers behind a common interface. Additional providers (NASA, Landsat,
Microsoft Planetary Computer, or any STAC-compatible catalog) can be added
without redesigning the CLI.

```console
$ earth providers
NAME        TYPE  STATUS
copernicus  stac  available
```

## Install

```bash
brew install oscarhugopaz/tap/earth-cli
```

Then upgrade with:

```bash
brew update
brew upgrade earth-cli
```

Prebuilt archives are also published for macOS and Linux (arm64 and amd64), so
an alternative is to download the archive for your platform from the
[releases page](https://github.com/oscarhugopaz/earth-cli/releases) and put the
`earth` binary on your `PATH`.

## Usage

### Discover collections

```console
$ earth collections --search sentinel
ID                         TITLE
sentinel-1-grd             Sentinel-1 Ground Range Detected (GRD)
sentinel-2-l2a             Sentinel-2 Level-2A
...
```

Inspect a single collection:

```console
$ earth collection sentinel-2-l2a
ID            sentinel-2-l2a
Title         Sentinel-2 Level-2A
License       other
Keywords      Copernicus, Sentinel, EU, ESA, Satellite, Global, Imagery, Reflectance
Spatial       -180,-90,180,90
Temporal      2015-06-27T10:25:31Z .. present
Item URL      https://stac.dataspace.copernicus.eu/v1/collections/sentinel-2-l2a/items
Queryables    https://stac.dataspace.copernicus.eu/v1/collections/sentinel-2-l2a/queryables
```

### Search items

```console
$ earth search \
    --collection sentinel-2-l2a \
    --bbox -70.8,-33.6,-70.4,-33.3 \
    --from 2026-09-01 \
    --to 2026-10-01
DATE        COLLECTION      CLOUD  ID
2026-09-29  sentinel-2-l2a  98.3   S2C_MSIL2A_20260929T143741_N0513_R096_T19HCD_20260929T174811
...
```

Relative windows and GeoJSON areas are supported too:

```bash
earth search --collection sentinel-2-l2a --bbox -70.8,-33.6,-70.4,-33.3 --since 30d
earth search --collection sentinel-2-l2a --area vineyard.geojson --since 30d
```

`--area` accepts GeoJSON `Polygon`, `MultiPolygon`, `Feature`, and
`FeatureCollection`. For discovery, its bounding box is used; intersects-based
searching can be added later without changing the command line.

### Observe

```bash
earth observe vegetation --area vineyard.geojson --since 90d
earth observe vegetation --bbox -70.8,-33.6,-70.4,-33.3 --since 90d
```

`observe` is the semantic layer. The name `vegetation` is resolved by the Earth
Engine to the appropriate provider collection (Sentinel-2 L2A on Copernicus)
and the appropriate observations. You do not need to know Sentinel-2 or
`B04`/`B08`.

The MVP reports what it can truthfully resolve: scene counts, cloud statistics
and the best (least cloudy) scene, plus the bands and formula a future
processor will use. **It deliberately does not fabricate an NDVI value**, since
that requires the authenticated Sentinel Hub Processing/Statistical APIs or
local raster processing.

### Machine-readable output

Every data-producing command supports `--json`, on stdout, without styling:

```bash
earth collections --json | jq '.[].id'
earth search --collection sentinel-2-l2a --bbox -70.8,-33.6,-70.4,-33.3 --since 30d --json | jq '.[0].id'
earth observe vegetation --area vineyard.geojson --since 90d --json | jq '.best_scene'
earth providers --json
earth version --json
```

### Version

```console
$ earth version
earth version 0.1.0
```

`earth version --json` also reports the commit and build date; those are
injected by the release process rather than hard-coded.

### Shell completions

```bash
earth completion bash
earth completion zsh
earth completion fish
```

Homebrew installs completions automatically.

## Configuration

`earth` works with no configuration file. Optional configuration lives at:

```
$XDG_CONFIG_HOME/earth/config.yaml
# or
~/.config/earth/config.yaml
```

```yaml
default_provider: copernicus

providers:
  copernicus:
    stac_url: https://stac.dataspace.copernicus.eu/v1
```

Environment variables override the file:

| Variable                      | Purpose                                  |
| ----------------------------- | ---------------------------------------- |
| `EARTH_PROVIDER`              | Default provider                         |
| `EARTH_COPERNICUS_STAC_URL`   | Override the Copernicus STAC endpoint     |
| `NO_COLOR`                    | Disable ANSI colors                       |

## Output philosophy

- Results go to **stdout**; diagnostics go to **stderr**.
- Human output is concise and pipe-friendly.
- Machine output is stable JSON, free of ANSI styling.
- Colors are disabled when stdout is not a terminal, when `--json` is used, or
  when `NO_COLOR` is set.
- Exit codes: `0` success, `1` operational error, `2` usage error.
- Errors are actionable, for example:

  ```
  earth: invalid bbox "bad": expected minLon,minLat,maxLon,maxLat
  earth: Copernicus STAC request failed: HTTP 503 Service Unavailable
  ```

## Architecture

```
CLI (cobra commands, rendering)
        |
        v
Earth Engine  (provider + observation registries)
        |
        +-- Provider interface
        |      |
        |      +-- Copernicus (STAC)
        |      +-- future providers (NASA, Planetary Computer, ...)
        |
        +-- Observation resolvers
               |
               +-- vegetation
               +-- future: flood, fire, burnt-area, surface-change, ...
```

Layout:

```
cmd/earth/                    entry point
internal/cli/                 cobra commands, time-window parsing, rendering
internal/engine/              provider + resolver wiring
internal/provider/            provider interface and normalized domain types
internal/providers/copernicus/ CDSE STAC client
internal/observation/         semantic resolvers (vegetation)
internal/geometry/            bbox and GeoJSON parsing
internal/config/              defaults, XDG file, env overrides
internal/output/              tables, JSON, color detection
```

The MVP intentionally avoids native geospatial dependencies (GDAL and friends)
so `brew install earth-cli` stays a lightweight standalone binary.

## Development

Requires Go 1.25 or newer.

```bash
make build      # build bin/earth
make test       # go test ./...
make vet        # go vet ./...
make check      # test + vet + build
make snapshot   # GoReleaser snapshot build
```

Or directly:

```bash
go run ./cmd/earth --help
go test ./...
```

### Releasing

Releases are automated. See [docs/RELEASING.md](docs/RELEASING.md). In short:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions then runs the tests, builds archives with GoReleaser, publishes
the GitHub Release with SHA256 checksums, and updates the Homebrew formula in
`oscarhugopaz/homebrew-tap`.

## Roadmap

The architecture is designed to grow, but nothing below is implemented yet and
the CLI does not claim otherwise:

- additional EO providers (NASA, Landsat, Microsoft Planetary Computer);
- more semantic observations (`flood`, `fire`, `burnt-area`,
  `surface-change`, `temperature`, `atmosphere`);
- real NDVI/vegetation indices through authenticated processing APIs or remote
  raster processing;
- `earth compare` to diff two areas or two points in time;
- `earth watch` as a persistent Earth observation resource (area + observation
  + provider + schedule + condition + action), with events and webhooks;
- declarative Earth observation as code (`earth.yaml`, `earth plan`,
  `earth apply`);
- MCP integration so agents can use Earth observation as a tool.

`earth-cli` is independently useful; none of these require any external runtime.

## License

Apache-2.0. See [LICENSE](LICENSE).
