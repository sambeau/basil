package auth

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnManager handles passkey registration and authentication.
type WebAuthnManager struct {
	webauthn *webauthn.WebAuthn
	db       *DB

	// Challenge storage (in-memory, short-lived)
	mu         sync.RWMutex
	challenges map[string]*challengeData
}

// challengeData stores pending challenge information.
type challengeData struct {
	sessionData    *webauthn.SessionData
	user           *webAuthnUser // For registration
	existingUserID string        // Non-empty if registering passkey for existing user
	expiresAt      time.Time
}

// webAuthnUser implements webauthn.User interface.
type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webAuthnUser) WebAuthnName() string                       { return u.name }
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// NewWebAuthnManager creates a new WebAuthn manager.
func NewWebAuthnManager(db *DB, rpID, rpOrigin, rpName string) (*WebAuthnManager, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
		// Use resident keys (discoverable credentials) for passwordless login
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationPreferred,
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("initializing webauthn: %w", err)
	}

	return &WebAuthnManager{
		webauthn:   w,
		db:         db,
		challenges: make(map[string]*challengeData),
	}, nil
}

// --- Registration ---

// BeginRegistration starts the passkey registration process.
// Returns the WebAuthn options to send to the browser.
func (m *WebAuthnManager) BeginRegistration(name, email string) (*protocol.CredentialCreation, string, error) {
	// Create a temporary user for registration
	userID := []byte(generateID("usr"))
	user := &webAuthnUser{
		id:          userID,
		name:        email, // Use email as username if provided, otherwise name
		displayName: name,
		credentials: nil,
	}
	if email == "" {
		user.name = name
	}

	// Generate registration options
	options, sessionData, err := m.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, "", fmt.Errorf("beginning registration: %w", err)
	}

	// Store challenge with temporary user data
	challengeID := generateID("chal")
	m.mu.Lock()
	m.challenges[challengeID] = &challengeData{
		sessionData: sessionData,
		user:        user,
		expiresAt:   time.Now().Add(5 * time.Minute),
	}
	m.mu.Unlock()

	return options, challengeID, nil
}

// BeginRegistrationForExisting starts passkey registration for an existing user
// who doesn't have any credentials (e.g., created via CLI).
func (m *WebAuthnManager) BeginRegistrationForExisting(existingUser *User) (*protocol.CredentialCreation, string, error) {
	// Use the existing user's ID
	user := &webAuthnUser{
		id:          []byte(existingUser.ID),
		name:        existingUser.Email,
		displayName: existingUser.Name,
		credentials: nil,
	}
	if existingUser.Email == "" {
		user.name = existingUser.Name
	}

	// Generate registration options
	options, sessionData, err := m.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, "", fmt.Errorf("beginning registration: %w", err)
	}

	// Store challenge with existing user ID marker
	challengeID := generateID("chal")
	m.mu.Lock()
	m.challenges[challengeID] = &challengeData{
		sessionData:    sessionData,
		user:           user,
		existingUserID: existingUser.ID, // Mark as existing user
		expiresAt:      time.Now().Add(5 * time.Minute),
	}
	m.mu.Unlock()

	return options, challengeID, nil
}

// FinishRegistration completes registration and creates the user.
// Returns the created user and recovery codes.
func (m *WebAuthnManager) FinishRegistration(challengeID string, response *protocol.ParsedCredentialCreationData) (*User, []string, error) {
	// Get and remove challenge
	m.mu.Lock()
	challenge, ok := m.challenges[challengeID]
	if ok {
		delete(m.challenges, challengeID)
	}
	m.mu.Unlock()

	if !ok {
		return nil, nil, fmt.Errorf("challenge not found or expired")
	}

	if time.Now().After(challenge.expiresAt) {
		return nil, nil, fmt.Errorf("challenge expired")
	}

	// Verify the credential
	credential, err := m.webauthn.CreateCredential(challenge.user, *challenge.sessionData, response)
	if err != nil {
		return nil, nil, fmt.Errorf("verifying credential: %w", err)
	}

	var user *User

	// Check if this is registration for an existing user (created via CLI)
	if challenge.existingUserID != "" {
		// Fetch the existing user
		user, err = m.db.GetUser(challenge.existingUserID)
		if err != nil {
			return nil, nil, fmt.Errorf("fetching existing user: %w", err)
		}
	} else {
		// Create new user in database
		// Extract name from the user object we stored
		name := challenge.user.displayName
		email := ""
		if challenge.user.name != name {
			email = challenge.user.name
		}

		user, err = m.db.CreateUser(name, email)
		if err != nil {
			return nil, nil, fmt.Errorf("creating user: %w", err)
		}
	}

	// Save credential
	cred := &Credential{
		ID:              credential.ID,
		UserID:          user.ID,
		PublicKey:       credential.PublicKey,
		SignCount:       credential.Authenticator.SignCount,
		Transports:      transportStrings(credential.Transport),
		AttestationType: string(credential.AttestationType),
		CreatedAt:       time.Now().UTC(),
	}

	if err := m.db.SaveCredential(cred); err != nil {
		// Rollback user creation only if we created a new user
		if challenge.existingUserID == "" {
			m.db.DeleteUser(user.ID)
		}
		return nil, nil, fmt.Errorf("saving credential: %w", err)
	}

	// Generate recovery codes
	codes, err := m.db.GenerateRecoveryCodes(user.ID, DefaultRecoveryCodeCount)
	if err != nil {
		// Log but don't fail - user can regenerate later
		codes = nil
	}

	return user, codes, nil
}

