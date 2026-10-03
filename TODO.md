# earth-cli TODO

Backlog of pending work. This is a living document, not a commitment.

Priority legend:

- **P0** — needed to make the current MVP trustworthy or to unblock real use.
- **P1** — high-value next features.
- **P2** — nice to have / longer term.

Status: **v0.6.0**.

## Released

### v0.1.0 — MVP
- [x] Cobra CLI: `providers`, `collections`, `collection`, `search`, `observe`, `version`, `completion`.
- [x] Provider-neutral `Provider` interface and Earth Engine registry.
- [x] Copernicus CDSE STAC provider (anonymous, public catalog).
- [x] `--json` on every data-producing command; stdout/stderr split; predictable exit codes.
- [x] bbox and GeoJSON (Polygon/MultiPolygon/Feature/FeatureCollection) area input.
- [x] Relative (`--since`) and absolute (`--from`/`--to`) time windows.
- [x] XDG config file plus `EARTH_PROVIDER`, `EARTH_COPERNICUS_STAC_URL`, `NO_COLOR`.
- [x] Semantic `observe vegetation` resolver (resolved scenes, no fabricated NDVI).
- [x] Unit tests, `go vet`, `go build`, CI on Linux.
- [x] GoReleaser release workflow with checksums.
- [x] Automatic Homebrew tap update via `repository_dispatch`.
- [x] Homebrew install/test verified locally and in CI.

### v0.2.0
- [x] STAC search pagination: follows `next` links (POST or GET, with the
      server-provided body/token) until `--limit` is reached.
- [x] HTTP retries with exponential backoff and jitter for 408/429/5xx and
      network errors, honoring `Retry-After`.
- [x] `earth item <collection> <id>` with full asset details (href, type, title,
      roles).
- [x] Opt-in integration tests against the live Copernicus STAC API
      (`EARTH_INTEGRATION=1`).
- [x] Real NDVI via the Sentinel Hub Statistical API with OAuth credentials;
      clouds/shadow/snow masked via the SCL band.
- [x] Copernicus OAuth client-credentials flow with in-memory token caching.
- [x] `observe` cost controls: `--resolution`, `--interval` and `--dry-run` with
      a processing-unit estimate.
- [x] `earth config` showing the resolved configuration with secrets redacted.

### v0.3.0
- [x] Twelve Sentinel-2 spectral indices (`ndvi`, `gndvi`, `evi`, `savi`,
      `ndre`, `ndmi`, `ndwi`, `mndwi`, `ndbi`, `ndsi`, `nbr`, `nbr2`) with a
      reusable catalog and `earth indices`.
- [x] `earth compare`: two time windows over one area, or two areas over one
      window, with an index delta when credentials are present.
- [x] `earth change`: change between two time windows using index distribution
      statistics (mean and p10/p50/p90).
- [x] Percentile statistics support in the index engine.
- [x] Pre-flight guard for the Statistical API 2500 px per-side output limit,
      with an actionable message and a suggested resolution.

### v0.4.0
- [x] `temperature` observation: Sentinel-3 SLSTR Level-2 land surface
      temperature in °C (custom evalscript; processing collection distinct from
      the discovery collection).
- [x] `flood` (MNDWI), `burnt-area` (NBR), `moisture` (NDMI), `snow` (NDSI),
      `urban` (NDBI), `crop` (GNDVI) observations.
- [x] `atmosphere`, `methane`, `ozone`, `carbon-monoxide`, `sulfur-dioxide`
      observations from Sentinel-5P trace gases.
- [x] `soil-moisture` observation: CLMS Surface Soil Moisture via BYOC.
- [x] `land-cover` observation: CLMS Global Land Cover, dominant class
      (categorical handling) with per-observation interval (P1Y).
- [x] Custom-evalscript support in the index engine.

### v0.5.0
- [x] `water-quality` observation: CLMS Lake Water Quality trophic state index.
- [x] `chlorophyll` observation: Sentinel-3 OLCI chlorophyll-a (OC4ME).
- [x] `aerosol` observation: Sentinel-5P Absorbing Aerosol Index.

### v0.6.0
- [x] `water-temperature` observation: CLMS Lake Surface Water Temperature in °C
      via BYOC (`LSWT * 0.01 + 273.15 → °C`).
- [x] HTTP(S) asset downloads, generic configurable STAC discovery, stderr
      diagnostics, and golangci-lint CI.
- [x] Bundled Earth agent skill and offline `earth skill install`: project/global,
      agent-specific and deduplicated `--all` destinations; explicit `--force`.

## Observations

Nineteen semantic observations are implemented and validated live; twelve
Sentinel-2 indices are available through `observe`/`earth indices`.

- [ ] **P1** Open-ocean `sea-surface-temperature`. The CLMS lake product is
      covered by `water-temperature`; a marine product/band from Sentinel-3
      could not be confirmed, so no sea-surface observation is claimed.
- [ ] **P1** Sentinel-3 `aerosol optical depth` (AOD). The Sentinel-5P aerosol
      index is implemented; the Sentinel-3 AOD band was not confirmed.
- [ ] **P1** `fire` / `burnt-area` from the CLMS Burnt Area product (currently
      derived from NBR rather than the dedicated CLMS layer).
- [ ] **P1** More trace gases / atmospheric products (HCHO is supported by the
      API but not yet exposed as an observation; water vapour, CO2).
- [ ] **P2** More CLMS bio-geophysical products (FAPAR, LAI, FCOVER, snow cover,
      evapotranspiration).
- [ ] **P2** Pluggable observations registered at runtime rather than compile
      time.
