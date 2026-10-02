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

Searches are paginated automatically against the STAC API and `--limit` caps
the total number of items returned. Transient API failures (429/5xx) are
retried with backoff, honoring `Retry-After`.

Inspect one item and its assets:

```console
$ earth item sentinel-2-l2a <item-id>
ID            S2C_MSIL2A_...
Collection    sentinel-2-l2a
Date          2026-09-29T14:37:41Z
Cloud cover   98.3%
Item URL      https://stac.dataspace.copernicus.eu/v1/collections/.../items/...

Assets (47)
NAME                TYPE             HREF
B04_10m             image/jp2        s3://eodata/...
...
```

`--json` includes full `asset_details` (href, type, title, roles).

### Observe

```bash
earth observe vegetation --area vineyard.geojson --since 90d
earth observe vegetation --bbox -70.8,-33.6,-70.4,-33.3 --since 90d
earth observe vegetation --area vineyard.geojson --since 90d --index ndmi
```

`observe` is the semantic layer. The name `vegetation` is resolved by the Earth
Engine to the appropriate provider collection (Sentinel-2 L2A on Copernicus)
and the appropriate observations. You do not need to know Sentinel-2 or
`B04`/`B08`.

Available observations (aliases in parentheses):

| Observation | Default index | Purpose |
| ----------- | ------------- | ------- |
| `vegetation` (`ndvi`) | NDVI | Vegetation vigor |
| `flood` (`water`) | MNDWI | Surface water extent |
| `burnt-area` (`fire`, `burn`, `burn-area`) | NBR | Burn severity |
| `moisture` | NDMI | Vegetation/soil moisture |
| `temperature` | LST (°C) | Land surface temperature (Sentinel-3 SLSTR) |

Most observations are Sentinel-2 reflectance indices. `temperature` is
different: it reads Land Surface Temperature from Sentinel-3 SLSTR Level-2 and
reports degrees Celsius (Kelvin from the product, offset by 273.15). It uses a
different collection and does not apply the Sentinel-2 scene classification
mask.

`--index <name>` selects which spectral index to compute (default `ndvi`). Run
`earth indices` to list them:

| Index | Measures | Bands |
| ----- | -------- | ----- |
| `ndvi` | Vegetation vigor | B04, B08 |
| `evi` | Enhanced vegetation (less atmospheric noise) | B02, B04, B08 |
| `savi` | Vegetation over exposed soil | B04, B08 |
| `ndre` | Chlorophyll in dense canopies | B05, B08 |
| `ndmi` | Vegetation/soil moisture | B08, B11 |
| `ndwi` | Open water | B03, B08 |
| `mndwi` | Water in built-up areas | B03, B11 |
| `ndbi` | Built-up surfaces | B11, B08 |
| `nbr` | Burn severity | B08, B12 |

There are two honest modes:

- **Without credentials**, it resolves the source scenes and reports scene
  counts, cloud statistics and the best (least cloudy) scene, plus the bands
  and formula a processor uses. It does **not** fabricate an NDVI value.
- **With Copernicus Sentinel Hub OAuth credentials**, it additionally computes
  NDVI (10-day aggregates by default) through the Statistical API, masking
  clouds, shadows and snow via the Scene Classification Layer:

  ```console
  $ export EARTH_COPERNICUS_CLIENT_ID=...
  $ export EARTH_COPERNICUS_CLIENT_SECRET=...
  $ earth observe vegetation --bbox -70.8,-33.6,-70.4,-33.3 --since 90d
  ...
  Bands         B04, B08
  NDVI formula  NDVI = (B08 - B04) / (B08 + B04)

  NDVI series (sentinel-2-l2a, P10D)
  FROM        TO          MEAN   MIN    MAX    SAMPLES
  2026-07-01  2026-07-11  0.610  0.420  0.780  8123
  ...
  ```

Create the OAuth client in the Sentinel Hub Services dashboard (User Settings →
OAuth clients → Client Credentials). The secret is shown only once; keep it in
the environment, never in the repository.

