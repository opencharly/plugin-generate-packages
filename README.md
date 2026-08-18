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

**Two different directories share the path `candy/generate-packages/`, in two different
repositories.** They are easy to conflate and this section is about the first:

| | repository | what it is |
|---|---|---|
| `candy/generate-packages/` | **this repo** (`opencharly/plugin-generate-packages`) | the Go module — the plugin itself |
| `candy/generate-packages/` | the superproject (`opencharly/charly`) | a thin re-export shim candy that `require`s the module above |

The superproject shim is the consumer; this repo's module is the dependency. That consumer
cannot `require` the dependency at its release tag. This repo's module lives in a repository
**subdirectory**, so Go looks for a tag whose name is the module's subdirectory path followed
by the version:

```
candy/generate-packages/v0.<YYYYDDD>.<HHMM>
```

Measured — Go names the ref it wants, and a bare root tag is not it:

```
$ go list -m github.com/opencharly/plugin-generate-packages/candy/generate-packages@v0.2026227.1233
go: ...@v0.2026227.1233: invalid version:
    unknown revision candy/generate-packages/v0.2026227.1233
```

**Go never falls back to a root tag for a subdirectory module**, which the failing case above
cannot show on its own. An external control settles it, using
[`hashicorp/consul`](https://github.com/hashicorp/consul) (a repo whose `api/` subdirectory
module is independently tagged). Verbatim:

```
$ go list -m github.com/hashicorp/consul/api@v1.32.0
github.com/hashicorp/consul/api v1.32.0
$ git ls-remote --tags https://github.com/hashicorp/consul refs/tags/v1.9.9 refs/tags/api/v1.32.0
1f486aab22986b70de1ae9aaa368d0613091898e	refs/tags/api/v1.32.0
8159a14bed92f774437618587fc8b38fe603ade1	refs/tags/v1.9.9
$ go list -m github.com/hashicorp/consul/api@v1.9.9
go: github.com/hashicorp/consul/api@v1.9.9: invalid version: unknown revision api/v1.9.9
$ git ls-remote --tags https://github.com/hashicorp/consul refs/tags/api/v1.9.9
(no output — the prefixed tag is absent)
```

A ROOT tag exists in BOTH cases (`v1.9.9`), so the only variable between resolving and failing
is whether the PREFIXED tag is present.

The negative case is the decisive one: a root tag that genuinely exists is still not consulted
for a subdirectory module.

The `v0.` major is required for the same reason the [sdk](https://github.com/opencharly/sdk)
uses it: a `major >= 2` version would force a `/vN` suffix on the module path. `<HHMM>` is
written with leading zeros stripped (`0013` -> `13`), mirroring the sdk scheme, because semver
rejects a leading-zero numeric segment. A midnight merge (`0000`) strips to `0`.

**Both tags are minted at merge, on the same merged HEAD**: the release tag
`v<YYYY.DDD.HHMM>` and the module tag `candy/generate-packages/v0.<YYYYDDD>.<HHMM>`. Only the
first triggers the release workflow — its `on.push.tags` filter is `v*`, and a module tag does
not begin with `v`. **Observed**, not predicted: minting the first module tag
(`candy/generate-packages/v0.2026230.708`) took the release-workflow run count from 2 to 3,
and the single new run was on `v2026.230.0708`. The module tag triggered nothing.

**A consumer with no module tag to pin resolves a pseudo-version** — `@latest` answered
`v0.0.0-<utc>-<sha>` before the first module tag existed. That resolves and builds; it simply
carries no release identity, which is what the module tag adds. The shim was therefore never
*blocked* on the tag, only pinned less precisely.

## License

MIT — see [LICENSE](LICENSE).