- [ ] **P2** Sentinel-1 SAR-based observations (soil moisture, flood).

## Providers

- [ ] **P1** NASA (LP DAAC / CMR STAC) provider.
- [ ] **P1** USGS Landsat provider.
- [ ] **P1** Microsoft Planetary Computer provider.
- [ ] **P1** Provider capability matrix (which observations/searches each
      provider supports) and honest errors when unsupported.
- [x] **P1** Copernicus OAuth (client credentials) with in-memory token cache.
- [ ] **P1** Persistent credential storage (keychain / 0600 file helper).
- [x] **P2** Generic public STAC API discovery configurable by named `stac_url`.
- [ ] **P2** Provider health/status checks and `providers --json` capabilities.

## Search and discovery

- [x] **P0** STAC search pagination.
- [x] **P0** Transient-error retry with backoff and `Retry-After`.
- [ ] **P1** STAC `fields` extension for heavy collections to cut payload/latency.
- [ ] **P1** Client-side rate limiting / request budget.
- [ ] **P1** Sort/filter: `--sortby`, `--ids`, `--queryable` (CQL2).
- [x] **P1** `earth item --download` for HTTP(S) assets.
- [ ] **P1** `earth item --url` variant; authenticated/S3 downloads.
- [ ] **P1** Field selection (`--fields`) to trim JSON payloads.
- [ ] **P1** `collections --search` currently fetches the whole catalog to filter
      client-side; use a server-side query when available.
- [ ] **P2** True geometry intersects (`intersects` with GeoJSON).
- [ ] **P2** Antimeridian-crossing bbox support.
- [ ] **P2** Saved searches / named areas in config.

## Compare and change

- [x] `compare` supports two time windows over one area and two areas over one
      window.
- [x] `change` reports mean and p10/p50/p90 between two windows.
- [ ] **P1** `change` for categorical products (currently tuned for continuous
      indices; land cover dominates via percentiles but change is not specific).
- [ ] **P2** Diffable JSON output and a concise human summary for `compare`.

## Watch (persistent observations)

Not implemented. Deliberately deferred.

- [ ] **P2** Design a `Watch` resource: area + observation + provider + schedule
      + state/history + condition + action.
- [ ] **P2** `earth watch` CRUD (create/list/get/delete) against a local store.
- [ ] **P2** Scheduler and evaluation loop (cron/interval) with change detection
      (`--change 15%`).
- [ ] **P2** Events/webhooks (`--webhook`), with delivery retries and signing.
- [ ] **P2** Deployment layer (potentially Plak) — must remain optional.

## Declarative configuration (Earth Observation as Code)

- [ ] **P2** Define `earth.yaml` (areas, observations, watches, quality,
      triggers, actions).
- [ ] **P2** `earth plan` (diff desired vs current).
- [ ] **P2** `earth apply` (idempotent reconciliation).

## CLI / UX

- [x] **P1** `earth config` (resolved config and file path; `--json`).
- [x] **P1** `--verbose`/`--debug` diagnostics to stderr.
- [ ] **P2** Additional output formats (`--output yaml|csv`) where meaningful.
- [ ] **P2** JSON schema versioning / stability guarantee for machine output.
- [ ] **P2** Man pages generated from Cobra.
- [ ] **P2** Windows install instructions and shell-completion docs.

## Configuration and secrets

- [ ] **P1** Profiles (multiple environments/providers).
- [x] **P1** Secret storage conventions (env first, then config, never committed).
- [ ] **P2** `XDG_CACHE_HOME` cache for collections and queryables.

## Testing and quality

- [x] **P0** Opt-in live integration tests (`EARTH_INTEGRATION=1`).
- [ ] **P1** Coverage reporting and a threshold for `internal/`.
- [ ] **P1** Fuzz tests for bbox/GeoJSON/time-window parsers.
- [x] **P1** `golangci-lint` in CI (`govet`, `ineffassign`, `staticcheck`).
- [ ] **P1** Contract tests with recorded fixtures for more providers.
- [ ] **P2** Benchmarks for parsing and normalization.

## Release / CI / Homebrew

- [x] Releases v0.1.0 → v0.6.0 published with Homebrew tap updates.
- [x] **P1** Add `earth-cli` to `homebrew-tap/audit.yml`.
- [ ] **P2** Dependabot for Go modules and GitHub Actions.
- [ ] **P2** SBOM generation and artifact signing (cosign).
- [ ] **P2** macOS notarization/signing.
- [ ] **P2** `CHANGELOG.md` or curated release notes.

## Documentation

- [ ] **P1** `CONTRIBUTING.md`.
- [ ] **P1** Document the provider + observation extension points with a worked
      example (adding an observation is a data change).
- [ ] **P1** Document JSON output shapes per command.
- [ ] **P2** `CODE_OF_CONDUCT.md`, issue/PR templates.
- [ ] **P2** Recipes (vineyard monitoring, deforestation, flood response).

## MCP integration

- [ ] **P2** Expose `earth` capabilities through an MCP server so agents can
      search and observe.
- [ ] **P2** Keep MCP optional and out of the core binary.

## Definition of done (for new providers/observations)

- [ ] Added without leaking catalog-specific logic into `internal/cli`.
- [ ] Normalized output is deterministic and covered by unit tests.
- [ ] `go test ./...`, `go vet ./...`, `go build ./...`, `gofmt -l` are clean.
- [ ] `--json` output documented and stable.
- [ ] Honest behavior: never fabricate derived metrics; report what was actually
      resolved and validated against the live API where possible.
- [ ] README/docs updated.
