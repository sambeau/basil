package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Operator-owned settings (FEAT-156, FEAT-157).
//
// basil.yaml ships inside the release, so a deployed config can describe the
// server that serves it. Some settings must not be describable that way: a
// release that arrived over the git endpoint could otherwise disable the
// authentication that guards it — bricking the deploy mechanism from inside a
// deploy, with recovery requiring shell access to the box.
//
// So on a site root (and only there) auth.enabled is operator-owned:
// whatever the release says, it is on. The data root joins it as an anchor
// the release may not move (ResolveAnchors), and the two facts a release must
// not own at all — which branch publishes, and whether /.git is served —
// left basil.yaml entirely in FEAT-157: they live in site.git, as HEAD and
// the basil.gitEnabled git-config key. In the legacy single-directory layout
// nothing is forced — a local dev server with no accounts is correct, and
// that is the shape `basil --init` now writes.
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
	Auth struct {
		Enabled *bool `yaml:"enabled"`
	} `yaml:"auth"`
	Deploy struct {
		Branch *string `yaml:"branch"`
	} `yaml:"deploy"`
	Git struct {
		Enabled *bool `yaml:"enabled"`
	} `yaml:"git"`
}

// enforceOperatorOwned forces the operator-owned settings on when the
// site-root layout is active, and records a warning for each one the config
// explicitly disabled. probe is the raw config source after env
// interpolation, parsed once by the caller.
//
// Omission is silent: a local config that simply has no git: block is the
// normal case after FEAT-156's init split, and warning about it would fire on
// every start of every graduated site.
func enforceOperatorOwned(cfg *Config, probe *operatorProbe) {
	if cfg.SiteRoot == "" {
		return
	}

	forced := []struct {
		key      string
		explicit *bool
		field    *bool
		why      string
	}{
		{"auth.enabled", probe.Auth.Enabled, &cfg.Auth.Enabled,
			"pushes are authenticated; a release cannot switch that off"},
	}

	for _, f := range forced {
		*f.field = true
		if f.explicit != nil && !*f.explicit {
			cfg.operatorOverrides = append(cfg.operatorOverrides, fmt.Sprintf(
				"%s: false in this release's %s is ignored on a site root — %s", f.key, ConfigFileName, f.why))
		}
	}
}

// noteRetiredKeys records one warning per removed key the config still
// carries. The struct fields are gone, so the raw-YAML probe is the only
// reader left — and it is deliberately the only consequence: a key nobody
// reads must never block a load, on a server or in the clone that pulls the
// file. Layout makes no difference either; the stale key is in the file
// wherever it is read.
func noteRetiredKeys(cfg *Config, probe *operatorProbe) {
	repo := "<site root>/" + BareRepoName
	if cfg.SiteRoot != "" {
		repo = filepath.Join(cfg.SiteRoot, BareRepoName)
	}

	if probe.Deploy.Branch != nil {
		cfg.retiredKeys = append(cfg.retiredKeys, fmt.Sprintf(
			"deploy.branch is no longer read — the release branch is %s's HEAD; change it with: git -C %s symbolic-ref HEAD refs/heads/<branch>",
			BareRepoName, repo))
	}
	if probe.Git.Enabled != nil {
		cfg.retiredKeys = append(cfg.retiredKeys, fmt.Sprintf(
			"git.enabled is no longer read — the git endpoint is controlled on the server with: git -C %s config basil.gitEnabled false",
			repo))
	}
}

// ReleaseWarnings returns everything loading this config decided to ignore:
// the operator-owned settings a site root overrode, and the removed keys the
// file still carries. Warnings includes them too; this is the narrow channel
// for the places that report on a config they are not themselves serving —
// a live release swap, and `basil publish`, where the developer's clone is
// the one place a stale committed key would otherwise never be seen.
func ReleaseWarnings(cfg *Config) []string {
	out := make([]string, 0, len(cfg.operatorOverrides)+len(cfg.retiredKeys))
	out = append(out, cfg.operatorOverrides...)
	return append(out, cfg.retiredKeys...)
}

// OperatorOwnedKeys names the settings a site root decides for itself
// whatever the release says, for docs and error text. It is not a config
// surface: nothing reads it to decide anything.
//
// Both entries are operator-owned in the same sense — the release's value is
// ignored and the server's stands — though by different mechanisms:
// auth.enabled is forced true here, and data_dir is pinned to the
// conventional root in ResolveAnchors. Listing only one of them made this
// disagree with the docs and left callers hand-appending the other.
func OperatorOwnedKeys() []string {
	return []string{"auth.enabled", "data_dir"}
}

// IsLocalHost reports whether a hostname names this machine rather than a
// public address. It decides whether a server is "public" for the deploy
// gate in server/deploy, so it lives here where both config consumers and
// the deploy package can reach it without an import cycle.
func IsLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local")
}
