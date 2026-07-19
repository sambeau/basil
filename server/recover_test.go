package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverMiddlewarePanicBecomes500(t *testing.T) {
	var stderr bytes.Buffer
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	h := newRecoverMiddleware(panicking, &stderr, false)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req) // must not panic out of the handler

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Internal Server Error") {
		t.Fatalf("expected error body, got %q", body)
	}
	if body := rec.Body.String(); strings.Contains(body, "boom") {
		t.Fatalf("production mode should not leak panic detail, got %q", body)
	}
	if !strings.Contains(stderr.String(), "panic recovered") {
		t.Fatalf("expected panic to be logged, got %q", stderr.String())
	}
}

func TestRecoverMiddlewareDevModeLeaksDetail(t *testing.T) {
	var stderr bytes.Buffer
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	h := newRecoverMiddleware(panicking, &stderr, true)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("dev mode should include panic detail, got %q", rec.Body.String())
	}
}

func TestRecoverMiddlewarePassesThroughNormalResponses(t *testing.T) {
	var stderr bytes.Buffer
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, "hello")
	})

	h := newRecoverMiddleware(ok, &stderr, false)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected pass-through status 418, got %d", rec.Code)
	}
	if rec.Body.String() != "hello" {
		t.Fatalf("expected pass-through body, got %q", rec.Body.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("no panic should be logged for a normal response, got %q", stderr.String())
	}
}
