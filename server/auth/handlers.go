package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
)

// Handlers provides HTTP handlers for authentication endpoints.
type Handlers struct {
	db           *DB
	webauthn     *WebAuthnManager
	emailService *EmailService
	sessionTTL   time.Duration
	secure       bool // Use secure cookies (HTTPS)
	regOpen      bool // Registration open to public
	requireVerif bool // Require email verification for protected routes
}

// NewHandlers creates a new auth handlers instance.
func NewHandlers(db *DB, webauthn *WebAuthnManager, emailService *EmailService, sessionTTL time.Duration, secure, regOpen, requireVerif bool) *Handlers {
	return &Handlers{
		db:           db,
		webauthn:     webauthn,
		emailService: emailService,
		sessionTTL:   sessionTTL,
		secure:       secure,
		regOpen:      regOpen,
		requireVerif: requireVerif,
	}
}

// --- Registration endpoints ---

// BeginRegisterHandler handles POST /__auth/register/begin
func (h *Handlers) BeginRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if registration is open
	if !h.regOpen {
		// Check if this is the first user (always allow first user)
		count, err := h.db.UserCount()
		if err != nil {
			jsonError(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			jsonError(w, "Registration is closed", http.StatusForbidden)
			return
		}
	}

	// Parse request body
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		jsonError(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check if email already exists
	if req.Email != "" {
		existing, _ := h.db.GetUserByEmail(req.Email)
		if existing != nil {
			// Check if they have any passkeys - CLI-created users may not
			hasCredentials, _ := h.db.HasCredentials(existing.ID)
			if hasCredentials {
				jsonError(w, "Email already registered", http.StatusConflict)
				return
			}
			// User exists but has no passkey - allow registration for existing user
			options, challengeID, err := h.webauthn.BeginRegistrationForExisting(existing)
			if err != nil {
				jsonError(w, "Failed to start registration", http.StatusInternalServerError)
				return
			}
			jsonResponse(w, map[string]any{
				"options":      options,
				"challenge_id": challengeID,
			})
			return
		}
	}

	// Begin WebAuthn registration for new user
	options, challengeID, err := h.webauthn.BeginRegistration(req.Name, req.Email)
	if err != nil {
		jsonError(w, "Failed to start registration", http.StatusInternalServerError)
		return
	}

	// Return options to browser
	jsonResponse(w, map[string]any{
		"options":      options,
		"challenge_id": challengeID,
	})
}

// FinishRegisterHandler handles POST /__auth/register/finish
func (h *Handlers) FinishRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	var req struct {
		ChallengeID string          `json:"challenge_id"`
		Response    json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse the WebAuthn credential response
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(
		io.NopCloser(newBytesReader(req.Response)),
	)
	if err != nil {
		jsonError(w, "Invalid credential response", http.StatusBadRequest)
		return
	}

	// Complete registration
	user, codes, err := h.webauthn.FinishRegistration(req.ChallengeID, parsedResponse)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send verification email if enabled and user has email
	var emailSent bool
	if h.emailService != nil && user.Email != "" {
		if err := h.emailService.SendVerificationEmail(r.Context(), user); err != nil {
			// Log error but don't fail registration
			fmt.Printf("Failed to send verification email: %v\n", err)
		} else {
			emailSent = true
		}
	}

	// Create session
	session, err := h.db.CreateSession(user.ID, h.sessionTTL)
	if err != nil {
		jsonError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	SetSessionCookie(w, session, h.secure)

	// Return user and recovery codes
	response := map[string]any{
		"user":           user,
		"recovery_codes": codes,
	}
	if emailSent {
		response["email_verification_sent"] = true
	}

	jsonResponse(w, response)
}

// --- Login endpoints ---

// BeginLoginHandler handles POST /__auth/login/begin
func (h *Handlers) BeginLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	options, challengeID, err := h.webauthn.BeginLogin()
	if err != nil {
		jsonError(w, "Failed to start login", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]any{
		"options":      options,
		"challenge_id": challengeID,
	})
}

// FinishLoginHandler handles POST /__auth/login/finish
func (h *Handlers) FinishLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	var req struct {
		ChallengeID string          `json:"challenge_id"`
		Response    json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse the WebAuthn assertion response
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(
		io.NopCloser(newBytesReader(req.Response)),
	)
	if err != nil {
		jsonError(w, fmt.Sprintf("Invalid credential response: %v", err), http.StatusBadRequest)
		return
	}

	// Complete login
	user, err := h.webauthn.FinishLogin(req.ChallengeID, parsedResponse)
	if err != nil {
		jsonError(w, fmt.Sprintf("Login failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Create session
	session, err := h.db.CreateSession(user.ID, h.sessionTTL)
	if err != nil {
		jsonError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	SetSessionCookie(w, session, h.secure)

	jsonResponse(w, map[string]any{
		"user": user,
	})
}

// --- Logout endpoint ---

// LogoutHandler handles POST /__auth/logout
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get and delete session
	token := GetSessionToken(r)
	if token != "" {
		h.db.DeleteSession(token)
	}

	ClearSessionCookie(w, h.secure)

	jsonResponse(w, map[string]any{
		"success": true,
	})
}

// --- Recovery endpoint ---

// RecoverHandler handles POST /__auth/recover
func (h *Handlers) RecoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find user by email
	user, err := h.db.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		// Don't reveal if user exists
		jsonError(w, "Invalid recovery code", http.StatusUnauthorized)
		return
	}

	// Validate recovery code
	valid, err := h.db.ValidateRecoveryCode(user.ID, req.Code)
	if err != nil || !valid {
		jsonError(w, "Invalid recovery code", http.StatusUnauthorized)
		return
	}

	// Create session - user can now add a new passkey
	session, err := h.db.CreateSession(user.ID, h.sessionTTL)
	if err != nil {
		jsonError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	SetSessionCookie(w, session, h.secure)

	// Get remaining code count
	remaining, _ := h.db.GetRecoveryCodeCount(user.ID)

	jsonResponse(w, map[string]any{
		"user":                   user,
		"remaining_codes":        remaining,
		"should_add_new_passkey": true,
	})
}

// --- User info endpoint ---

// MeHandler handles GET /__auth/me
func (h *Handlers) MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := GetSessionToken(r)
	if token == "" {
		jsonResponse(w, map[string]any{"user": nil})
		return
	}

	user, err := h.db.ValidateSession(token)
	if err != nil || user == nil {
		jsonResponse(w, map[string]any{"user": nil})
		return
	}

	jsonResponse(w, map[string]any{"user": user})
}

// --- Email Verification endpoints (FEAT-084) ---

// VerifyEmailHandler handles GET /__auth/verify-email?token=xxx
func (h *Handlers) VerifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	// Look up token
	verification, err := h.db.LookupVerificationToken(r.Context(), token)
	if err != nil {
		// Token not found or expired - show error page with resend option
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Verification Failed</title></head>
<body>
<h1>Email Verification Failed</h1>
<p>This verification link is invalid or has expired.</p>
<p><a href="/__auth/resend-verification">Request a new verification email</a></p>
</body>
</html>`)
		return
	}

	// Mark email as verified
	if err := h.db.MarkEmailVerified(r.Context(), verification.UserID); err != nil {
		http.Error(w, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	// Consume token
	if err := h.db.ConsumeVerificationToken(r.Context(), verification.ID); err != nil {
		// Log error but continue - email is verified
		fmt.Printf("Failed to consume token: %v\n", err)
	}

	// Show success page with redirect
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<title>Email Verified</title>
<meta http-equiv="refresh" content="3;url=/">
</head>
<body>
<h1>Email Verified!</h1>
<p>Your email address has been successfully verified.</p>
<p>Redirecting to dashboard...</p>
</body>
</html>`)
}

// ResendVerificationHandler handles POST /__auth/resend-verification
func (h *Handlers) ResendVerificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current session
	sessionToken := GetSessionToken(r)
	if sessionToken == "" {
		jsonError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	user, err := h.db.ValidateSession(sessionToken)
	if err != nil || user == nil {
		jsonError(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	// Check if already verified
	if user.EmailVerifiedAt != nil {
		jsonError(w, "Email already verified", http.StatusBadRequest)
		return
	}

	// Check if user has email
	if user.Email == "" {
		jsonError(w, "No email address on file", http.StatusBadRequest)
		return
	}

	// Send verification email (includes rate limiting)
	if err := h.emailService.SendVerificationEmail(r.Context(), user); err != nil {
		jsonError(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	jsonResponse(w, map[string]any{
		"success": true,
		"message": "Verification email sent",
	})
}

// VerificationRequiredHandler handles GET /__auth/verify-email-required
func (h *Handlers) VerificationRequiredHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Email Verification Required</title></head>
<body>
<h1>Email Verification Required</h1>
<p>Please verify your email address to access this page.</p>
<form method="POST" action="/__auth/resend-verification">
<button type="submit">Resend Verification Email</button>
</form>
<p><a href="/__auth/logout">Logout</a></p>
</body>
</html>`)
}

// RecoverEmailHandler handles POST /__auth/recover/email
func (h *Handlers) RecoverEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Always return success to prevent email enumeration
	// Perform checks and send email only if valid

	if req.Email != "" {
		user, _ := h.db.GetUserByEmail(req.Email)
		// Only send if user exists and email is verified
		if user != nil && user.EmailVerifiedAt != nil {
			// Send recovery email (errors are logged but not exposed)
			_ = h.emailService.SendRecoveryEmail(r.Context(), user)
		}
	}

	// Constant-time response
	time.Sleep(100 * time.Millisecond)

	jsonResponse(w, map[string]any{
		"success": true,
		"message": "If this email is registered and verified, a recovery link has been sent.",
	})
}

// RecoverVerifyHandler handles GET /__auth/recover/verify?token=xxx
func (h *Handlers) RecoverVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	// Look up token
	verification, err := h.db.LookupVerificationToken(r.Context(), token)
	if err != nil {
		// Token not found or expired
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Recovery Failed</title></head>
<body>
<h1>Account Recovery Failed</h1>
<p>This recovery link is invalid or has expired.</p>
<p><a href="/__auth/recover">Try recovery codes instead</a></p>
</body>
</html>`)
		return
	}

	// Get user
	user, err := h.db.GetUser(verification.UserID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Consume token
	if err := h.db.ConsumeVerificationToken(r.Context(), verification.ID); err != nil {
		http.Error(w, "Failed to consume token", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := h.db.CreateSession(user.ID, h.sessionTTL)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	SetSessionCookie(w, session, h.secure)

	// Redirect to register new passkey
	http.Redirect(w, r, "/?recovery=true", http.StatusSeeOther)
}

// --- Helpers ---

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// bytesReader wraps []byte for io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
