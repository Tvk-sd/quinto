// Package config loads quinto's settings. Credentials live outside the
// working tree — either in the environment or in ~/.config/quinto/config —
// so there is never a token inside the repo for a stray `git add` to catch.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// Site is the full GoatCounter host, e.g. "example.goatcounter.com".
	Site string
	// Token is an API token with Export permission and nothing more.
	Token string
}

// NamedSite is one configured site: the unsectioned default, or a bracketed
// [name] section. Name is what `--site` and `quinto sites` address it by —
// derived from the host's first label for the default, taken verbatim from
// the bracket for a named section.
type NamedSite struct {
	Name  string
	Site  string
	Token string
	// IsDefault is true for the config's unsectioned entry — the one bare
	// `quinto` (no --site) resolves to, and the one that keeps `quinto.db`
	// as its database file rather than getting a name-derived one.
	IsDefault bool
}

// Path returns the config file location, honouring XDG_CONFIG_HOME.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locating home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "quinto", "config"), nil
}

// Sites returns every configured site: the default first (if the file
// configures one), using its host's first label as a display name, then
// named sections in file order. An absent config file is not an error — it
// returns an empty slice, the same as today's "no config, env-only" case.
func Sites() ([]NamedSite, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	f, err := parseFile(path)
	if err != nil {
		return nil, err
	}

	var out []NamedSite
	if f.Default.Site != "" || f.Default.Token != "" {
		d := f.Default
		d.IsDefault = true
		out = append(out, d)
	}
	return append(out, f.Named...), nil
}

// Load resolves credentials for site ("" selects the default, unsectioned
// entry). Environment variables override the default — so a one-off site
// needs no config edit — but never a named site: mixing --site with an env
// override would fetch one site's data into another's database file, which
// is exactly what per-site database files exist to prevent.
func Load(site string) (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	f, err := parseFile(path)
	if err != nil {
		return Config{}, err
	}

	envSite := os.Getenv("QUINTO_GOATCOUNTER_SITE")
	envToken := os.Getenv("QUINTO_GOATCOUNTER_TOKEN")

	var c Config
	switch {
	case site != "" && (envSite != "" || envToken != ""):
		return c, fmt.Errorf(
			"--site %q can't be combined with QUINTO_GOATCOUNTER_SITE/QUINTO_GOATCOUNTER_TOKEN — "+
				"env vars point at a one-off site's credentials, --site picks a named one from %s, "+
				"and mixing them would land one site's data in another's database file. "+
				"Unset the env vars, or drop --site.", site, path)

	case site == "":
		c = Config{Site: f.Default.Site, Token: f.Default.Token}
		if envSite != "" {
			c.Site = envSite
		}
		if envToken != "" {
			c.Token = envToken
		}

	default:
		found := false
		for _, ns := range append([]NamedSite{f.Default}, f.Named...) {
			if ns.Name == site {
				c = Config{Site: ns.Site, Token: ns.Token}
				found = true
				break
			}
		}
		if !found {
			return c, fmt.Errorf("no such site %q; configured: %s", site, siteNames(f))
		}
	}

	if c.Site == "" || c.Token == "" {
		return c, fmt.Errorf(
			"missing GoatCounter credentials: set site and token in %s, or export "+
				"QUINTO_GOATCOUNTER_SITE and QUINTO_GOATCOUNTER_TOKEN.\n"+
				"Create a token at https://<your-site>.goatcounter.com/user/api with "+
				"the Export permission only", path)
	}
	return c, nil
}

func siteNames(f parsedFile) string {
	var names []string
	if f.Default.Name != "" {
		names = append(names, f.Default.Name)
	}
	for _, ns := range f.Named {
		names = append(names, ns.Name)
	}
	if len(names) == 0 {
		return "(none configured)"
	}
	return strings.Join(names, ", ")
}

type parsedFile struct {
	Default NamedSite // the unsectioned entry — Name is derived, not stored
	Named   []NamedSite
}

// parseFile parses a "key = value" file, with optional "[name]" section
// headers. Lines before the first section belong to the default site; a
// missing file is not an error, the same as before sections existed.
func parseFile(path string) (parsedFile, error) {
	var f parsedFile

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return f, fmt.Errorf("opening %s: %w", path, err)
	}
	defer file.Close()

	var current *NamedSite // nil while reading the default's unsectioned lines
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				return f, fmt.Errorf("%s: a [section] needs a name", path)
			}
			f.Named = append(f.Named, NamedSite{Name: name})
			current = &f.Named[len(f.Named)-1]
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		target := &f.Default
		if current != nil {
			target = current
		}
		switch strings.TrimSpace(key) {
		case "site":
			target.Site = strings.TrimSpace(value)
		case "token":
			target.Token = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return f, err
	}

	if f.Default.Site != "" {
		f.Default.Name = firstLabel(f.Default.Site)
	}

	seen := map[string]bool{}
	if f.Default.Name != "" {
		seen[f.Default.Name] = true
	}
	for _, ns := range f.Named {
		if seen[ns.Name] {
			return f, fmt.Errorf("%s: two sites named %q", path, ns.Name)
		}
		seen[ns.Name] = true
	}

	return f, nil
}

// firstLabel derives a short display name from a host, e.g.
// "tillvonkrueger.goatcounter.com" -> "tillvonkrueger".
func firstLabel(host string) string {
	if i := strings.Index(host, "."); i > 0 {
		return host[:i]
	}
	return host
}
