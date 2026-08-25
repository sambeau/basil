package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

// writeReleaseConfig makes a directory holding one basil.yaml, standing in
// for a release directory.
func writeReleaseConfig(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A public listener, the state the warning protects.
const publicRelease = `server:
  host: mysite.example.com
  port: 443
  https:
    auto: true
site:
  path: ./site
`

func TestListenerChanges(t *testing.T) {
	cases := []struct {
		name      string
		active    string
		candidate string
		want      []string // substrings, one per expected line
	}{
		{
			name:      "identical configs are silent",
			active:    publicRelease,
			candidate: publicRelease,
		},
		{
			name:   "host change warns",
			active: publicRelease,
			candidate: `server:
  host: elsewhere.example.com
  port: 443
  https:
    auto: true
site:
  path: ./site
`,
			want: []string{"server.host: mysite.example.com → elsewhere.example.com"},
		},
		{
			name:   "port change warns",
			active: publicRelease,
			candidate: `server:
  host: mysite.example.com
  port: 8080
  https:
    auto: true
site:
  path: ./site
`,
			want: []string{"server.port: 443 → 8080"},
		},
		{
			// Dropping a block that named a certificate is a real change:
			// the site falls back to Let's Encrypt.
			name: "dropping an https block that carried a certificate warns",
			active: `server:
  host: mysite.example.com
  port: 443
  https:
    cert: ./certs/site.pem
    key: ./certs/site.key
site:
  path: ./site
`,
			candidate: publicRelease,
			want:      []string{"https: manual certificate (site.pem) → automatic (Let's Encrypt)"},
		},
		{
			// Turning TLS off is the hazard the https comparison exists for.
			name:   "disabling https warns",
			active: publicRelease,
			candidate: `server:
  host: mysite.example.com
  port: 443
  https:
    auto: false
site:
  path: ./site
`,
			want: []string{"https: automatic (Let's Encrypt) → none (plain HTTP)"},
		},
		{
			// https.auto defaults to true, so omitting the block changes
			// nothing about how TLS is obtained — and a warning about it
			// would fire on every release from a config that never had one.
			name:   "omitting an https block that only said auto is silent",
			active: publicRelease,
			candidate: `server:
  host: mysite.example.com
  port: 443
site:
  path: ./site
`,
		},
		{
			// The graduation accident: a local config deployed onto a public
			// server. The host and port changes are the outage.
			name:   "the graduation accident warns about host and port",
			active: publicRelease,
			candidate: `server:
  host: localhost
  port: 8080
site:
  path: ./site
`,
			want: []string{"server.host: mysite.example.com → localhost", "server.port: 443 → 8080"},
		},
		{
			// A developer's own machine: every listener value there is
			// their business, so nothing is reported.
			name: "a localhost active release is silent",
			active: `server:
  host: localhost
  port: 8080
site:
  path: ./site
`,
			candidate: publicRelease,
		},
		{
			// Validate owns broken configs; reporting them twice in two
			// voices helps nobody.
			name:      "a broken candidate config is silent",
			active:    publicRelease,
			candidate: "server:\n  port: [not, a, port]\n",
		},
		{
			name:      "a broken active config is silent",
			active:    "server:\n  port: [not, a, port]\n",
			candidate: publicRelease,
		},
		{
			// An active config with no host is not a public server.
			name: "an active release with no host is silent",
			active: `server:
  port: 8080
site:
  path: ./site
`,
			candidate: publicRelease,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			active := writeReleaseConfig(t, tc.active)
			candidate := writeReleaseConfig(t, tc.candidate)

			got := ListenerChanges(active, candidate)
			if len(got) != len(tc.want) {
				t.Fatalf("ListenerChanges = %v, want %d line(s) naming %v", got, len(tc.want), tc.want)
			}
			joined := strings.Join(got, "\n")
			for _, want := range tc.want {
				if !strings.Contains(joined, want) {
					t.Errorf("ListenerChanges = %v, want a line containing %q", got, want)
				}
			}
		})
	}
}

// A missing release directory has nothing to compare, and must not panic or
// invent a change.
func TestListenerChangesWithMissingRelease(t *testing.T) {
	active := writeReleaseConfig(t, publicRelease)
	if got := ListenerChanges(active, filepath.Join(t.TempDir(), "gone")); got != nil {
		t.Errorf("ListenerChanges against a missing release = %v, want nil", got)
	}
	if got := ListenerChanges("", active); got != nil {
		t.Errorf("ListenerChanges with no active release = %v, want nil", got)
	}
}
