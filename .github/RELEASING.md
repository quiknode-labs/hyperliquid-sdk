# Releasing

Each language SDK is released independently by pushing a tag of the form `<lang>/vX.Y.Z`.
Every publish workflow asserts that the tag version exactly matches the version in that
language's manifest and fails loudly (printing both values) if they differ — bump the
manifest first, then tag.

## How to release

1. Bump the version in the manifest:
   - TypeScript: `typescript/package.json` → `version`
   - Python: `python/pyproject.toml` → `[project] version`
   - Rust: `rust/Cargo.toml` → `[package] version`
   - Go: no manifest version — the tag *is* the version
2. Commit and merge to `main`.
3. Tag and push:

   ```sh
   git tag typescript/v0.2.0   # or python/v..., rust/v..., go/v...
   git push origin typescript/v0.2.0
   ```

What each tag triggers:

| Tag | Workflow | Publishes to |
| --- | --- | --- |
| `typescript/vX.Y.Z` | `publish-typescript.yml` | npm (`@quicknode/hyperliquid-sdk`) with provenance |
| `python/vX.Y.Z` | `publish-python.yml` | PyPI (`hyperliquid-sdk`) via Trusted Publishing |
| `rust/vX.Y.Z` | `publish-rust.yml` | crates.io (`quicknode-hyperliquid-sdk`) |
| `go/vX.Y.Z` | `publish-go.yml` | nothing — the Go module proxy serves the tag; the workflow is a build/test release gate |

## One-time setup

- **npm**: add an `NPM_TOKEN` repository secret — a granular automation token with
  publish access to `@quicknode/hyperliquid-sdk`. The workflow publishes with
  `--provenance` (uses GitHub OIDC via `id-token: write`; the token authenticates,
  provenance is attested automatically).
- **PyPI**: no secret. Configure a Trusted Publisher on the `hyperliquid-sdk` PyPI
  project pointing at:
  - owner/repo: `quiknode-labs/hyperliquid-sdk`
  - workflow: `publish-python.yml`
  - environment: `pypi`

  Also create a GitHub environment named `pypi` in this repo (Settings → Environments);
  optionally add required reviewers to gate releases.
- **crates.io**: add a `CARGO_REGISTRY_TOKEN` repository secret — a crates.io API token
  with publish rights for `quicknode-hyperliquid-sdk`.
- **Go**: nothing to configure.

## Known drift to reconcile (as of 2026-07-19)

Earlier versions were published manually, so tags and registries are out of sync:

- **Rust**: crates.io is at **0.1.9** but a `rust/v0.1.10` tag already exists and
  `rust/Cargo.toml` says 0.1.10. First automated release should either re-push a fresh
  `rust/v0.1.10` tag to publish 0.1.10, or bump to 0.1.11 and tag `rust/v0.1.11`.
  (Note: pushing a tag that already exists does not trigger the workflow — delete and
  re-push, or just bump.)
- **npm**: 0.1.10 is live on npm but was published without a `typescript/v0.1.10` tag.
  Next release starts the tagged flow (e.g. `typescript/v0.1.11`).
- **PyPI**: 0.7.6 is live but was published without a `python/v0.7.6` tag. Next release
  starts the tagged flow (e.g. `python/v0.7.7`).
