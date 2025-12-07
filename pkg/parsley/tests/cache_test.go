package tests

import (
	"testing"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// mockFragmentCache implements evaluator.FragmentCacher for testing
type mockFragmentCache struct {
	entries map[string]string
	enabled bool
}

func newMockFragmentCache() *mockFragmentCache {
	return &mockFragmentCache{
		entries: make(map[string]string),
		enabled: true,
	}
}

func (m *mockFragmentCache) Get(key string) (string, bool) {
	if !m.enabled {
		return "", false
	}
	html, ok := m.entries[key]
	return html, ok
}

func (m *mockFragmentCache) Set(key string, html string, maxAge time.Duration) {
	if m.enabled && maxAge > 0 {
		m.entries[key] = html
	}
}

func (m *mockFragmentCache) Invalidate(key string) {
	delete(m.entries, key)
}

// testCacheEval evaluates Parsley code with a fragment cache available
func testCacheEval(input string, cache *mockFragmentCache, handlerPath string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	env.FragmentCache = cache
	env.HandlerPath = handlerPath
	env.DevMode = false
	return evaluator.Eval(program, env)
}

func TestCacheTag_BasicUsage(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache key="sidebar" maxAge={@5m}><div>Cached content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/dashboard")

	if result == nil {
		t.Fatal("Eval returned nil")
	}

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T: %v", result, result.Inspect())
	}

	expected := "<div>Cached content</div>"
	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}

	// Verify it was cached with the correct key
	if cached, ok := cache.entries["/dashboard:sidebar"]; !ok {
		t.Error("Expected entry to be cached")
	} else if cached != expected {
		t.Errorf("Cached value mismatch: expected '%s', got '%s'", expected, cached)
	}
}

func TestCacheTag_CacheHit(t *testing.T) {
	cache := newMockFragmentCache()

	// Pre-populate cache
	cache.entries["/dashboard:sidebar"] = "<div>Pre-cached content</div>"

	// The input has different content, but we should get the cached version
	input := `<basil.cache.Cache key="sidebar" maxAge={@5m}><div>New content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/dashboard")

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	// Should return cached content, not new content
	expected := "<div>Pre-cached content</div>"
	if strResult.Value != expected {
		t.Errorf("Expected cached '%s', got '%s'", expected, strResult.Value)
	}
}

func TestCacheTag_DynamicKey(t *testing.T) {
	cache := newMockFragmentCache()

	input := `let userId = 123
<basil.cache.Cache key={"user-" + userId} maxAge={@1h}><div>User content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/profile")

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T: %v", result, result.Inspect())
	}

	expected := "<div>User content</div>"
	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}

	// Verify dynamic key was used
	if _, ok := cache.entries["/profile:user-123"]; !ok {
		t.Error("Expected entry with dynamic key 'user-123' to be cached")
	}
}

func TestCacheTag_MissingKeyAttribute(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache maxAge={@5m}><div>Content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/test")

	errResult, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("Expected Error for missing key, got %T: %v", result, result.Inspect())
	}

	if errResult.Code != "CACHE-0001" {
		t.Errorf("Expected error code CACHE-0001, got %s", errResult.Code)
	}
}

func TestCacheTag_MissingMaxAgeAttribute(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache key="sidebar"><div>Content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/test")

	errResult, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("Expected Error for missing maxAge, got %T: %v", result, result.Inspect())
	}

	if errResult.Code != "CACHE-0003" {
		t.Errorf("Expected error code CACHE-0003, got %s", errResult.Code)
	}
}

func TestCacheTag_InvalidKeyType(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache key={123} maxAge={@5m}><div>Content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/test")

	errResult, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("Expected Error for invalid key type, got %T: %v", result, result.Inspect())
	}

	if errResult.Code != "CACHE-0002" {
		t.Errorf("Expected error code CACHE-0002, got %s", errResult.Code)
	}
}

