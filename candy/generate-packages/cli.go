package generatepackages

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/opencharly/sdk/packagekit"
)

// stderr is a package var so tests can capture the CLI's error output.
var stderr io.Writer = os.Stderr

// cli is the Kong grammar for `charly generate-packages`. The command args are
// pass-through CLI tokens (a command has no structured plugin_input), so the grammar
// is built host-side (externalCommandHolder) for both placements and parsed here.
type cli struct {
	Candy      string   `name:"candy" help:"path to the charly candy's charly.yml (the packaging: section is the single metadata source)" required:""`
	Binary     string   `name:"binary" help:"path to the charly binary to package" required:""`
	Plugins    string   `name:"plugins" help:"path to the plugins dir to package (filtered to the variant's plugin list)" required:""`
	Version    string   `name:"version" help:"the CalVer to stamp the packages with" required:""`
	Arch       string   `name:"arch" help:"GOARCH name (amd64/arm64)" required:""`
	Variant    string   `name:"variant" help:"the variant name (default: the format's default_variant)"`
	Formats    []string `name:"formats" help:"the nFPM formats to build (default: all registered formats)" sep:","`
	Out        string   `name:"out" help:"the output dir" required:""`
	SigningKey string   `name:"signing-key" help:"deb/rpm GPG key file (passphrase via NFPM_PASSPHRASE)"`
	APKKey     string   `name:"apk-signing-key" help:"apk RSA key file (passphrase via NFPM_APK_PASSPHRASE)"`
	MSIXPFX    string   `name:"msix-pfx" help:"msix PFX cert file (passphrase via NFPM_MSIX_PASSPHRASE)"`
	MSIXLogo   string   `name:"msix-logo" help:"msix logo file (required by the msix packager)"`
}

// runFromArgs parses the pass-through args via Kong and runs the command effect. It is
// the ONE effect shared by both placements (CliMain out-of-process, Invoke(OpRun)
// in-proc).
func runFromArgs(args []string) error {
	var c cli
	k, err := kong.New(&c,
		kong.Name("generate-packages"),
		kong.Description("Build native packages (deb/rpm/apk/archlinux/ipk/msix) via nFPM, driven by the packaging: section of a charly candy's charly.yml."),
	)
	if err != nil {
		return fmt.Errorf("generate-packages: %w", err)
	}
	if _, err := k.Parse(args); err != nil {
		return fmt.Errorf("generate-packages: %w", err)
	}
	return runGeneratePackages(&c)
}

// runGeneratePackages is the command's effect: load the packaging section from the
// charly.yml, build the requested formats for the (variant, arch), and print the
// written package paths to stdout.
func runGeneratePackages(c *cli) error {
	pkg, err := packagekit.LoadPackaging(c.Candy)
	if err != nil {
		return fmt.Errorf("generate-packages: load packaging: %w", err)
	}
	opts := packagekit.BuildOptions{
		Binary:     c.Binary,
		PluginsDir: c.Plugins,
		Version:    c.Version,
		Arch:       c.Arch,
		Variant:    c.Variant,
		Formats:    c.Formats,
		Out:        c.Out,
		SigningKey: c.SigningKey,
		APKKey:     c.APKKey,
		MSIXPFX:    c.MSIXPFX,
		MSIXLogo:   c.MSIXLogo,
	}
	written, err := packagekit.Build(pkg, opts)
	if err != nil {
		return fmt.Errorf("generate-packages: build: %w", err)
	}
	for _, p := range written {
		fmt.Println(p)
	}
	return nil
}
