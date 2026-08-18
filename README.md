# plugin-generate-packages

The `charly generate-packages` command plugin — the single packaging engine for
all nFPM formats (deb, rpm, apk, archlinux, ipk, msix), driven by the
`packaging:` section of a charly candy's `charly.yml`.

This plugin is the **command surface only**. Every nFPM/parsing/variant/
optdepends function lives in [`sdk/packagekit`](https://github.com/opencharly/sdk)
(the shared nFPM packaging driver), which the plugin calls via
`packagekit.LoadPackaging` + `packagekit.Build`. The plugin is only the Kong
grammar + provider wiring.

## What it does

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

- **`--candy` is the single metadata input** — the charly candy's charly.yml.
  The plugin reads the `packaging:` section from it and builds the packages from
  that + the flags.
- **`--variant`** selects the variant from `packaging.variants`; the package name
  is `charly` for the format's `default_variant`, `charly-<variant>` otherwise.
  The `--plugins` dir is filtered to the variant's plugin list.
- **One `--arch` per invocation** — `--binary`/`--plugins` are arch-specific, so
  a single arch keeps the CLI unambiguous; a distro workflow loops over
  `amd64`/`arm64`.
- `--formats` defaults to all six; validated against `nfpm.Enumerate()`.

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
    archlinux:
      depends: [podman]
      optdepends:
        libvirt: for VM management
      default_variant: default
```

A variant naming a plugin not present in the `--plugins` dir fails loudly at
package-build time.

## Known nFPM gap: archlinux optdepends

nFPM does not emit Arch `optdepends` (PKGINFO supports it, nFPM doesn't expose
it). `sdk/packagekit` post-processes the `.pkg.tar.zst` to inject
`optdepend = <pkg>: <desc>` lines into `.PKGINFO` (deterministic, unit-tested).

## Development

```bash
cd candy/generate-packages
go test ./...   # CLI parse tests + the e2e (builds a package per format + per variant)
```

The e2e drives the FULL plugin path (`runFromArgs` → Kong parse →
`runGeneratePackages` → `sdk/packagekit.Build`) against the `testdata/charly.yml`
fixture and asserts the packages land with the right structure: one file per
requested format, the archlinux `.PKGINFO` carrying the injected optdepends, and
the variant's plugin set filtered into the package.

## Release

On a `v*` tag push, the release workflow publishes the prebuilt
`generate-packages-linux-<arch>` binaries (amd64 + arm64, `CGO_ENABLED=0`) + the
committed `generate-packages.providers` manifest (content:
`command:generate-packages`) as release assets. The per-distro package repos
download the amd64 asset and drop it into `/usr/lib/charly/plugins/` so
`charly generate-packages` resolves project-less via
`discoverBakedPluginWords`.

### Go-module consumption — the tag form is NOT the release tag

A consumer that `require`s this plugin as a **Go module** (the superproject's
`candy/generate-packages/` re-export shim) cannot use the release tag. The module
lives in a repository **subdirectory**, so Go looks for a tag whose name is the
module's subdirectory path followed by the version:

```
candy/generate-packages/v0.<YYYYDDD>.<HHMM>
```

Measured — Go names the ref it wants, and a bare root tag is not it:

```
$ go list -m github.com/opencharly/plugin-generate-packages/candy/generate-packages@v0.2026227.1233
go: ...@v0.2026227.1233: invalid version:
    unknown revision candy/generate-packages/v0.2026227.1233
```

The `v0.` major is required for the same reason the sdk uses it: a `major >= 2`
version would force a `/vN` suffix on the module path. `<HHMM>` is written with
leading zeros stripped (`0013` -> `13`), again mirroring the sdk scheme — semver
rejects a leading-zero numeric segment.

**Both tags are minted at merge, on the same merged HEAD**: the release tag
`v<YYYY.DDD.HHMM>` and the module tag `candy/generate-packages/v0.<YYYYDDD>.<HHMM>`.
Only the first is expected to trigger the release workflow — its `on.push.tags`
filter is `v*`, and a module tag does not begin with `v`. That follows from the
filter's glob semantics; the first module tag pushed is the confirmation.

**Until a module tag exists, a consumer resolves a pseudo-version** — `@latest`
answers `v0.0.0-<utc>-<sha>` today. That resolves and builds; it simply carries no
release identity, which is why the module tag exists.

## License

MIT — see [LICENSE](LICENSE).
