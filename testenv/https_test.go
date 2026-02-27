package testenv

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestHTTPS_ServeJSON(t *testing.T) {
	env := Start(t, WithHTTPS())

	type payload struct {
		Name string `json:"name"`
	}
	env.ServeJSON("/data.json", payload{Name: "basil"})

	resp, err := env.HTTPSClient.Get(env.HTTPSURL + "/data.json")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var got payload
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Name != "basil" {
		t.Errorf("expected name=basil, got %q", got.Name)
	}
}

func TestHTTPS_ServeText(t *testing.T) {
	env := Start(t, WithHTTPS())
	env.ServeText("/hello.txt", "hello world")

	resp, err := env.HTTPSClient.Get(env.HTTPSURL + "/hello.txt")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("expected body %q, got %q", "hello world", string(body))
	}
}

func TestHTTPS_ServeError(t *testing.T) {
	env := Start(t, WithHTTPS())
	env.ServeError("/missing", http.StatusNotFound)

	resp, err := env.HTTPSClient.Get(env.HTTPSURL + "/missing")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHTTPS_ServeRedirect(t *testing.T) {
	env := Start(t, WithHTTPS())
	env.ServeText("/final", "landed")
	env.ServeRedirect("/redirect", "/final", http.StatusFound)

	resp, err := env.HTTPSClient.Get(env.HTTPSURL + "/redirect")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// The default http.Client follows redirects, so we should land on /final.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after redirect, got %d", resp.StatusCode)
	}
	if resp.Request.URL.Path != "/final" {
		t.Errorf("expected final URL path /final, got %q", resp.Request.URL.Path)
	}
}

func TestHTTPS_TLSRequired(t *testing.T) {
	env := Start(t, WithHTTPS())
	env.ServeText("/tls-check", "ok")

	// http.DefaultClient does not trust the self-signed cert; the request must fail.
	_, err := http.DefaultClient.Get(env.HTTPSURL + "/tls-check")
	if err == nil {
		t.Error("expected TLS error with http.DefaultClient, got nil")
	}
}
