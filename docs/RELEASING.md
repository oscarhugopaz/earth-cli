# Releasing earth-cli

The release flow mirrors the conventions already used by `penpot-cli` and
`mainwp-cli`, adapted for a Go binary built with GoReleaser.

## Developer flow

```bash
git tag v0.1.0
git push origin v0.1.0
```

That is all. The tag triggers `.github/workflows/release.yml`, which:

1. validates the tag shape (`vMAJOR.MINOR.PATCH`);
2. runs `go test ./...` and `go vet ./...`;
3. runs GoReleaser to build and publish the release;
4. notifies the Homebrew tap so the formula is updated.

## What GoReleaser publishes

For each release, configured in `.goreleaser.yaml`:

| OS      | Architectures |
| ------- | ------------- |
| darwin  | arm64, amd64  |
| linux   | arm64, amd64  |
| windows | amd64         |

Artifacts use deterministic names:

```
earth-cli_<version>_<os>_<arch>.tar.gz   # darwin/linux
earth-cli_<version>_windows_amd64.zip    # windows
checksums.txt                            # SHA256 for every artifact
```

The binary is always called `earth`. Version, commit, and build date are
injected into `internal/version` via `-ldflags`.

A snapshot build can be produced locally with:

```bash
make snapshot   # goreleaser release --snapshot --clean
```

## Automatic Homebrew tap update

The `notify-homebrew` job in `release.yml` calls the reusable
`.github/workflows/notify-homebrew-tap.yml`, which sends a
`repository_dispatch` event named `earth-cli-release` to
`oscarhugopaz/homebrew-tap`, carrying the tag.

The tap contains a matching workflow
(`.github/workflows/update-earth-cli.yml`) that:

1. resolves the requested version;
2. downloads and verifies the four `earth-cli_<version>_*` archives against
   `checksums.txt`;
3. generates `Formula/earth-cli.rb` with per-architecture URLs and checksums;
4. validates it with `brew style`, `brew audit`, `brew install`, and `brew test`;
5. commits and pushes the updated formula.

The formula installs the binary as `earth`, installs shell completions, and has
a network-free test (`earth version`).

## Required GitHub secret

The dispatch needs a token that can create repository dispatch events on
`oscarhugopaz/homebrew-tap`.

- **Where:** `oscarhugopaz/earth-cli` → Settings → Secrets and variables →
  Actions → New repository secret.
- **Name:** `TAP_DISPATCH_TOKEN`
- **Type:** fine-grained Personal Access Token (preferred).
- **Repository access:** only `oscarhugopaz/homebrew-tap`.
- **Permissions:** `Contents: Read and write` (`Metadata: Read` is added
  automatically).
- **Classic PAT alternative:** any token with the `repo` scope.

Only the dispatch is performed with this token. The tap workflow itself commits
with the tap repository's own `GITHUB_TOKEN` (`contents: write`), so no secret
needs to be stored in `homebrew-tap`.

To re-run a missed dispatch manually:

```
earth-cli → Actions → Notify Homebrew Tap → Run workflow → version: 0.1.0
```

or, in the tap, `Update earth-cli formula → Run workflow → version: 0.1.0`.

## Differences from mainwp-cli

`mainwp-cli` is a shell tool that releases the source tarball (`archive/refs/tags/*.tar.gz`)
and updates a formula that installs the whole repository plus runtime
dependencies (`gum`, `jq`).

`earth-cli` is a compiled Go binary, so:

- artifacts are per-platform GoReleaser archives instead of a single source
  tarball;
- the formula has `on_macos`/`on_linux` (arm/intel) blocks with one checksum per
  archive, instead of a single `url`/`sha256`;
- there are no runtime dependencies: it is a standalone binary.

The user experience and the automation strategy are otherwise identical: a tag
push publishes a GitHub Release and updates the tap automatically through a
`repository_dispatch` using the same `TAP_DISPATCH_TOKEN` secret name and the
same tap-side validation (`brew audit`, `brew install`, `brew test`).
