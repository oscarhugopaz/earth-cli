# earth-cli TODO

Backlog of pending work. This is a living document, not a commitment.

Priority legend:

- **P0** — needed to make the current MVP trustworthy or to unblock real use.
- **P1** — high-value next features.
- **P2** — nice to have / longer term.

## Shipped in v0.1.0

- [x] Cobra CLI: `providers`, `collections`, `collection`, `search`, `observe`, `version`, `completion`.
- [x] Provider-neutral `Provider` interface and Earth Engine registry.
- [x] Copernicus CDSE STAC provider (anonymous, public catalog).
- [x] `--json` on every data-producing command; stdout/stderr split; predictable exit codes.
- [x] bbox and GeoJSON (Polygon/MultiPolygon/Feature/FeatureCollection) area input.
- [x] Relative (`--since`) and absolute (`--from`/`--to`) time windows.
- [x] XDG config file plus `EARTH_PROVIDER`, `EARTH_COPERNICUS_STAC_URL`, `NO_COLOR`.
- [x] Semantic `observe vegetation` resolver (resolves scenes, no fake NDVI).
- [x] Unit tests, `go vet`, `go build`, CI on Linux.
- [x] GoReleaser release workflow with checksums.
- [x] Automatic Homebrew tap update via `repository_dispatch`.
- [x] Homebrew install/test verified locally and in CI.

## Done after v0.1.0 (unreleased on main)

- [x] STAC search pagination: follows `next` links (POST or GET, respecting the
      server-provided body/token) until `--limit` is reached.
- [x] HTTP retries with exponential backoff and jitter for 408/429/5xx and
      network errors, honoring `Retry-After`.
- [x] Unit tests for pagination and retry behavior.

## Known limitations (v0.1.0)

- [x] **P0** Search returns a single STAC page; it does not follow `next` links yet. *(fixed on main)*
- [ ] **P0** `search` never computes/returns derived indices; `observe vegetation` reports scenes only.
- [x] **P0** No HTTP retry/backoff for transient errors (429/5xx) and no rate limiting. *(retries fixed on main; rate limiting still pending)*
- [ ] **P0** No authentication; only operations the public catalog allows anonymously.
- [ ] **P1** `collections --search` fetches the whole catalog (up to 5000) to filter client-side.
- [ ] **P1** GeoJSON areas are reduced to a bounding box; no true intersects-based search.
- [ ] **P1** `windows/amd64` binaries are published but not covered by Homebrew or docs.
- [ ] **P2** No caching, no request concurrency, no progress reporting.

## Observations

- [ ] **P0** Real vegetation indices (NDVI/NDWI/EVI) via Copernicus Sentinel Hub Statistical API or remote/optional raster processing. Requires auth.
- [ ] **P0** A band-math processor abstraction (`NDVI = (B08 - B04) / (B08 + B04)`) isolated from the resolver.
- [ ] **P1** `flood` observation.
- [ ] **P1** `fire` and `burnt-area` observations (Sentinel-2/3, CLMS burnt area).
- [ ] **P1** `surface-change` (two time windows, changed area).
- [ ] **P1** `temperature` (thermal / LST).
- [ ] **P2** `atmosphere` (aerosol, water vapour).
- [ ] **P1** Make observation→collection mapping per provider explicit and testable.
- [ ] **P2** Pluggable resolvers (register at runtime, not compile time).

## Providers

- [ ] **P1** NASA (LP DAAC / CMR STAC) provider.
- [ ] **P1** USGS Landsat provider.
- [ ] **P1** Microsoft Planetary Computer provider.
- [ ] **P1** Provider capability matrix (which observations/searches each provider supports) and honest errors when unsupported.
- [ ] **P1** Auth layer: API keys / OAuth client credentials, token refresh, secure storage.
- [ ] **P1** Copernicus OAuth (client credentials) for Sentinel Hub APIs.
- [ ] **P2** A "stac" provider type that can be configured with any STAC endpoint and no code.
- [ ] **P2** Provider health/status checks and `earth providers --json` capability details.

## Search and discovery

