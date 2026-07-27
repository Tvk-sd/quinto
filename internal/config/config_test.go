package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, contents string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if contents == "" {
		return
	}
	if err := os.MkdirAll(filepath.Join(dir, "quinto"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "quinto", "config")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

// An existing single-site config, with no [section] anywhere, must keep
// working exactly as it did before named sites existed.
func TestUnsectionedConfigIsTheDefaultSite(t *testing.T) {
	writeConfig(t, "site  = tillvonkrueger.goatcounter.com\ntoken = abc123\n")

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	if c.Site != "tillvonkrueger.goatcounter.com" || c.Token != "abc123" {
		t.Fatalf("got %+v", c)
	}

	sites, err := Sites()
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 1 || sites[0].Name != "tillvonkrueger" {
		t.Fatalf("Sites() = %+v, want one entry named tillvonkrueger", sites)
	}
}

func TestNamedSectionIsSelectableByBracketName(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[mctimey]
site  = mctimey.goatcounter.com
token = def456
`)

	c, err := Load("mctimey")
	if err != nil {
		t.Fatalf("Load(\"mctimey\"): %v", err)
	}
	if c.Site != "mctimey.goatcounter.com" || c.Token != "def456" {
		t.Fatalf("got %+v", c)
	}
}

// The default is addressable by its derived name too, not just by omitting
// --site — bare `quinto` and `--site tillvonkrueger` mean the same thing.
func TestDefaultIsSelectableByItsDerivedName(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[mctimey]
site  = mctimey.goatcounter.com
token = def456
`)

	bare, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	named, err := Load("tillvonkrueger")
	if err != nil {
		t.Fatal(err)
	}
	if bare != named {
		t.Fatalf("Load(\"\") = %+v, Load(\"tillvonkrueger\") = %+v — should match", bare, named)
	}
}

func TestSitesListsDefaultThenNamedInFileOrder(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[mctimey]
site  = mctimey.goatcounter.com
token = def456

[work]
site  = work.goatcounter.com
token = ghi789
`)

	sites, err := Sites()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"tillvonkrueger", "mctimey", "work"}
	if len(sites) != len(want) {
		t.Fatalf("Sites() = %+v, want %d entries", sites, len(want))
	}
	for i, name := range want {
		if sites[i].Name != name {
			t.Errorf("Sites()[%d].Name = %q, want %q", i, sites[i].Name, name)
		}
	}
}

func TestUnknownSiteErrorsNamingWhatIsConfigured(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[mctimey]
site  = mctimey.goatcounter.com
token = def456
`)

	_, err := Load("nope")
	if err == nil {
		t.Fatal("expected an error for an unconfigured site")
	}
	msg := err.Error()
	if !strings.Contains(msg, "tillvonkrueger") || !strings.Contains(msg, "mctimey") {
		t.Fatalf("error %q does not name the configured sites", msg)
	}
}

func TestEnvOverridesTheDefaultSite(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[mctimey]
site  = mctimey.goatcounter.com
token = def456
`)
	t.Setenv("QUINTO_GOATCOUNTER_SITE", "onetime.goatcounter.com")
	t.Setenv("QUINTO_GOATCOUNTER_TOKEN", "onetime-token")

	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.Site != "onetime.goatcounter.com" || c.Token != "onetime-token" {
		t.Fatalf("env override did not win: %+v", c)
	}
}

// --site and the env override are two different ways of picking a site's
// credentials. Combining them is ambiguous about which site's data ends up
// in which database file, so it must fail rather than silently pick one.
func TestSiteAndEnvOverrideTogetherIsAnError(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[mctimey]
site  = mctimey.goatcounter.com
token = def456
`)
	t.Setenv("QUINTO_GOATCOUNTER_SITE", "onetime.goatcounter.com")

	if _, err := Load("mctimey"); err == nil {
		t.Fatal("expected an error combining --site with an env override")
	}
}

// A one-off site needs no config edit at all — env vars alone are valid,
// exactly as they were before named sites existed.
func TestEnvOnlyConfigNeedsNoFile(t *testing.T) {
	writeConfig(t, "")
	t.Setenv("QUINTO_GOATCOUNTER_SITE", "onetime.goatcounter.com")
	t.Setenv("QUINTO_GOATCOUNTER_TOKEN", "onetime-token")

	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.Site != "onetime.goatcounter.com" || c.Token != "onetime-token" {
		t.Fatalf("got %+v", c)
	}

	sites, err := Sites()
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 0 {
		t.Fatalf("Sites() = %+v, want none — env-only config has no named entries", sites)
	}
}

func TestMissingConfigFileIsNotAnErrorForSitesOrDefaultLoad(t *testing.T) {
	writeConfig(t, "")

	if _, err := Sites(); err != nil {
		t.Fatalf("Sites() on a missing file: %v", err)
	}

	// Load("") with nothing configured anywhere still errors — but with the
	// "set up credentials" message, not a file-not-found error.
	if _, err := Load(""); err == nil {
		t.Fatal("expected an error when nothing is configured at all")
	}
}

func TestTwoSitesWithTheSameNameIsAConfigError(t *testing.T) {
	writeConfig(t, `site  = tillvonkrueger.goatcounter.com
token = abc123

[tillvonkrueger]
site  = other.goatcounter.com
token = def456
`)

	if _, err := Sites(); err == nil {
		t.Fatal("expected an error for two sites sharing a name")
	}
}