You can store the credentials in `~/.config/earth/config.yaml` instead of
exporting them every time (see [Configuration](#configuration)).

`observe` accepts cost-related controls for the derived index:

| Flag | Default | Purpose |
| ---- | ------- | ------- |
| `--resolution <m>` | `10` | Ground sample distance; coarser costs less |
| `--interval <P..>` | `P10D` | ISO8601 aggregation (for example `P10D`, `P30D`) |
| `--dry-run` | off | Plan the request and estimate cost without spending quota |

```console
$ earth observe vegetation --area vineyard.geojson --since 90d --dry-run --resolution 20 --interval P30D
...
NDVI plan (dry run)
Index         NDVI
Collection    sentinel-2-l2a
Resolution    20m
Interval      P30D
Bands         B04, B08, SCL
Estimated cost ~2.24 PU (±25%)
```

Processing-unit estimates follow the Sentinel Hub model; the number of
acquisitions is assumed from a ~5-day revisit, so the estimate is a range, not
a promise. Your free CDSE account includes 10,000 PU/month.

> NDVI uses the Sentinel Hub **Statistical API** at
> `https://sh.dataspace.copernicus.eu/statistics/v1`. CDSE migrated API paths
> away from `/api/v1/...` and retired the old `statistics.dataspace.copernicus.eu`
> host; if that ever changes again, override it with
> `EARTH_COPERNICUS_STATISTICS_URL`.

### Compare

Compare two points in time over one area, or two areas over one window:

```console
$ earth compare --collection sentinel-2-l2a \
    --bbox -70.8,-33.6,-70.4,-33.3 --since 90d --against-since 180d \
    --index ndvi --interval P30D --resolution 20

Collection    sentinel-2-l2a
Provider      copernicus
Index         NDVI

Side A
  Period      2026-07-04 to 2026-10-02
  Area        -70.8,-33.6,-70.4,-33.3
  Scenes      45
  Mean cloud  68.3%
  Mean index  0.154

Side B
  Period      2026-04-05 to 2026-10-02
  Scenes      60
  Mean index  0.177

Difference (B − A)
Absolute      +0.023
Relative      +14.7%
```

`compare` reuses the index engine, so it needs credentials to produce the mean
index; without them it still reports scenes and cloud cover per side.

### Change

Measure change over one area between two time windows, using the distribution
(p10/p50/p90) rather than only the mean, so localized change is visible even
when the average barely moves:

```console
$ earth change --observation vegetation --area vineyard.geojson \
    --before-from 2026-05-01 --before-to 2026-06-01 \
    --after-from 2026-08-01 --after-to 2026-09-01

Observation   vegetation
Collection    sentinel-2-l2a
Index         NDVI

Before
  Period      2026-05-01 to 2026-06-01
  Scenes      9
  Mean        0.306
  p10/p50/p90 0.151 / 0.281 / 0.516

After
  Period      2026-08-01 to 2026-09-01
  Scenes      8
  Mean        0.444
  p10/p50/p90 0.144 / 0.446 / 0.716

Change (After − Before)
Mean          +0.138
Median (p50)  +0.165
```

`--observation` accepts the same names as `observe` (`vegetation`, `flood`,
`burnt-area`, `moisture`). `--dry-run` counts scenes for both windows without
spending quota.

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

### Configuration check

```console
$ earth config
Default provider copernicus
Config file   /home/oscar/.config/earth/config.yaml
Config found  yes

Provider      copernicus
STAC URL      https://stac.dataspace.copernicus.eu/v1
Client ID     …4616
Credentials   yes
```

Secrets are never printed: the client secret is reported only as a boolean.

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
    # Optional: enable authenticated index computation (NDVI).
    # Prefer the environment variables below for secrets.
    # client_id: ...
    # client_secret: ...
```

Environment variables override the file:

| Variable                            | Purpose                                        |
| ----------------------------------- | ---------------------------------------------- |
| `EARTH_PROVIDER`                    | Default provider                               |
| `EARTH_COPERNICUS_STAC_URL`         | Override the Copernicus STAC endpoint           |
| `EARTH_COPERNICUS_CLIENT_ID`        | Copernicus Sentinel Hub OAuth client id         |
| `EARTH_COPERNICUS_CLIENT_SECRET`    | Copernicus Sentinel Hub OAuth client secret     |
| `EARTH_COPERNICUS_TOKEN_URL`        | Override the OAuth token endpoint (advanced)    |
| `EARTH_COPERNICUS_STATISTICS_URL`   | Override the Statistical API endpoint (advanced)|
| `NO_COLOR`                          | Disable ANSI colors                             |

Secrets are read from the environment first and are never written to the cache
or to the repository. The OAuth token is kept in memory for the lifetime of the
process.

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
make test-integration   # opt-in live Copernicus STAC tests
make vet        # go vet ./...
make check      # test + vet + build
make snapshot   # GoReleaser snapshot build
```

Or directly:

```bash
go run ./cmd/earth --help
go test ./...
EARTH_INTEGRATION=1 go test ./internal/providers/copernicus/
```

Normal unit tests never touch the network. The integration tests are explicitly
opt-in via `EARTH_INTEGRATION=1` and only hit the public Copernicus STAC API
with small queries.

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
- additional vegetation indices and remote raster processing (NDVI is already
  available with Copernicus Sentinel Hub credentials);
- `earth compare` to diff two areas or two points in time;
- `earth watch` as a persistent Earth observation resource (area + observation
  + provider + schedule + condition + action), with events and webhooks;
- declarative Earth observation as code (`earth.yaml`, `earth plan`,
  `earth apply`);
- MCP integration so agents can use Earth observation as a tool.

`earth-cli` is independently useful; none of these require any external runtime.

See [TODO.md](TODO.md) for the concrete, prioritized backlog.

## License

Apache-2.0. See [LICENSE](LICENSE).
