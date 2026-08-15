package generatepackages

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

// TestCliGrammar_Parse asserts the Kong grammar parses the full flag surface and
// defaults the optional flags.
func TestCliGrammar_Parse(t *testing.T) {
	var c cli
	k, err := kong.New(&c, kong.Name("generate-packages"))
	if err != nil {
		t.Fatalf("kong.New: %v", err)
	}
	kctx, err := k.Parse([]string{
		"--candy", "candy/charly/charly.yml",
		"--binary", "bin/charly",
		"--plugins", "plugins",
		"--version", "2026.225.1200",
		"--arch", "amd64",
		"--variant", "full",
		"--formats", "deb,rpm,archlinux",
		"--out", "dist/",
		"--signing-key", "key.gpg",
		"--apk-signing-key", "key.rsa",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_ = kctx
	if c.Candy != "candy/charly/charly.yml" {
		t.Errorf("Candy = %q, want %q", c.Candy, "candy/charly/charly.yml")
	}
	if c.Binary != "bin/charly" || c.Plugins != "plugins" {
		t.Errorf("Binary/Plugins = %q/%q", c.Binary, c.Plugins)
	}
	if c.Version != "2026.225.1200" || c.Arch != "amd64" {
		t.Errorf("Version/Arch = %q/%q", c.Version, c.Arch)
	}
	if c.Variant != "full" {
		t.Errorf("Variant = %q, want full", c.Variant)
	}
	if len(c.Formats) != 3 || c.Formats[0] != "deb" || c.Formats[2] != "archlinux" {
		t.Errorf("Formats = %v, want [deb rpm archlinux]", c.Formats)
	}
	if c.Out != "dist/" {
		t.Errorf("Out = %q, want dist/", c.Out)
	}
	if c.SigningKey != "key.gpg" || c.APKKey != "key.rsa" {
		t.Errorf("SigningKey/APKKey = %q/%q", c.SigningKey, c.APKKey)
	}
	// Optional flags default empty.
	if c.MSIXPFX != "" || c.MSIXLogo != "" {
		t.Errorf("MSIXPFX/MSIXLogo = %q/%q, want empty", c.MSIXPFX, c.MSIXLogo)
	}
}

// TestCliGrammar_Required asserts a missing required flag is a parse error.
func TestCliGrammar_Required(t *testing.T) {
	var c cli
	k, err := kong.New(&c, kong.Name("generate-packages"))
	if err != nil {
		t.Fatalf("kong.New: %v", err)
	}
	_, err = k.Parse([]string{
		"--binary", "bin/charly",
		"--plugins", "plugins",
		"--version", "2026.225.1200",
		"--arch", "amd64",
		"--out", "dist/",
	})
	if err == nil {
		t.Fatal("expected a parse error for the missing --candy flag")
	}
	if !strings.Contains(err.Error(), "candy") {
		t.Errorf("error %q does not name the missing --candy flag", err)
	}
}
