---
name: earth
description: Discover Earth-observation scenes and compute supported observations using the Earth CLI. Use when working with satellite imagery, Copernicus, STAC catalogs, vegetation indices, environmental monitoring, or comparisons of areas and time windows.
---

# Earth CLI

## Start here

1. Run `earth version` and `earth --help` to check the installed CLI.
2. Use `earth <command> --help` before assuming flags or capabilities.
3. Establish the area, time window, provider, and intended measurement.
4. Prefer `--json` for machine-readable results; diagnostics go to stderr.
5. If Earth is missing, ask the user to install it; do not install software without permission.

## Discover data (no processing credentials required)

```sh
earth providers --json
earth collections --limit 10 --json
earth collection sentinel-2-l2a --json
earth search --collection sentinel-2-l2a --bbox -70.8,-33.6,-70.4,-33.3 --since 30d --limit 5 --json
earth item <collection> <item-id> --json
```

- A bbox is `minLon,minLat,maxLon,maxLat`, in longitude/latitude degrees.
- Use either `--area area.geojson` or `--bbox`, not both. GeoJSON areas are currently reduced to bounding boxes, not exact polygon masks.
- Use `--since 30d` or explicit `--from` / `--to` dates, not both.
- `--limit` caps returned scenes. A successful query does not guarantee usable imagery.
- Generic configured STAC providers support discovery only. Select one with `--provider <name>`; do not assume Copernicus observation mappings or processing work elsewhere.

## Observe an area

```sh
earth indices --json
earth observe vegetation --area area.geojson --since 90d --dry-run --json
earth observe vegetation --area area.geojson --since 90d --resolution 20 --interval P10D --json
```

- `earth observe --help` describes usage; invoking it without a name lists supported observations.
- Without Copernicus processing credentials, observations may return source scenes only. Do not fabricate measurements from scene counts or thumbnails.
- Processing uses Copernicus OAuth client credentials. The user may store them in their private Earth config instead of exporting them every session.
- Never read, print, commit, or request the user's secret values. Use `earth config` to inspect the redacted effective configuration.
- A dry run plans processing without spending processing quota, but may still query the catalog. Its PU estimate is approximate, not a guarantee.
- Obtain approval before quota-consuming processing unless already authorized. Use coarser resolution or shorter windows to reduce cost.
- Respect the Statistical API output-size guard; follow its suggested coarser resolution rather than repeatedly retrying an oversized request.

## Compare and measure change

```sh
earth compare --collection sentinel-2-l2a --area area.geojson --from 2025-06-01 --to 2025-07-01 --against-from 2025-07-01 --against-to 2025-08-01 --json
earth compare --collection sentinel-2-l2a --area north.geojson --against-area south.geojson --since 30d --json
earth change --observation vegetation --area area.geojson --before-from 2025-06-01 --before-to 2025-07-01 --after-from 2025-07-01 --after-to 2025-08-01 --dry-run --json
```

Compare can spend processing quota when credentials and time windows are present.
Use comparable areas, seasons, resolution, and cloud conditions. Begin change workflows with a dry run.
Do not assume `change` handles every custom or categorical product correctly; check the actual plan and index mapping first.

## Download an asset

```sh
earth item <collection> <item-id> --download thumbnail --download-dir ./out --json
```

Inspect the item's named assets first. Downloads support HTTP(S), not S3 or provider-specific authentication/signing. Existing files are not overwritten. Confirm the destination before writing files.

## Interpret honestly

- Report provider, product, area, dates, index/measurement, resolution, and available sample counts.
- Distinguish live measurements from plans, fixtures, and missing/null data. API acceptance alone does not establish scientific correctness.
- Cloud cover is scene-level metadata, not necessarily the cloud fraction of the requested area.
- An index is a proxy, not proof of crop yield, flooding, fire damage, or a causal explanation.
- Check product units, scaling, masks, and categorical treatment before interpreting values. A median class code is not necessarily a dominant land-cover class.
- Do not describe averaged interval percentiles as percentiles of the pooled samples.
- `water-temperature` concerns lakes, not open-ocean SST. `aerosol` is an absorbing aerosol index, not Sentinel-3 aerosol optical depth.
- Persistent `earth watch` monitoring is not implemented; do not invent scheduling or alerting commands.

## Troubleshoot

Use `--verbose` or `--debug` for request/status/retry diagnostics on stderr.
Earth omits request bodies, headers, URL credentials, and query strings from these diagnostics. Still inspect other output before sharing it.
Handle authentication and quota failures explicitly; do not retry non-transient errors blindly.
Exit codes: `0` success, `1` operational failure, `2` usage error.
