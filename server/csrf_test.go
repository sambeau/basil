package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateCSRFToken(t *testing.T) {
	token, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken() error = %v", err)
	}

	// Token should be 64 hex characters (32 bytes)
	if len(token) != 64 {
		t.Errorf("GenerateCSRFToken() token length = %d, want 64", len(token))
	}

	// Token should be hex characters only
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("GenerateCSRFToken() contains non-hex character: %c", c)
		}
	}

	// Tokens should be unique
	token2, _ := GenerateCSRFToken()
	if token == token2 {
		t.Error("GenerateCSRFToken() generated duplicate tokens")
	}
}

func TestGetCSRFToken(t *testing.T) {
	tests := []struct {
		name        string
		cookieValue string
		wantNew     bool
	}{
		{
			name:        "no cookie",
			cookieValue: "",
			wantNew:     true,
		},
		{
			name:        "valid cookie",
			cookieValue: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantNew:     false,
		},
		{
			name:        "invalid cookie - too short",
			cookieValue: "tooshort",
			wantNew:     true,
		},
		{
			name:        "invalid cookie - too long",
			cookieValue: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0000",
			wantNew:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookieValue})
			}

			token, isNew := GetCSRFToken(req)

			if isNew != tt.wantNew {
				t.Errorf("GetCSRFToken() isNew = %v, want %v", isNew, tt.wantNew)
			}

			if !isNew && token != tt.cookieValue {
				t.Errorf("GetCSRFToken() token = %v, want %v", token, tt.cookieValue)
			}

			if isNew && len(token) != 64 {
				t.Errorf("GetCSRFToken() new token length = %d, want 64", len(token))
			}
		})
	}
}

