package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/config"
)

// TestAPIParamsFreshnessAcrossRequests verifies that @params and basil/http params
// are correctly resolved fresh for each request, even when modules are cached.
// This is the critical test for determining if ClearModuleCache() can be removed.
func TestAPIParamsFreshnessAcrossRequests(t *testing.T) {
	// Clear module cache once at start to ensure clean state
	evaluator.ClearModuleCache()

	dir := t.TempDir()

	// Create API handler that uses @basil/http params
	// The key test: import at module scope, use in function
	apiPath := filepath.Join(dir, "api.pars")
	apiCode := `let api = import @std/api
let {params} = import @basil/http

export get = api.public(fn(req) {
    let orderBy = params.orderBy ?? "default"
    let page = params.page ?? "1"
    {
        orderBy: orderBy,
        page: page
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiCode), 0644); err != nil {
		t.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/test", Handler: apiPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	stderr := &noopBuffer{}
	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Test case: Multiple requests with different query parameters
	// The module should be cached after request 1, but params should still be fresh
	testCases := []struct {
		name            string
		queryString     string
		expectedOrderBy string
		expectedPage    string
	}{
		{
			name:            "first request - orderBy=name, page=1",
			queryString:     "?orderBy=name&page=1",
			expectedOrderBy: "name",
			expectedPage:    "1",
		},
		{
			name:            "second request - orderBy=age, page=2",
			queryString:     "?orderBy=age&page=2",
			expectedOrderBy: "age",
			expectedPage:    "2",
		},
		{
			name:            "third request - orderBy=date, page=3",
			queryString:     "?orderBy=date&page=3",
			expectedOrderBy: "date",
			expectedPage:    "3",
		},
		{
			name:            "fourth request - no params (test defaults)",
			queryString:     "",
			expectedOrderBy: "default",
			expectedPage:    "1",
		},
		{
			name:            "fifth request - back to first params",
			queryString:     "?orderBy=name&page=1",
			expectedOrderBy: "name",
			expectedPage:    "1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test"+tc.queryString, http.NoBody)
			rec := httptest.NewRecorder()

			srv.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Logf("stderr: %s", stderr.String())
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}

			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			if body["orderBy"] != tc.expectedOrderBy {
				t.Errorf("orderBy: expected %q, got %q", tc.expectedOrderBy, body["orderBy"])
			}
			if body["page"] != tc.expectedPage {
				t.Errorf("page: expected %q, got %q", tc.expectedPage, body["page"])
			}
		})
	}
}

// TestAPIParamsFreshnessWithMethod tests that method from @basil/http is also fresh
func TestAPIParamsFreshnessWithMethod(t *testing.T) {
	evaluator.ClearModuleCache()

	dir := t.TempDir()

	// Create API handler that uses method from @basil/http
	apiPath := filepath.Join(dir, "method_api.pars")
	apiCode := `let api = import @std/api
let {params, method} = import @basil/http

export get = api.public(fn(req) {
    let filter = params.filter ?? "all"
    {
        filter: filter,
        httpMethod: method
    }
})

export post = api.public(fn(req) {
    let filter = params.filter ?? "all"
    {
        filter: filter,
        httpMethod: method
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiCode), 0644); err != nil {
		t.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/method", Handler: apiPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Make several requests and verify method changes correctly
	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/method?filter=active", http.NoBody)
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if body["httpMethod"] != "GET" {
			t.Errorf("httpMethod: expected GET, got %q", body["httpMethod"])
		}
		if body["filter"] != "active" {
			t.Errorf("filter: expected active, got %q", body["filter"])
		}
	})

	t.Run("POST request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/method?filter=pending", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if body["httpMethod"] != "POST" {
			t.Errorf("httpMethod: expected POST, got %q", body["httpMethod"])
		}
		if body["filter"] != "pending" {
			t.Errorf("filter: expected pending, got %q", body["filter"])
		}
	})

	// Another GET to make sure we don't get cached POST values
	t.Run("GET request after POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/method?filter=archived", http.NoBody)
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if body["httpMethod"] != "GET" {
			t.Errorf("httpMethod: expected GET, got %q", body["httpMethod"])
		}
		if body["filter"] != "archived" {
			t.Errorf("filter: expected archived, got %q", body["filter"])
		}
	})
}