func TestCacheTag_InvalidMaxAgeType(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache key="sidebar" maxAge={300}><div>Content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/test")

	errResult, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("Expected Error for invalid maxAge type, got %T: %v", result, result.Inspect())
	}

	if errResult.Code != "CACHE-0004" {
		t.Errorf("Expected error code CACHE-0004, got %s", errResult.Code)
	}
}

func TestCacheTag_EnabledFalse(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache key="sidebar" maxAge={@5m} enabled={false}><div>Content</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/test")

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T: %v", result, result.Inspect())
	}

	expected := "<div>Content</div>"
	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}

	// Should NOT be cached when enabled=false
	if _, ok := cache.entries["/test:sidebar"]; ok {
		t.Error("Entry should not be cached when enabled=false")
	}
}

func TestCacheTag_DevMode(t *testing.T) {
	cache := newMockFragmentCache()

	l := lexer.New(`<basil.cache.Cache key="sidebar" maxAge={@5m}><div>Content</div></basil.cache.Cache>`)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	env.FragmentCache = cache
	env.HandlerPath = "/test"
	env.DevMode = true // Enable dev mode

	result := evaluator.Eval(program, env)

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T: %v", result, result.Inspect())
	}

	expected := "<div>Content</div>"
	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}

	// Should NOT be cached in dev mode
	if _, ok := cache.entries["/test:sidebar"]; ok {
		t.Error("Entry should not be cached in dev mode")
	}
}

func TestCacheTag_NoCacheAvailable(t *testing.T) {
	// No cache set on environment
	l := lexer.New(`<basil.cache.Cache key="sidebar" maxAge={@5m}><div>Content</div></basil.cache.Cache>`)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	// FragmentCache is nil

	result := evaluator.Eval(program, env)

	// Should still work, just not cache
	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String when no cache available, got %T: %v", result, result.Inspect())
	}

	expected := "<div>Content</div>"
	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}
}

func TestCacheTag_NestedContent(t *testing.T) {
	cache := newMockFragmentCache()

	input := `<basil.cache.Cache key="complex" maxAge={@10m}><div class="wrapper"><header><h1>Title</h1></header><main>Main content</main><footer>Footer</footer></div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/page")

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T: %v", result, result.Inspect())
	}

	// Verify HTML structure is preserved
	if len(strResult.Value) == 0 {
		t.Error("Expected non-empty result")
	}

	// Verify it contains expected elements
	if !contains(strResult.Value, "<header>") || !contains(strResult.Value, "</footer>") {
		t.Errorf("Expected nested HTML structure, got '%s'", strResult.Value)
	}
}

func TestCacheTag_WithInterpolation(t *testing.T) {
	cache := newMockFragmentCache()

	input := `let name = "Alice"
<basil.cache.Cache key="greeting" maxAge={@5m}><div>Hello, {name}!</div></basil.cache.Cache>`
	result := testCacheEval(input, cache, "/greet")

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T: %v", result, result.Inspect())
	}

	expected := "<div>Hello, Alice!</div>"
	if strResult.Value != expected {
		t.Errorf("Expected '%s', got '%s'", expected, strResult.Value)
	}
}

func TestCacheTag_HandlerNamespacing(t *testing.T) {
	cache := newMockFragmentCache()

	// Same key, different handlers
	input := `<basil.cache.Cache key="sidebar" maxAge={@5m}><div>Dashboard sidebar</div></basil.cache.Cache>`
	testCacheEval(input, cache, "/dashboard")

	input2 := `<basil.cache.Cache key="sidebar" maxAge={@5m}><div>Profile sidebar</div></basil.cache.Cache>`
	testCacheEval(input2, cache, "/profile")

	// Both should be cached separately
	if cached, ok := cache.entries["/dashboard:sidebar"]; !ok || cached != "<div>Dashboard sidebar</div>" {
		t.Error("Dashboard sidebar should be cached separately")
	}
	if cached, ok := cache.entries["/profile:sidebar"]; !ok || cached != "<div>Profile sidebar</div>" {
		t.Error("Profile sidebar should be cached separately")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
