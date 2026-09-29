# plugin-generate-packages

The `charly generate-packages` command plugin — the single packaging engine for
all nFPM formats (deb, rpm, apk, archlinux, ipk, msix), driven by the
`packaging:` section of a charly candy's `charly.yml`.

This plugin is the **command surface only**. Every nFPM/parsing/variant/
optdepends function lives in [`sdk/packagekit`](https://github.com/opencharly/sdk)
(the shared nFPM packaging driver), which the plugin calls via
`packagekit.LoadPackaging` + `packagekit.Build`. The plugin is only the Kong
grammar + provider wiring.

## What it provides

| Capability | Surface |
|---|---|
| `command:generate-packages` | the `charly generate-packages` CLI — build native packages per `packaging:` variant/format |

## How to use it

`charly generate-packages` reads the `packaging:` section from a charly candy's
`charly.yml` — the **single metadata source** — and builds native packages for
the requested formats from that + the CLI flags. No deps table, no PKGBUILD /
spec / debian-control, no other metadata resource.

```
charly generate-packages \
  --candy ./candy/charly/charly.yml \
  --binary ./bin/charly \
  --plugins ./plugins \
  --version 2026.225.1200 \
  --arch amd64 \
  --variant full \
  --formats deb,rpm,archlinux,apk,ipk \
  --out dist/ \
  [--signing-key key.gpg]        # deb/rpm; passphrase via NFPM_PASSPHRASE
  [--apk-signing-key key.rsa]    # apk; passphrase via NFPM_APK_PASSPHRASE
  [--msix-pfx cert.pfx]          # msix (deferred); passphrase via NFPM_MSIX_PASSPHRASE
```

- **`--candy` is the single metadata input** — the plugin reads the `packaging:`
  section from it and builds from that + the flags.
- **`--variant`** selects the variant from `packaging.variants`; the package name
  is `charly` for the format's `default_variant`, `charly-<variant>` otherwise.
  The `--plugins` dir is filtered to the variant's plugin list.
- **One `--arch` per invocation** — `--binary`/`--plugins` are arch-specific, so a
  distro workflow loops over `amd64`/`arm64`.
- A variant naming a plugin absent from `--plugins` fails loudly.

## The `packaging:` section

The `packaging:` section (spec `#Packaging`) declares the package metadata +
per-variant plugin sets + per-format deps:

```yaml
packaging:
  name: charly
  description: the charly CLI
  maintainer: OpenCharly <maintainers@opencharly.ai>
  homepage: https://opencharly.ai
  license: MIT
  variants:
    default:
      description: the default plugin set
      plugins: [plugin-a, plugin-b]
    minimal:
      description: the minimal plugin set
      plugins: [plugin-a]
  formats:
    deb:
      depends: [podman]
      default_variant: default
```

## Development

```bash
cd candy/generate-packages
go test ./...   # CLI parse tests + the e2e (builds a package per format + per variant)
```

The e2e drives the full plugin path (`runFromArgs` → Kong parse →
`runGeneratePackages` → `sdk/packagekit.Build`) against the `testdata/charly.yml`
fixture and asserts the packages land with the right structure.

## Release

On a `v*` tag push, the release workflow publishes the prebuilt
`generate-packages-linux-<arch>` binaries (amd64 + arm64, `CGO_ENABLED=0`) + the
committed `generate-packages.providers` manifest (content:
`command:generate-packages`) as release assets. The per-distro package repos
download the amd64 asset and drop it into `/usr/lib/charly/plugins/` so
`charly generate-packages` resolves project-less via `discoverBakedPluginWords`.

## Consuming the Go module — the tag form is not the release tag

The module lives in a repository **subdirectory**, so Go looks for a tag whose
name is the module's subdirectory path followed by the version:

```
candy/generate-packages/v0.<YYYYDDD>.<HHMM>
```

A bare root tag `v<YYYY.DDD.HHMM>` is **not** consulted for a subdirectory module.
Both tags are minted at merge on the same merged HEAD; only the root `v*` tag
triggers the release workflow. The `v0.` major is required because a `major >= 2`
version would force a `/vN` suffix on the module path, and `<HHMM>` strips leading
zeros (`0013` → `13`) because semver rejects a leading-zero numeric segment.

## Layout

- `candy/generate-packages/` — the plugin module: `plugin.go` (provider + meta),
  `cli.go` (the Kong grammar + effect), `schema/generate-packages.cue`,
  `testdata/charly.yml`, `cmd/serve/main.go`.
- `generate-packages.providers` — the committed providers manifest
  (`command:generate-packages`).
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/ci.yml` / `release.yml` — the `go test` CI and the release
  asset publish.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-internals:plugin` — the plugin/provider model. This candy
  carries no `skill:` entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- [`opencharly/sdk`](https://github.com/opencharly/sdk) — `packagekit`.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.

## License

MIT — see [LICENSE](LICENSE).
