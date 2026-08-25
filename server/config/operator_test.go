package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// The table the spec asks for: one case per forced key per layout, across
// omitted / explicitly true / explicitly false. It asserts both halves of the
// decision — the resulting value, and whether the load warned about it.
func TestOperatorOwnedSettings(t *testing.T) {
	// A minimal, loadable config; the block under test is appended.
	const base = "server:\n  host: example.com\n  port: 8080\nsite:\n  path: ./site\n"

	cases := []struct {
		name      string
		yaml      string
		siteRoot  bool
		wantGit   bool
		wantAuth  bool
		wantWarns []string
	}{
		// --- site root: git.enabled ---------------------------------------
		{
			name:     "site root, git omitted",
			yaml:     base,
			siteRoot: true,
			wantGit:  true, wantAuth: true,
		},
		{
			name:     "site root, git explicitly true",
			yaml:     base + "git:\n  enabled: true\n",
			siteRoot: true,
			wantGit:  true, wantAuth: true,
		},
		{
			name:     "site root, git explicitly false",
			yaml:     base + "git:\n  enabled: false\n",
			siteRoot: true,
			wantGit:  true, wantAuth: true,
			wantWarns: []string{"git.enabled"},
		},
		// --- site root: auth.enabled --------------------------------------
		{
			name:     "site root, auth explicitly true",
			yaml:     base + "auth:\n  enabled: true\n",
			siteRoot: true,
			wantGit:  true, wantAuth: true,
		},
		{
			name:     "site root, auth explicitly false",
			yaml:     base + "auth:\n  enabled: false\n",
			siteRoot: true,
			wantGit:  true, wantAuth: true,
			wantWarns: []string{"auth.enabled"},
		},
		{
			name:     "site root, both explicitly false",
			yaml:     base + "git:\n  enabled: false\nauth:\n  enabled: false\n",
			siteRoot: true,
			wantGit:  true, wantAuth: true,
			wantWarns: []string{"git.enabled", "auth.enabled"},
		},
		// --- legacy layout: nothing is forced, nothing is warned ----------
		{
			name:    "legacy, omitted",
			yaml:    base,
			wantGit: true, wantAuth: false, // the shipped defaults, untouched
		},
		{
			name:    "legacy, git explicitly true",
			yaml:    base + "git:\n  enabled: true\n",
			wantGit: true, wantAuth: false,
		},
		{
			name:    "legacy, git explicitly false",
			yaml:    base + "git:\n  enabled: false\n",
			wantGit: false, wantAuth: false,
		},
		{
			name:    "legacy, auth explicitly true",
			yaml:    base + "auth:\n  enabled: true\n",
			wantGit: true, wantAuth: true,
		},
		{
			name:    "legacy, auth explicitly false",
			yaml:    base + "auth:\n  enabled: false\n",
			wantGit: true, wantAuth: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			if tc.siteRoot {
				root, _ := writeSiteRoot(t, tc.yaml)
				path = filepath.Join(root, CurrentLinkName, ConfigFileName)
			} else {
				path = filepath.Join(writeLegacyProject(t, tc.yaml), ConfigFileName)
			}

			cfg, err := Load(path, func(string) string { return "" })
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Git.Enabled != tc.wantGit {
				t.Errorf("git.enabled = %v, want %v", cfg.Git.Enabled, tc.wantGit)
			}
			if cfg.Auth.Enabled != tc.wantAuth {
				t.Errorf("auth.enabled = %v, want %v", cfg.Auth.Enabled, tc.wantAuth)
			}

			overrides := operatorWarnings(Warnings(cfg))
			if len(overrides) != len(tc.wantWarns) {
				t.Fatalf("operator-owned warnings = %v, want %d naming %v", overrides, len(tc.wantWarns), tc.wantWarns)
			}
			for _, key := range tc.wantWarns {
				if !containsKey(overrides, key) {
					t.Errorf("warnings %v name no override of %s", overrides, key)
				}
			}
		})
	}
}

// operatorWarnings picks the operator-owned override warnings out of the full
// warning list, so unrelated warnings (routes, HTTPS) cannot mask a missing
// one or fake a present one.
func operatorWarnings(all []string) []string {
	var out []string
	for _, w := range all {
		for _, key := range OperatorOwnedKeys() {
			if strings.HasPrefix(w, key+":") {
				out = append(out, w)
			}
		}
	}
	return out
}

func containsKey(warnings []string, key string) bool {
	for _, w := range warnings {
		if strings.HasPrefix(w, key+":") {
			return true
		}
	}
	return false
}

func TestIsLocalHost(t *testing.T) {
	local := []string{"localhost", "127.0.0.1", "::1", "mysite.local"}
	public := []string{"example.com", "mysite.example.com", "192.0.2.10", ""}

	for _, h := range local {
		if !IsLocalHost(h) {
			t.Errorf("IsLocalHost(%q) = false, want true", h)
		}
	}
	for _, h := range public {
		if IsLocalHost(h) {
			t.Errorf("IsLocalHost(%q) = true, want false", h)
		}
	}
}
