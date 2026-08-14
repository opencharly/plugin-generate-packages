package generatepackages

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// TestGeneratePackagesE2E drives the FULL plugin path (runFromArgs → Kong parse →
// runGeneratePackages → sdk/packagekit.Build) against the testdata fixture and
// asserts the packages land with the right structure: one file per requested format,
// the archlinux .PKGINFO carrying the injected optdepends, and the variant's plugin
// set filtered into the package. The deep per-format structure is covered by the
// sdk/packagekit unit tests; this is the plugin's thin end-to-end smoke.
func TestGeneratePackagesE2E(t *testing.T) {
	dir := t.TempDir()
	binary := writeFakeBinary(t, dir)
	plugins := writeFakePlugins(t, dir)
	out := filepath.Join(dir, "dist")

	// Default variant, the five Linux formats (msix is deferred — no Windows binary).
	args := []string{
		"--candy", "testdata/charly.yml",
		"--binary", binary,
		"--plugins", plugins,
		"--version", "2026.225.1200",
		"--arch", "amd64",
		"--formats", "deb,rpm,archlinux,apk,ipk",
		"--out", out,
	}
	if err := runFromArgs(args); err != nil {
		t.Fatalf("runFromArgs (default variant): %v", err)
	}
	for _, f := range []string{"charly_2026.225.1200_amd64.deb", "charly-2026.225.1200-1.x86_64.rpm", "charly-2026.225.1200-1-x86_64.pkg.tar.zst", "charly_2026.225.1200_x86_64.apk", "charly_2026.225.1200_x86_64.ipk"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("expected package %s: %v", f, err)
		}
	}

	// The archlinux .PKGINFO: name/version/arch + the injected optdepends.
	pkginfo, files := readPkg(t, filepath.Join(out, "charly-2026.225.1200-1-x86_64.pkg.tar.zst"))
	for _, want := range []string{"pkgname = charly", "pkgver = 2026.225.1200", "arch = x86_64", "optdepend = libvirt: for VM management"} {
		if !strings.Contains(pkginfo, want) {
			t.Errorf(".PKGINFO missing %q:\n%s", want, pkginfo)
		}
	}
	// The default variant's plugin set is filtered into the package (the tar file list).
	for _, want := range []string{"usr/bin/charly", "usr/lib/charly/plugins/plugin-a", "usr/lib/charly/plugins/plugin-b", "usr/lib/charly/plugins/plugin-a.providers"} {
		if !files[want] {
			t.Errorf("package file list missing %q (have %v)", want, sortedKeys(files))
		}
	}

	// The minimal variant packages as charly-minimal with only plugin-a.
	minOut := filepath.Join(dir, "dist-minimal")
	args = []string{
		"--candy", "testdata/charly.yml",
		"--binary", binary,
		"--plugins", plugins,
		"--version", "2026.225.1200",
		"--arch", "amd64",
		"--variant", "minimal",
		"--formats", "archlinux",
		"--out", minOut,
	}
	if err := runFromArgs(args); err != nil {
		t.Fatalf("runFromArgs (minimal variant): %v", err)
	}
	minPkg := filepath.Join(minOut, "charly-minimal-2026.225.1200-1-x86_64.pkg.tar.zst")
	if _, err := os.Stat(minPkg); err != nil {
		t.Fatalf("expected charly-minimal package: %v", err)
	}
	minInfo, minFiles := readPkg(t, minPkg)
	if !strings.Contains(minInfo, "pkgname = charly-minimal") {
		t.Errorf("minimal .PKGINFO pkgname not charly-minimal:\n%s", minInfo)
	}
	if minFiles["usr/lib/charly/plugins/plugin-b"] {
		t.Errorf("minimal variant leaked plugin-b into the package (have %v)", sortedKeys(minFiles))
	}
	if !minFiles["usr/lib/charly/plugins/plugin-a"] {
		t.Errorf("minimal variant missing plugin-a (have %v)", sortedKeys(minFiles))
	}
}

// writeFakeBinary writes a minimal executable "binary" (a shell script) and returns
// its path.
func writeFakeBinary(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "charly")
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho fake charly\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return p
}

// writeFakePlugins writes a plugins dir with two fake plugins + their .providers
// manifests and returns the dir path.
func writeFakePlugins(t *testing.T, dir string) string {
	t.Helper()
	plugins := filepath.Join(dir, "plugins")
	if err := os.MkdirAll(plugins, 0o755); err != nil {
		t.Fatalf("mkdir plugins: %v", err)
	}
	for _, f := range []string{"plugin-a", "plugin-b"} {
		if err := os.WriteFile(filepath.Join(plugins, f), []byte("#!/bin/sh\necho "+f+"\n"), 0o755); err != nil {
			t.Fatalf("write fake plugin %s: %v", f, err)
		}
		if err := os.WriteFile(filepath.Join(plugins, f+".providers"), []byte("verb:"+f+"\n"), 0o644); err != nil {
			t.Fatalf("write fake providers %s: %v", f, err)
		}
	}
	return plugins
}

// readPkg reads an archlinux .pkg.tar.zst (zstd → tar) and returns the .PKGINFO
// text plus the set of entry names in the tar (the package file list).
func readPkg(t *testing.T, pkgPath string) (string, map[string]bool) {
	t.Helper()
	f, err := os.Open(pkgPath)
	if err != nil {
		t.Fatalf("open %s: %v", pkgPath, err)
	}
	defer func() { _ = f.Close() }()
	zr, err := zstd.NewReader(f)
	if err != nil {
		t.Fatalf("zstd reader: %v", err)
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	files := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		files[hdr.Name] = true
		if hdr.Name == ".PKGINFO" {
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, tr); err != nil {
				t.Fatalf("read .PKGINFO: %v", err)
			}
			return buf.String(), files
		}
	}
	t.Fatalf("no .PKGINFO in %s", pkgPath)
	return "", nil
}

// sortedKeys returns the sorted keys of m for stable error messages.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
