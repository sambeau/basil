package server

import (
	"bytes"
	"crypto/tls"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/basil/server/config"
	"golang.org/x/crypto/acme/autocert"
)

// probeServer builds the smallest Server the certificate probe needs: a data
// root for the failure marker, and somewhere to write logs.
func probeServer(t *testing.T) (*Server, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfg := config.Defaults()
	cfg.ReleaseDir = t.TempDir()
	cfg.DataDir = t.TempDir()
	cfg.Server.Host = "mysite.example.com"
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &Server{config: cfg, stdout: stdout, stderr: stderr}, stdout, stderr
}

func closedReady() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// A failed startup probe must not be repeated on the next start. Let's
// Encrypt caps failed authorizations at 5 per hostname per hour, and a
// server under Restart=always with broken DNS restarts far faster than that.
func TestCertificateProbeBacksOffAfterAFailure(t *testing.T) {
	s, _, stderr := probeServer(t)

	var calls int
	get := func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		calls++
		return nil, errors.New("acme: authorization failed")
	}

	s.certificateProbe(s.config.Server.Host, closedReady(), get)
	if calls != 1 {
		t.Fatalf("first probe made %d requests, want 1", calls)
	}
	first := stderr.String()
	if !strings.Contains(first, "DNS") || !strings.Contains(first, "port 80") {
		t.Errorf("the failure was not diagnosed:\n%s", first)
	}
	if _, err := os.Stat(filepath.Join(s.config.DataDir, ".acme-probe-failed")); err != nil {
		t.Fatalf("no failure marker was recorded: %v", err)
	}

	// The restart: same diagnosis, no second ACME request.
	stderr.Reset()
	s.certificateProbe(s.config.Server.Host, closedReady(), get)
	if calls != 1 {
		t.Errorf("the restart asked the ACME server again (%d requests)", calls)
	}
	second := stderr.String()
	if !strings.Contains(second, "not asking") {
		t.Errorf("the restart did not explain why it is not asking:\n%s", second)
	}
	if !strings.Contains(second, "DNS") || !strings.Contains(second, "port 80") {
		t.Errorf("the restart lost the diagnosis:\n%s", second)
	}
}

// A failure older than the cooldown must be retried, and a success must
// clear the marker.
func TestCertificateProbeRetriesAfterTheCooldown(t *testing.T) {
	s, stdout, _ := probeServer(t)
	marker := filepath.Join(s.config.DataDir, ".acme-probe-failed")
	stale := time.Now().Add(-2 * certificateFailureCooldown).Format(time.RFC3339)
	if err := os.WriteFile(marker, []byte(stale), 0600); err != nil {
		t.Fatal(err)
	}

	var calls int
	get := func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		calls++
		return &tls.Certificate{}, nil
	}
	s.certificateProbe(s.config.Server.Host, closedReady(), get)

	if calls != 1 {
		t.Errorf("a stale failure marker suppressed the probe (%d requests)", calls)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("a successful probe left the failure marker behind")
	}
	if !strings.Contains(stdout.String(), "certificate ready") {
		t.Errorf("the success was not reported:\n%s", stdout.String())
	}
}

// The probe is bounded: a hung ACME call must not hold the diagnosis back
// for ever.
func TestCertificateProbeTimesOut(t *testing.T) {
	s, _, stderr := probeServer(t)

	old := certificateProbeTimeout
	certificateProbeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { certificateProbeTimeout = old })

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	get := func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		<-release
		return nil, errors.New("too late")
	}

	done := make(chan struct{})
	go func() {
		s.certificateProbe(s.config.Server.Host, closedReady(), get)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the probe did not return when its timeout expired")
	}

	if !strings.Contains(stderr.String(), "timed out") {
		t.Errorf("the timeout was not reported:\n%s", stderr.String())
	}
}

// A certificate already in the cache needs no issuance request at startup.
// Asking anyway would spend rate limit on every restart.
func TestObtainCertificateSkipsWhenTheCacheHasTheCertificate(t *testing.T) {
	s, stdout, stderr := probeServer(t)
	cacheDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cacheDir, s.config.Server.Host), []byte("cached"), 0600); err != nil {
		t.Fatal(err)
	}
	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(cacheDir),
	}

	s.obtainCertificate(manager, closedReady())

	if !strings.Contains(stdout.String(), "already in the cache") {
		t.Errorf("a cached certificate was not recognised:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("a cached certificate produced errors:\n%s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(s.config.DataDir, ".acme-probe-failed")); err == nil {
		t.Error("a cached certificate recorded a failure")
	}
}
