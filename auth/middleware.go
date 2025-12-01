package auth

import (
	"context"
	"net/http"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey contextKey = "auth_user"
)

// Middleware provides authentication middleware for HTTP handlers.
type Middleware struct {
	db *DB
}

// NewMiddleware creates a new auth middleware instance.
func NewMiddleware(db *DB) *Middleware {
	return &Middleware{db: db}
}

// RequireAuth returns middleware that requires authentication.
// If not authenticated, returns 401 Unauthorized.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.authenticate(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to context and continue
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth returns middleware that checks authentication but doesn't require it.
// Sets user in context if authenticated, nil otherwise.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.authenticate(r)
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticate checks the session cookie and returns the user if valid.
func (m *Middleware) authenticate(r *http.Request) *User {
	token := GetSessionToken(r)
	if token == "" {
		return nil
	}

	user, err := m.db.ValidateSession(token)
	if err != nil || user == nil {
		return nil
	}

	return user
}

// GetUser retrieves the authenticated user from the request context.
// Returns nil if not authenticated.
func GetUser(r *http.Request) *User {
	user, ok := r.Context().Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// GetUserFromContext retrieves the authenticated user from a context.
// Returns nil if not authenticated.
func GetUserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}