func TestSetCSRFCookie(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		devMode    bool
		wantSecure bool
	}{
		{
			name:       "production mode",
			token:      "testtoken123",
			devMode:    false,
			wantSecure: true,
		},
		{
			name:       "dev mode",
			token:      "testtoken123",
			devMode:    true,
			wantSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetCSRFCookie(w, tt.token, tt.devMode)

			resp := w.Result()
			cookies := resp.Cookies()

			if len(cookies) != 1 {
				t.Fatalf("SetCSRFCookie() set %d cookies, want 1", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != CSRFCookieName {
				t.Errorf("SetCSRFCookie() cookie name = %s, want %s", cookie.Name, CSRFCookieName)
			}
			if cookie.Value != tt.token {
				t.Errorf("SetCSRFCookie() cookie value = %s, want %s", cookie.Value, tt.token)
			}
			if cookie.Secure != tt.wantSecure {
				t.Errorf("SetCSRFCookie() cookie.Secure = %v, want %v", cookie.Secure, tt.wantSecure)
			}
			if !cookie.HttpOnly {
				t.Error("SetCSRFCookie() cookie.HttpOnly = false, want true")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("SetCSRFCookie() cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteStrictMode)
			}
			if cookie.Path != "/" {
				t.Errorf("SetCSRFCookie() cookie.Path = %s, want /", cookie.Path)
			}
		})
	}
}

func TestValidateCSRF(t *testing.T) {
	validToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name        string
		cookieToken string
		formToken   string
		headerToken string
		contentType string
		want        bool
	}{
		{
			name:        "valid - form field matches cookie",
			cookieToken: validToken,
			formToken:   validToken,
			want:        true,
		},
		{
			name:        "valid - header matches cookie",
			cookieToken: validToken,
			headerToken: validToken,
			want:        true,
		},
		{
			name:        "invalid - no cookie",
			cookieToken: "",
			formToken:   validToken,
			want:        false,
		},
		{
			name:        "invalid - no form token or header",
			cookieToken: validToken,
			want:        false,
		},
		{
			name:        "invalid - token mismatch",
			cookieToken: validToken,
			formToken:   "differenttoken",
			want:        false,
		},
		{
			name:        "invalid - header mismatch",
			cookieToken: validToken,
			headerToken: "differenttoken",
			want:        false,
		},
		{
			name:        "form takes precedence over header",
			cookieToken: validToken,
			formToken:   validToken,
			headerToken: "wrongtoken",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build form body if form token provided
			var body string
			contentType := "text/plain"
			if tt.formToken != "" {
				body = url.Values{CSRFFormField: {tt.formToken}}.Encode()
				contentType = "application/x-www-form-urlencoded"
			}

			req := httptest.NewRequest("POST", "/", strings.NewReader(body))
			req.Header.Set("Content-Type", contentType)

			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookieToken})
			}
			if tt.headerToken != "" {
				req.Header.Set(CSRFHeaderName, tt.headerToken)
			}

			got := ValidateCSRF(req)
			if got != tt.want {
				t.Errorf("ValidateCSRF() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "equal strings",
			a:    "test",
			b:    "test",
			want: true,
		},
		{
			name: "different strings same length",
			a:    "test",
			b:    "best",
			want: false,
		},
		{
			name: "different lengths",
			a:    "test",
			b:    "testing",
			want: false,
		},
		{
			name: "empty strings",
			a:    "",
			b:    "",
			want: true,
		},
		{
			name: "one empty",
			a:    "test",
			b:    "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secureCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("secureCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMutatingMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"GET", false},
		{"HEAD", false},
		{"OPTIONS", false},
		{"POST", true},
		{"PUT", true},
		{"PATCH", true},
		{"DELETE", true},
		{"post", true},   // case insensitive
		{"delete", true}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isMutatingMethod(tt.method); got != tt.want {
				t.Errorf("isMutatingMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestCSRFMiddleware_Validate(t *testing.T) {
	validToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Handler that always succeeds
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name        string
		method      string
		cookieToken string
		formToken   string
		headerToken string
		devMode     bool
		wantStatus  int
	}{
		{
			name:       "GET request - no validation",
			method:     "GET",
			wantStatus: http.StatusOK,
		},
		{
			name:       "HEAD request - no validation",
			method:     "HEAD",
			wantStatus: http.StatusOK,
		},
		{
			name:       "OPTIONS request - no validation",
			method:     "OPTIONS",
			wantStatus: http.StatusOK,
		},
		{
			name:        "POST with valid token",
			method:      "POST",
			cookieToken: validToken,
			formToken:   validToken,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "PUT with valid token",
			method:      "PUT",
			cookieToken: validToken,
			formToken:   validToken,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "PATCH with valid token",
			method:      "PATCH",
			cookieToken: validToken,
			formToken:   validToken,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "DELETE with valid token via header",
			method:      "DELETE",
			cookieToken: validToken,
			headerToken: validToken, // DELETE typically uses header since no body
			wantStatus:  http.StatusOK,
		},
		{
			name:       "POST without token - fails",
			method:     "POST",
			wantStatus: http.StatusForbidden,
		},
		{
			name:        "POST with mismatched token - fails",
			method:      "POST",
			cookieToken: validToken,
			formToken:   "wrongtoken",
			wantStatus:  http.StatusForbidden,
		},
		{
			name:       "POST dev mode - detailed error",
			method:     "POST",
			devMode:    true,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := NewCSRFMiddleware(tt.devMode)
			handler := mw.Validate(successHandler)

			var body string
			contentType := "text/plain"
			if tt.formToken != "" {
				body = url.Values{CSRFFormField: {tt.formToken}}.Encode()
				contentType = "application/x-www-form-urlencoded"
			}

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(body))
			req.Header.Set("Content-Type", contentType)

			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookieToken})
			}
			if tt.headerToken != "" {
				req.Header.Set(CSRFHeaderName, tt.headerToken)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("CSRFMiddleware.Validate() status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Check that dev mode has detailed error message
			if tt.wantStatus == http.StatusForbidden && tt.devMode {
				body := w.Body.String()
				if !strings.Contains(body, "CSRF token validation failed") {
					t.Error("Dev mode error should contain detailed message")
				}
				if !strings.Contains(body, "basil.csrf.token") {
					t.Error("Dev mode error should contain usage hint")
				}
			}
		})
	}
}

func TestCSRFMiddleware_HeaderToken(t *testing.T) {
	validToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCSRFMiddleware(false)
	handler := mw.Validate(successHandler)

	// Test with X-CSRF-Token header (for AJAX)
	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validToken})
	req.Header.Set(CSRFHeaderName, validToken)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CSRF validation with header token failed, status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_JSONBody(t *testing.T) {
	validToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewCSRFMiddleware(false)
	handler := mw.Validate(successHandler)

	// Test with JSON body - must use header since JSON doesn't have form fields
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validToken})
	req.Header.Set(CSRFHeaderName, validToken)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CSRF validation with JSON body failed, status = %d, want %d", w.Code, http.StatusOK)
	}
}
