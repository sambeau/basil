package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Operator-owned settings (FEAT-156).
//
// basil.yaml ships inside the release, so a deployed config can describe the
// server that serves it. Two settings must not be describable that way: a
// release that arrived over the git endpoint could otherwise disable the git
// endpoint it arrived through, and disable the authentication that guards it
// — bricking the deploy mechanism from inside a deploy, with recovery
// requiring shell access to the box.
//
// So on a site root (and only there) git.enabled and auth.enabled are
// operator-owned: whatever the release says, they are on. In the legacy
// single-directory layout nothing is forced — a local dev server with no git
// endpoint is correct, and that is the shape `basil --init` now writes.
//
// This is the narrow fence, not the general answer to "may a release change
// server settings?" (FEAT-152 Open Question 1 stays open): it covers exactly
// the settings whose loss removes the remote recovery path. Listener settings
// stay deployable and are gated at push time instead (server/deploy).

// operatorProbe reads just enough of the raw YAML to tell "the config set
// this to false" from "the config did not mention it". Config itself cannot
// answer that — Defaults() has already filled both fields in by the time
// yaml.Unmarshal runs, so a false in the struct is indistinguishable from an
// omitted key for auth.enabled (default false). Pointers distinguish them.
type operatorProbe struct {
	Git struct {
		Enabled *bool `yaml:"enabled"`
	} `yaml:"git"`
	Auth struct {
		Enabled *bool `yaml:"enabled"`
	} `yaml:"auth"`
}

// enforceOperatorOwned forces the operator-owned settings on when the
// site-root layout is active, and records a warning for each one the config
// explicitly disabled. data is the config source after env interpolation —
// the same bytes that were unmarshalled into cfg.
//
// Omission is silent: a local config that simply has no git: block is the
// normal case after FEAT-156's init split, and warning about it would fire on
// every start of every graduated site.
func enforceOperatorOwned(cfg *Config, data []byte) {
	if cfg.SiteRoot == "" {
		return
	}

	// A probe that will not parse is not a reason to skip enforcement: the
	// forcing is the safety property, the warning is only its explanation.
	var probe operatorProbe
	_ = yaml.Unmarshal(data, &probe)

	forced := []struct {
		key      string
		explicit *bool
		field    *bool
		why      string
	}{
		{"git.enabled", probe.Git.Enabled, &cfg.Git.Enabled,
			"the git endpoint is how releases arrive; a release cannot switch it off"},
		{"auth.enabled", probe.Auth.Enabled, &cfg.Auth.Enabled,
			"pushes are authenticated; a release cannot switch that off"},
	}

	var overridden []string
	for _, f := range forced {
		*f.field = true
		if f.explicit != nil && !*f.explicit {
			overridden = append(overridden, fmt.Sprintf(
				"%s: false in this release's %s is ignored on a site root — %s", f.key, ConfigFileName, f.why))
		}
	}
	cfg.operatorOverrides = overridden
}

// OperatorOwnedKeys names the settings a site root forces on, for docs and
// error text. It is not a config surface: nothing reads it to decide anything.
func OperatorOwnedKeys() []string {
	return []string{"git.enabled", "auth.enabled"}
}

// IsLocalHost reports whether a hostname names this machine rather than a
// public address. It decides whether a server is "public" for the deploy
// gate in server/deploy, so it lives here where both config consumers and
// the deploy package can reach it without an import cycle.
func IsLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local")
}