// TestPageHandlerParamsFreshness verifies params work correctly in page handlers
// Page handlers already don't call ClearModuleCache(), so this validates the pattern
func TestPageHandlerParamsFreshness(t *testing.T) {
	evaluator.ClearModuleCache()

	dir := t.TempDir()

	// Create a page that uses @params (matches benchmark pattern)
	pagePath := filepath.Join(dir, "search.parsley")
	pageCode := `let query = @params["q"] ?? ""
let page = @params["page"] ?? "1"

<div class="results">
    <span id="query">query</span>
    <span id="page">page</span>
</div>
`
	if err := os.WriteFile(pagePath, []byte(pageCode), 0644); err != nil {
		t.Fatalf("failed to write page: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/search", Handler: pagePath, Type: "page"}},
		Logging: config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	stderr := &noopBuffer{}
	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, stderr)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Test multiple requests with different params
	testCases := []struct {
		query       string
		expectQuery string
		expectPage  string
	}{
		{"?q=hello&page=1", "hello", "1"},
		{"?q=world&page=2", "world", "2"},
		{"?q=test", "test", "1"},
		{"", "", "1"},
		{"?q=hello&page=1", "hello", "1"}, // Same as first to verify no caching issues
	}

	for i, tc := range testCases {
		t.Run("page_request_"+string(rune('A'+i)), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/search"+tc.query, http.NoBody)
			rec := httptest.NewRecorder()

			srv.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Logf("stderr: %s", stderr.String())
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			body := rec.Body.String()

			// Check that the query value is in the response
			expectedQuerySpan := `<span id="query">` + tc.expectQuery + `</span>`
			if !strings.Contains(body, expectedQuerySpan) {
				t.Errorf("expected query span %q in body, got:\n%s", expectedQuerySpan, body)
			}

			expectedPageSpan := `<span id="page">` + tc.expectPage + `</span>`
			if !strings.Contains(body, expectedPageSpan) {
				t.Errorf("expected page span %q in body, got:\n%s", expectedPageSpan, body)
			}
		})
	}
}

// TestAPIParamsWithAtParamsDirect tests using @params directly in handler (not via @basil/http)
func TestAPIParamsWithAtParamsDirect(t *testing.T) {
	evaluator.ClearModuleCache()

	dir := t.TempDir()

	// Create API handler that uses @params directly
	apiPath := filepath.Join(dir, "direct_api.pars")
	apiCode := `let api = import @std/api

export get = api.public(fn(req) {
    let search = @params.search ?? "none"
    let limit = @params.limit ?? "10"
    {
        search: search,
        limit: limit
    }
})
`
	if err := os.WriteFile(apiPath, []byte(apiCode), 0644); err != nil {
		t.Fatalf("failed to write API script: %v", err)
	}

	cfg := &config.Config{
		BaseDir: dir,
		Server:  config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		Routes:  []config.Route{{Path: "/api/direct", Handler: apiPath, Type: "api"}},
		Logging: config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	testCases := []struct {
		query          string
		expectedSearch string
		expectedLimit  string
	}{
		{"?search=foo&limit=20", "foo", "20"},
		{"?search=bar&limit=5", "bar", "5"},
		{"", "none", "10"},
		{"?search=baz", "baz", "10"},
	}

	for i, tc := range testCases {
		t.Run("request_"+string(rune('A'+i)), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/direct"+tc.query, http.NoBody)
			rec := httptest.NewRecorder()
			srv.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}

			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			if body["search"] != tc.expectedSearch {
				t.Errorf("search: expected %q, got %q", tc.expectedSearch, body["search"])
			}
			if body["limit"] != tc.expectedLimit {
				t.Errorf("limit: expected %q, got %q", tc.expectedLimit, body["limit"])
			}
		})
	}
}
