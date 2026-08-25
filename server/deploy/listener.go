package deploy

import (
	"fmt"
	"path/filepath"

	"github.com/sambeau/basil/server/config"
)

// ListenerChanges reports, in human-readable lines, every way a candidate
// release's config would move the listener the active release is serving on:
// server.host, server.port, and the https block.
//
// Like Unformatted this is a WARNING channel, deliberately NOT part of
// Validate: renaming a site or moving its port over git is legitimate, and
// ValidationError is the fatal channel — routing this through it would reject
// releases the operator meant to deploy. The hazard it exists for is the
// graduation accident (FEAT-156): a local config says host localhost, port
// 8080, no https:, and deploying it onto a public server would take down the
// site AND the git endpoint at the next restart, leaving no way in but a
// shell. Naming it at push time puts the mistake one commit from reverted.
//
// The check applies only when the ACTIVE release is a public server: on a
// localhost site every listener value is a developer's business and a warning
// would be noise. A config that will not load produces no lines at all —
// Validate already owns broken configs, and reporting the same file twice in
// two voices helps nobody. nil means nothing to say.
func ListenerChanges(activeReleaseDir, candidateReleaseDir string) []string {
	active, ok := loadReleaseConfig(activeReleaseDir)
	if !ok {
		return nil
	}
	candidate, ok := loadReleaseConfig(candidateReleaseDir)
	if !ok {
		return nil
	}

	// "Public" is decided from the release that is live now: it is the one
	// whose loss would be an outage.
	if !isPublicListener(active) {
		return nil
	}

	var changes []string
	if active.Server.Host != candidate.Server.Host {
		changes = append(changes, fmt.Sprintf("server.host: %s → %s", quoteEmpty(active.Server.Host), quoteEmpty(candidate.Server.Host)))
	}
	if active.Server.Port != candidate.Server.Port {
		changes = append(changes, fmt.Sprintf("server.port: %d → %d", active.Server.Port, candidate.Server.Port))
	}
	if a, c := httpsSummary(active.Server.HTTPS), httpsSummary(candidate.Server.HTTPS); a != c {
		changes = append(changes, fmt.Sprintf("https: %s → %s", a, c))
	}
	return changes
}

// isPublicListener reports whether a release is serving the world rather than
// one developer's machine. A named host that is not this machine says so
// outright; so does a manual certificate with no host at all, which is a
// legitimate production shape — the certificate names the site and the host
// arrives from DNS or a proxy in front. An explicitly local host stays local
// whatever certificate it carries: every listener value on a developer's own
// machine is that developer's business, which is the whole reason the warning
// is scoped this way.
func isPublicListener(cfg *config.Config) bool {
	if cfg.Server.Host != "" {
		return !config.IsLocalHost(cfg.Server.Host)
	}
	return cfg.Server.HTTPS.Cert != ""
}

// loadReleaseConfig loads a release's basil.yaml for COMPARISON — which is
// not quite the way the server loads it to run. The environment is read as
// empty (a null getenv) on purpose: the lines this feeds are printed into a
// remote pusher's terminal, and `host: ${INTERNAL_NAME}` interpolated with
// the server's real environment would put a secret in that output. The
// comparison needs the shape of the listener, not the resolved value, and an
// unresolved ${VAR} reads the same on both sides.
//
// The second result is false when there is nothing to compare — which is the
// same answer for "no config there" and "config is broken", because both are
// already Validate's business.
func loadReleaseConfig(releaseDir string) (*config.Config, bool) {
	if releaseDir == "" {
		return nil, false
	}
	path := filepath.Join(releaseDir, config.ConfigFileName)
	cfg, err := config.Load(path, nullGetenv)
	if err != nil {
		return nil, false
	}
	return cfg, true
}

// nullGetenv is an environment with nothing in it.
func nullGetenv(string) string { return "" }

// httpsSummary describes how a release obtains its certificate, in the terms
// an operator would use. It compares the SHAPE of the https block rather than
// its fields: the change worth warning about is "this release stops serving
// TLS the way the live one does", and a raw struct comparison would also fire
// on cosmetic differences (an ACME contact address edit) that change no
// listener.
//
// It reads the EFFECTIVE block, after defaults — https.auto defaults to true,
// so a release that merely omits the https: block still serves TLS the same
// way and is correctly silent here. What warns is a release that would really
// change how the site is reached: auto turned off, or a manual certificate
// appearing, disappearing or moving.
func httpsSummary(h config.HTTPSConfig) string {
	switch {
	case h.Cert != "" && h.Key != "":
		return "manual certificate (" + filepath.Base(h.Cert) + ")"
	case h.Auto:
		return "automatic (Let's Encrypt)"
	default:
		return "none (plain HTTP)"
	}
}

// quoteEmpty renders an unset value visibly, so "host removed" cannot read as
// a missing word in the warning line.
func quoteEmpty(v string) string {
	if v == "" {
		return "(unset)"
	}
	return v
}
