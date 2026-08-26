package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// The table the spec asks for: one case per operator-owned decision per
// layout. It asserts both halves of each — the resulting value, and whether
// the load warned about it. auth.enabled is the only setting a site root
// still FORCES (FEAT-157 moved the git switch and the release branch out of
// basil.yaml entirely); data_dir is the anchor it ignores.
func TestOperatorOwnedSettings(t *testing.T) {
	// A minimal, loadable config; the block under test is appended.
	const base = "server:\n  host: example.com\n  port: 8080\nsite:\n  path: ./site\n"

	cases := []struct {
		name      string
		yaml      string
		siteRoot  bool
		wantAuth  bool
		wantWarns []string
		// wantDataDir, when set, is checked relative to the layout root.
		wantDataDir string
	}{
		// --- site root: auth.enabled --------------------------------------
		{
			name:     "site root, auth omitted",
			yaml:     base,
			siteRoot: true, wantAuth: true,
		},
		{
			name:     "site root, auth explicitly true",
			yaml:     base + "auth:\n  enabled: true\n",
			siteRoot: true, wantAuth: true,
		},
		{
			name:     "site root, auth explicitly false",
			yaml:     base + "auth:\n  enabled: false\n",
			siteRoot: true, wantAuth: true,
			wantWarns: []string{"auth.enabled"},
		},
		// --- site root: data_dir is the operator's, ignored + warned ------
		{
			name:     "site root, data_dir elsewhere",
			yaml:     base + "data_dir: /var/lib/elsewhere\n",
			siteRoot: true, wantAuth: true,
			wantWarns:   []string{"data_dir"},
			wantDataDir: DataDirName,
		},
		{
			name:     "site root, data_dir spelling the convention",
			yaml:     base + "data_dir: ./data\n",
			siteRoot: true, wantAuth: true,
			wantDataDir: DataDirName, // agreeing is silent
		},
		{
			name:     "site root, data_dir omitted",
			yaml:     base,
			siteRoot: true, wantAuth: true,
			wantDataDir: DataDirName,
		},
		// --- legacy layout: nothing is forced, nothing is warned ----------
		{
			name:     "legacy, auth omitted",
			yaml:     base,
			wantAuth: false, // the shipped default, untouched
		},
		{
			name:     "legacy, auth explicitly true",
			yaml:     base + "auth:\n  enabled: true\n",
			wantAuth: true,
		},
		{
			name:     "legacy, auth explicitly false",
			yaml:     base + "auth:\n  enabled: false\n",
			wantAuth: false,
		},
		{
			name:        "legacy, data_dir is the operator speaking and is honoured",
			yaml:        base + "data_dir: ./state\n",
			wantAuth:    false,
			wantDataDir: "state",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var path, root string
			if tc.siteRoot {
				root, _ = writeSiteRoot(t, tc.yaml)
				path = filepath.Join(root, CurrentLinkName, ConfigFileName)
			} else {
				root = writeLegacyProject(t, tc.yaml)
				path = filepath.Join(root, ConfigFileName)
			}

			cfg, err := Load(path, func(string) string { return "" })
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Auth.Enabled != tc.wantAuth {
				t.Errorf("auth.enabled = %v, want %v", cfg.Auth.Enabled, tc.wantAuth)
			}
			if tc.wantDataDir != "" {
				want := filepath.Join(root, tc.wantDataDir)
				if cfg.DataDir != want {
					t.Errorf("data_dir = %q, want %q", cfg.DataDir, want)
				}
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

// Retired keys warn and never fail, in either layout — a clone carrying a
// stale committed key is exactly the case the warning exists for.
func TestRetiredKeysWarnNeverFail(t *testing.T) {
	const base = "server:\n  host: example.com\n  port: 8080\nsite:\n  path: ./site\n"

	cases := []struct {
		name     string
		yaml     string
		siteRoot bool
		want     []string
	}{
		{"neither key", base, true, nil},
		{"deploy.branch on a site root", base + "deploy:\n  branch: shipping\n", true, []string{"deploy.branch"}},
		{"git.enabled on a site root", base + "git:\n  enabled: false\n", true, []string{"git.enabled"}},
		{"git.enabled: true still warns", base + "git:\n  enabled: true\n", true, []string{"git.enabled"}},
		{"both, in a clone", base + "deploy:\n  branch: main\ngit:\n  enabled: false\n", false,
			[]string{"deploy.branch", "git.enabled"}},
		{"deploy.keep is not retired", base + "deploy:\n  keep: 9\n", true, nil},
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
				t.Fatalf("Load: %v — a removed key must never block a load", err)
			}

			got := ReleaseWarnings(cfg)
			if len(got) != len(tc.want) {
				t.Fatalf("retired-key warnings = %v, want %d naming %v", got, len(tc.want), tc.want)
			}
			for _, key := range tc.want {
				if !containsSubstring(got, key) {
					t.Errorf("warnings %v name no retired %s", got, key)
				}
			}
			// Every retired-key warning also reaches the startup channel.
			for _, w := range got {
				if !containsSubstring(Warnings(cfg), w) {
					t.Errorf("Warnings() omits the retired-key warning %q", w)
				}
			}
		})
	}
}

// The replacement is named in the message: a warning that only says "gone"
// leaves the operator no way forward.
func TestRetiredKeyWarningsNameTheFix(t *testing.T) {
	const yaml = "server:\n  host: example.com\nsite:\n  path: ./site\ndeploy:\n  branch: shipping\ngit:\n  enabled: false\n"
	root, _ := writeSiteRoot(t, yaml)
	cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	warns := strings.Join(ReleaseWarnings(cfg), "\n")
	repo := filepath.Join(root, BareRepoName)
	for _, want := range []string{
		"git -C " + repo + " symbolic-ref HEAD refs/heads/",
		"git -C " + repo + " config basil.gitEnabled false",
	} {
		if !strings.Contains(warns, want) {
			t.Errorf("retired-key warnings do not name the fix %q:\n%s", want, warns)
		}
	}
}

// The question initGit asks before it tells anyone the /.git endpoint is
// gone. It used to ask the filesystem — "is the project directory a git
// repository?" — which stopped meaning anything once FEAT-156 made a plain
// `basil --init` write exactly that shape. The first case here is the
// regression: the folder every new local project starts life as, which must
// be silent.
func TestRequestedGitEndpoint(t *testing.T) {
	const base = "server:\n  host: example.com\n  port: 8080\nsite:\n  path: ./site\n"

	cases := []struct {
		name string
		yaml string
		want bool
	}{
		{"local init: no git block at all", base, false},
		{"switched off on purpose", base + "git:\n  enabled: false\n", false},
		{"switched on: this operator used the endpoint", base + "git:\n  enabled: true\n", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeLegacyProject(t, tc.yaml)
			cfg, err := Load(filepath.Join(root, ConfigFileName), func(string) string { return "" })
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := RequestedGitEndpoint(cfg); got != tc.want {
				t.Errorf("RequestedGitEndpoint = %v, want %v", got, tc.want)
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
			if strings.HasPrefix(w, key) {
				out = append(out, w)
			}
		}
	}
	return out
}

// containsKey matches on the key alone, not key+":": an operator-owned
// warning names its key first but need not quote a value after it —
// data_dir's deliberately does not (the value is env-interpolated and must
// not reach the log).
func containsKey(warnings []string, key string) bool {
	for _, w := range warnings {
		if strings.HasPrefix(w, key) {
			return true
		}
	}
	return false
}

func containsSubstring(warnings []string, want string) bool {
	for _, w := range warnings {
		if strings.Contains(w, want) {
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