- [x] **P0** STAC search pagination (`next` links). *(done on main; automatic, no flag needed)*
- [x] **P0** Transient-error retry with exponential backoff and `Retry-After` handling. *(done on main)*
- [ ] **P1** STAC `fields` extension for heavy collections (e.g. sentinel-2-l2a) to cut payload size and latency.
- [ ] **P1** Client-side rate limiting / request budget.
- [ ] **P1** Sort/filter: `--sortby`, `--ids`, `--queryable` (CQL2 / query extension).
- [ ] **P1** `earth item <collection> <item-id>` (or `earth item --url`) to inspect one item.
- [ ] **P1** Show/download asset links (`--assets`, `earth item --download`).
- [ ] **P1** Field selection (`--fields`) to trim JSON payloads.
- [ ] **P2** True geometry intersects (`intersects` with GeoJSON) once providers support it.
- [ ] **P2** Antimeridian-crossing bbox support.
- [ ] **P2** Saved searches / named areas in config.

## Compare

- [ ] **P1** `earth compare --area ... --before ... --after ...` for two time windows.
- [ ] **P1** `earth compare --a <aoi> --b <aoi>` for two areas.
- [ ] **P2** Diffable JSON output and a concise human summary.

## Watch (persistent observations)

- [ ] **P2** Design a `Watch` resource: area + observation + provider + schedule + state/history + condition + action.
- [ ] **P2** `earth watch` CRUD (create/list/get/delete) against a local store first.
- [ ] **P2** Scheduler and evaluation loop (cron/interval) with change detection (`--change 15%`).
- [ ] **P2** Events/webhooks (`--webhook`), with delivery retries and signing.
- [ ] **P2** Deployment layer (potentially Plak) — must remain optional; `earth-cli` stays independently useful.

## Declarative configuration (Earth Observation as Code)

- [ ] **P2** Define `earth.yaml` (areas, observations, watches, quality, triggers, actions).
- [ ] **P2** `earth plan` (diff desired vs current).
- [ ] **P2** `earth apply` (idempotent resource reconciliation).
- [ ] Avoid becoming a Terraform clone; keep the primitive set small.

## CLI / UX

- [ ] **P1** `earth config` (show resolved config and file path; `--json`).
- [ ] **P1** `--verbose`/`--debug` diagnostics to stderr.
- [ ] **P2** Additional output formats (`--output yaml|csv`) where meaningful.
- [ ] **P2** JSON schema versioning / stability guarantee for machine output.
- [ ] **P2** Man pages generated from Cobra.
- [ ] **P2** Windows install instructions (`scoop`/zip) and shell completion docs.

## Configuration and secrets

- [ ] **P1** Profiles (multiple environments/providers).
- [ ] **P1** Secret storage conventions (env first, then config, never committed).
- [ ] **P2** `XDG_CACHE_HOME` cache for collections and queryables.

## Testing and quality

- [ ] **P0** Integration tests against the live Copernicus STAC API, explicitly enabled (e.g. `EARTH_INTEGRATION=1`), never in normal unit runs.
- [ ] **P1** Coverage reporting and a coverage threshold for `internal/`.
- [ ] **P1** Fuzz tests for bbox/GeoJSON/time-window parsers.
- [ ] **P1** `golangci-lint` in CI.
- [ ] **P1** Contract tests with recorded STAC fixtures for more providers.
- [ ] **P2** Benchmarks for parsing and normalization.

## Release / CI / Homebrew

- [ ] **P1** Add `earth-cli` to `homebrew-tap/audit.yml` now that the formula exists.
- [ ] **P2** Dependabot for Go modules and GitHub Actions.
- [ ] **P2** SBOM generation and artifact signing (cosign).
- [ ] **P2** macOS notarization/signing.
- [ ] **P2** Release notes curation (keep a `CHANGELOG.md` or rely on GoReleaser generated notes).

## Documentation

- [ ] **P1** `CONTRIBUTING.md`.
- [ ] **P1** Document the provider + resolver extension points with a worked example.
- [ ] **P1** Document JSON output shapes per command.
- [ ] **P2** `CODE_OF_CONDUCT.md`, issue/PR templates.
- [ ] **P2** Recipes (e.g. vineyard monitoring, deforestation, flood response).

## MCP integration

- [ ] **P2** Expose `earth` capabilities through an MCP server so agents can search and observe.
- [ ] **P2** Keep MCP optional and out of the core binary or behind a separate command.

## Definition of done (for new providers/resolvers/tests)

- [ ] Provider/resolver added without leaking catalog-specific logic into `internal/cli`.
- [ ] Normalized output is deterministic and covered by unit tests.
- [ ] `go test ./...`, `go vet ./...`, `go build ./...`, `gofmt -l` are clean.
- [ ] `--json` output documented and stable.
- [ ] Honest behavior: never fabricate derived metrics; report what was actually resolved.
- [ ] README/docs updated.