// --- Login ---

// BeginLogin starts the passkey login process.
// Returns the WebAuthn options to send to the browser.
func (m *WebAuthnManager) BeginLogin() (*protocol.CredentialAssertion, string, error) {
	// For discoverable credentials, we don't specify allowed credentials
	// The browser will show all available passkeys for this origin
	options, sessionData, err := m.webauthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", fmt.Errorf("beginning login: %w", err)
	}

	// Store challenge
	challengeID := generateID("chal")
	m.mu.Lock()
	m.challenges[challengeID] = &challengeData{
		sessionData: sessionData,
		expiresAt:   time.Now().Add(5 * time.Minute),
	}
	m.mu.Unlock()

	return options, challengeID, nil
}

// FinishLogin completes login and returns the authenticated user.
func (m *WebAuthnManager) FinishLogin(challengeID string, response *protocol.ParsedCredentialAssertionData) (*User, error) {
	// Get and remove challenge
	m.mu.Lock()
	challenge, ok := m.challenges[challengeID]
	if ok {
		delete(m.challenges, challengeID)
	}
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("challenge not found or expired")
	}

	if time.Now().After(challenge.expiresAt) {
		return nil, fmt.Errorf("challenge expired")
	}

	// For discoverable login, we need to look up the user from the credential
	credential, err := m.webauthn.ValidateDiscoverableLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			return m.findUserByCredential(rawID, userHandle)
		},
		*challenge.sessionData,
		response,
	)
	if err != nil {
		return nil, fmt.Errorf("validating login: %w", err)
	}

	// Update sign count for replay protection
	if err := m.db.UpdateCredentialSignCount(credential.ID, credential.Authenticator.SignCount); err != nil {
		// Log but don't fail login
	}

	// Get the user from the credential
	cred, err := m.db.GetCredential(credential.ID)
	if err != nil || cred == nil {
		return nil, fmt.Errorf("credential not found")
	}

	return m.db.GetUser(cred.UserID)
}

// findUserByCredential looks up a user from their credential.
// This is called by webauthn during discoverable login.
func (m *WebAuthnManager) findUserByCredential(credentialID, userHandle []byte) (webauthn.User, error) {
	// Try to find by credential ID
	cred, err := m.db.GetCredential(credentialID)
	if err != nil {
		return nil, err
	}
	if cred == nil {
		return nil, fmt.Errorf("credential not found")
	}

	// Load the user
	user, err := m.db.GetUser(cred.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Load all credentials for this user
	creds, err := m.db.GetCredentialsByUser(user.ID)
	if err != nil {
		return nil, err
	}

	// Convert to webauthn credentials
	var waCredentials []webauthn.Credential
	for _, c := range creds {
		waCredentials = append(waCredentials, webauthn.Credential{
			ID:              c.ID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: webauthn.Authenticator{
				SignCount: c.SignCount,
			},
		})
	}

	return &webAuthnUser{
		id:          []byte(user.ID),
		name:        user.Email,
		displayName: user.Name,
		credentials: waCredentials,
	}, nil
}

// CleanExpiredChallenges removes expired challenges from memory.
func (m *WebAuthnManager) CleanExpiredChallenges() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := 0
	for id, c := range m.challenges {
		if now.After(c.expiresAt) {
			delete(m.challenges, id)
			count++
		}
	}
	return count
}

// --- Helpers ---

// transportStrings converts protocol.AuthenticatorTransport to strings.
func transportStrings(transports []protocol.AuthenticatorTransport) []string {
	var result []string
	for _, t := range transports {
		result = append(result, string(t))
	}
	return result
}
