package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	middleware := NewMiddleware(db)

	// Create a test handler that checks for user
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			t.Error("expected user in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.RequireAuth(handler)

	// Test without auth - should get 401
	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	// Test with valid session
	user, _ := db.CreateUser("Alice", "")
	session, _ := db.CreateSession(user.ID, 0)

	req = httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: session.ID,
	})
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOptionalAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	middleware := NewMiddleware(db)

	var capturedUser *User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = GetUser(r)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.OptionalAuth(handler)

	// Test without auth - should succeed but no user
	req := httptest.NewRequest("GET", "/optional", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedUser != nil {
		t.Error("expected nil user without auth")
	}

	// Test with valid session
	user, _ := db.CreateUser("Alice", "")
	session, _ := db.CreateSession(user.ID, 0)

	req = httptest.NewRequest("GET", "/optional", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: session.ID,
	})
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedUser == nil {
		t.Error("expected user with valid session")
	}
	if capturedUser.ID != user.ID {
		t.Errorf("wrong user: got %s, want %s", capturedUser.ID, user.ID)
	}
}

func TestGetUser_NoContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	user := GetUser(req)
	if user != nil {
		t.Error("expected nil user without context")
	}
}
