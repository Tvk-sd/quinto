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

// Load reads the config file, then lets environment variables override it.
// Env wins so a shell can point quinto at a different site without editing
// files — useful for testing against a second account.
func Load() (Config, error) {
	var c Config

	path, err := Path()
	if err != nil {
		return c, err
	}
	if err := c.loadFile(path); err != nil {
		return c, err
	}

	if v := os.Getenv("QUINTO_GOATCOUNTER_SITE"); v != "" {
		c.Site = v
	}
	if v := os.Getenv("QUINTO_GOATCOUNTER_TOKEN"); v != "" {
		c.Token = v
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

// loadFile parses a minimal "key = value" file. A missing file is not an
// error — the environment alone is a valid way to configure quinto.
func (c *Config) loadFile(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "site":
			c.Site = strings.TrimSpace(value)
		case "token":
			c.Token = strings.TrimSpace(value)
		}
	}
	return scanner.Err()
}
