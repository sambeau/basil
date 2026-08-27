package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/config"
)

// benchSiteFixture builds a page importing componentCount components, each
// importing a shared helper - a component-heavy page, which is where the cost
// of re-importing on every request would be felt.
func benchSiteFixture(b *testing.B, dev, devCache bool) *Server {
	b.Helper()
	evaluator.ClearModuleCache()

	dir := b.TempDir()
	componentDir := filepath.Join(dir, "components")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		b.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(componentDir, "helper.pars"),
		[]byte("export label = \"shared\"\n"), 0o644); err != nil {
		b.Fatalf("write helper: %v", err)
	}

	const componentCount = 20
	handler := ""
	body := ""
	for i := range componentCount {
		src := fmt.Sprintf(`let helper = import @~/components/helper.pars
export name = helper.label + "-%d"
`, i)
		name := fmt.Sprintf("c%d.pars", i)
		if err := os.WriteFile(filepath.Join(componentDir, name), []byte(src), 0o644); err != nil {
			b.Fatalf("write component: %v", err)
		}
		handler += fmt.Sprintf("let c%d = import @~/components/c%d.pars\n", i, i)
		if i > 0 {
			body += " + "
		}
		body += fmt.Sprintf("c%d.name", i)
	}
	handler += fmt.Sprintf("\n<html><body>%s</body></html>\n", body)

	handlerPath := filepath.Join(dir, "api.pars")
	if err := os.WriteFile(handlerPath, []byte(handler), 0o644); err != nil {
		b.Fatalf("write handler: %v", err)
	}

	cfg := &config.Config{
		ReleaseDir: dir,
		DataDir:    dir,
		Server:     config.ServerConfig{Host: "localhost", Port: 8080, Dev: dev},
		Dev:        config.DevConfig{Cache: devCache},
		Routes:     []config.Route{{Path: "/names", Handler: handlerPath}},
		Logging:    config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		b.Fatalf("New(): %v", err)
	}
	return srv
}

func benchRequests(b *testing.B, srv *Server) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		req := httptest.NewRequest(http.MethodGet, "/names", http.NoBody)
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	}
}

// A page route rather than an API one: API handlers are rate limited, and the
// question here is the cost of importing, not of the limiter.

// BenchmarkModuleCacheDev is the cost of a dev-mode request against a
// component-heavy page: 20 components sharing a helper, so 21 files whose
// stamps must agree before the cached modules are served.
func BenchmarkModuleCacheDev(b *testing.B) {
	benchRequests(b, benchSiteFixture(b, true, false))
}

// BenchmarkModuleCacheProduction is the same page with entries trusted as they
// stand. The gap between the two is what revalidation costs.
func BenchmarkModuleCacheProduction(b *testing.B) {
	benchRequests(b, benchSiteFixture(b, false, false))
}
