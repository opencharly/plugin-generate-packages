# AGENTS.md — plugin-generate-packages

Standalone plugin repo owning the externalized `charly generate-packages` command
(`command:generate-packages`). The plugin is a Go module at
`candy/generate-packages/` (module path
`github.com/opencharly/plugin-generate-packages/candy/generate-packages`); the
root `charly.yml` declares `discover: candy` (non-recursive) so the repo is a
project and its candy is scanned.

Canonical files:

- `candy/generate-packages/charly.yml` — the `generate-packages:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/generate-packages/plugin.go` — `NewProvider()` / `NewMeta()` /
  `CliMain` and the `Invoke(OpRun)` surface.
- `candy/generate-packages/cli.go` — the `charly generate-packages` Kong grammar
  + the shared effect (`runFromArgs` → `runGeneratePackages`).
- `candy/generate-packages/schema/generate-packages.cue` — the self-contained
  plugin schema.
- `candy/generate-packages/testdata/charly.yml` — the e2e packaging fixture.
- `generate-packages.providers` — the committed providers manifest
  (`command:generate-packages`).
- `.github/workflows/ci.yml` — `go vet` + `go test ./...` on the plugin module.
- `.github/workflows/release.yml` — prebuilt binary + providers-manifest release
  assets on a `v*` tag.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:go` — the `charly generate-packages` packaging/release
  command reference (the plugin's user-facing surface: the CLI flags + the
  `packaging:` section it reads). Load before changing the CLI grammar or the
  packaging output.
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the `command` provider class, the per-plugin CUE-schema contract,
  placement. Load before touching the provider or schema.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/generate-packages/` — compile the plugin module.
- `go test ./...` in `candy/generate-packages/` — the CLI parse tests + the e2e
  (builds a real package per format + per variant).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.

## Modify this repo

- Edit the `generate-packages:` candy entity, the Go source, and
  `schema/generate-packages.cue` **together** — the schema is the served
  declaration surface.
- The plugin is the command surface ONLY: nFPM/parsing/variant/optdepends
  functions belong in `sdk/packagekit`, not here. Keep one implementation (R3).
- The module lives in a subdirectory, so its Go module tag is
  `candy/generate-packages/v0.<YYYYDDD>.<HHMM>` — NOT the root `v<YYYY.DDD.HHMM>`
  release tag. Both are minted at merge.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
