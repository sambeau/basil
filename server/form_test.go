package server

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambeau/basil/config"
)

func TestParseURLEncodedForm(t *testing.T) {
	body := "name=John&email=john@example.com&tags=go&tags=web"
	req := httptest.NewRequest("POST", "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	route := config.Route{Path: "/submit", Handler: "test.pars"}
	ctx := buildRequestContext(req, route)

	form, ok := ctx["form"].(map[string]interface{})
	if !ok {
		t.Fatal("form should be a map")
	}

	if form["name"] != "John" {
		t.Errorf("expected name=John, got %v", form["name"])
	}
	if form["email"] != "john@example.com" {
		t.Errorf("expected email=john@example.com, got %v", form["email"])
	}
	// Multiple values should be a slice
	tags, ok := form["tags"].([]string)
	if !ok {
		t.Errorf("tags should be a slice, got %T", form["tags"])
	} else if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestParseJSONBody(t *testing.T) {
	body := `{"name": "John", "age": 30, "active": true}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	route := config.Route{Path: "/api/users", Handler: "test.pars"}
	ctx := buildRequestContext(req, route)

	// Raw body should be available
	rawBody, ok := ctx["body"].(string)
	if !ok || rawBody != body {
		t.Errorf("expected raw body, got %v", ctx["body"])
	}

	// Parsed form should have JSON data
	form, ok := ctx["form"].(map[string]interface{})
	if !ok {
		t.Fatal("form should be a map for JSON")
	}

	if form["name"] != "John" {
		t.Errorf("expected name=John, got %v", form["name"])
	}
	if form["age"] != float64(30) { // JSON numbers are float64
		t.Errorf("expected age=30, got %v", form["age"])
	}
	if form["active"] != true {
		t.Errorf("expected active=true, got %v", form["active"])
	}
}

func TestParseMultipartForm(t *testing.T) {
	// Create a multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add a form field
	writer.WriteField("username", "johndoe")
	writer.WriteField("bio", "Hello world")

	// Add a file
	part, err := writer.CreateFormFile("avatar", "profile.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("fake image data"))

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	route := config.Route{Path: "/upload", Handler: "test.pars"}
	ctx := buildRequestContext(req, route)

	// Check form fields
	form, ok := ctx["form"].(map[string]interface{})
	if !ok {
		t.Fatal("form should be a map")
	}
	if form["username"] != "johndoe" {
		t.Errorf("expected username=johndoe, got %v", form["username"])
	}

	// Check file metadata
	files, ok := ctx["files"].(map[string]interface{})
	if !ok {
		t.Fatal("files should be a map")
	}

	avatar, ok := files["avatar"].(map[string]interface{})
	if !ok {
		t.Fatal("avatar should be a file map")
	}
	if avatar["filename"] != "profile.png" {
		t.Errorf("expected filename=profile.png, got %v", avatar["filename"])
	}
	if avatar["size"].(int64) != 15 { // "fake image data" = 15 bytes
		t.Errorf("expected size=15, got %v", avatar["size"])
	}
}

func TestParseRawBody(t *testing.T) {
	body := "plain text content"
	req := httptest.NewRequest("POST", "/raw", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	route := config.Route{Path: "/raw", Handler: "test.pars"}
	ctx := buildRequestContext(req, route)

	rawBody, ok := ctx["body"].(string)
	if !ok || rawBody != body {
		t.Errorf("expected raw body '%s', got %v", body, ctx["body"])
	}

	// form and files should be nil for plain text
	if form := ctx["form"]; form != nil {
		// Check if it's an empty map (which is fine) vs a populated map
		if formMap, ok := form.(map[string]interface{}); ok && len(formMap) > 0 {
			t.Error("form should be nil or empty for plain text")
		}
	}
}

func TestGETRequestHasNoBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/page", nil)

	route := config.Route{Path: "/page", Handler: "test.pars"}
	ctx := buildRequestContext(req, route)

	// GET requests should not have body/form/files
	if _, ok := ctx["body"]; ok {
		t.Error("GET request should not have body")
	}
	if _, ok := ctx["form"]; ok {
		t.Error("GET request should not have form")
	}
	if _, ok := ctx["files"]; ok {
		t.Error("GET request should not have files")
	}
}
