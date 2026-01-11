package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/sambeau/basil/server/auth/email"
	"github.com/sambeau/basil/server/config"
)

// EmailService handles email sending and verification
type EmailService struct {
	provider email.Provider
	db       *DB
	config   *config.EmailVerificationConfig
	baseURL  string
}

// NewEmailService creates a new email service
func NewEmailService(cfg *config.EmailVerificationConfig, db *DB, baseURL string) (*EmailService, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var provider email.Provider
	var err error

	switch cfg.Provider {
	case "mailgun":
		provider, err = email.NewMailgunProvider(
			cfg.Mailgun.APIKey,
			cfg.Mailgun.Domain,
			cfg.Mailgun.From,
			cfg.Mailgun.Region,
		)
	case "resend":
		provider, err = email.NewResendProvider(
			cfg.Resend.APIKey,
			cfg.Resend.From,
		)
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("initializing email provider: %w", err)
	}

	return &EmailService{
		provider: provider,
		db:       db,
		config:   cfg,
		baseURL:  baseURL,
	}, nil
}

// SendVerificationEmail sends an email verification email to a user
func (s *EmailService) SendVerificationEmail(ctx context.Context, user *User) error {
	if s == nil || s.provider == nil {
		return nil // Email verification not enabled
	}

	// Check rate limits
	cooldown := s.config.ResendCooldown
	if cooldown == 0 {
		cooldown = 5 * time.Minute
	}
	dailyLimit := s.config.MaxSendsPerDay
	if dailyLimit == 0 {
		dailyLimit = 10
	}

	rateLimit, err := s.db.CheckVerificationRateLimit(ctx, user.ID, user.Email, cooldown, dailyLimit)
	if err != nil {
		return fmt.Errorf("checking rate limit: %w", err)
	}

	if !rateLimit.Allowed {
		return fmt.Errorf("rate limit exceeded: %s", rateLimit.Reason)
	}

	// Generate token
	token, err := GenerateVerificationToken()
	if err != nil {
		return fmt.Errorf("generating token: %w", err)
	}

	// Hash token
	tokenHash, err := HashToken(token)
	if err != nil {
		return fmt.Errorf("hashing token: %w", err)
	}

	// Store token
	ttl := s.config.TokenTTL
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	expiresAt := time.Now().Add(ttl)

	tokenID, err := s.db.StoreVerificationToken(ctx, user.ID, user.Email, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("storing token: %w", err)
	}

	// Build verification URL
	verificationURL := fmt.Sprintf("%s/__auth/verify-email?token=%s", s.baseURL, token)

	// Render email template
	siteName := s.config.TemplateVars.SiteName
	if siteName == "" {
		siteName = "Basil"
	}
	siteURL := s.config.TemplateVars.SiteURL
	if siteURL == "" {
		siteURL = s.baseURL
	}

	templateData := email.TemplateData{
		DisplayName:     user.Name,
		VerificationURL: verificationURL,
		TTL:             email.FormatDuration(ttl),
		SiteName:        siteName,
		SiteURL:         siteURL,
	}

	subject, body, err := email.RenderVerificationEmail(templateData)
	if err != nil {
		return fmt.Errorf("rendering email: %w", err)
	}

	// Send email
	msg := &email.Message{
		From:    s.getFromAddress(),
		To:      []string{user.Email},
		Subject: subject,
		Text:    body,
	}

	messageID, err := s.provider.Send(ctx, msg)

	// Log email (success or failure)
	logEntry := &EmailLog{
		UserID:            &user.ID,
		Recipient:         user.Email,
		EmailType:         "verification",
		Provider:          s.provider.Name(),
		ProviderMessageID: &messageID,
		Status:            "sent",
	}

	if err != nil {
		logEntry.Status = "failed"
		errMsg := err.Error()
		logEntry.Error = &errMsg
		s.db.LogEmail(ctx, logEntry)
		return fmt.Errorf("sending email: %w", err)
	}

	s.db.LogEmail(ctx, logEntry)

	// Update token send count
	s.db.IncrementSendCount(ctx, tokenID)

	return nil
}

// SendRecoveryEmail sends an account recovery email to a user
func (s *EmailService) SendRecoveryEmail(ctx context.Context, user *User) error {
	if s == nil || s.provider == nil {
		return nil // Email verification not enabled
	}

	// Generate token
	token, err := GenerateVerificationToken()
	if err != nil {
		return fmt.Errorf("generating token: %w", err)
	}

	// Hash token
	tokenHash, err := HashToken(token)
	if err != nil {
		return fmt.Errorf("hashing token: %w", err)
	}

	// Store token (reuse verification table)
	ttl := s.config.TokenTTL
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	expiresAt := time.Now().Add(ttl)

	_, err = s.db.StoreVerificationToken(ctx, user.ID, user.Email, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("storing recovery token: %w", err)
	}

	// Build recovery URL
	recoveryURL := fmt.Sprintf("%s/__auth/recover/verify?token=%s", s.baseURL, token)

	// Render email template
	siteName := s.config.TemplateVars.SiteName
	if siteName == "" {
		siteName = "Basil"
	}
	siteURL := s.config.TemplateVars.SiteURL
	if siteURL == "" {
		siteURL = s.baseURL
	}

	templateData := email.TemplateData{
		DisplayName: user.Name,
		RecoveryURL: recoveryURL,
		TTL:         email.FormatDuration(ttl),
		SiteName:    siteName,
		SiteURL:     siteURL,
	}

	subject, body, err := email.RenderRecoveryEmail(templateData)
	if err != nil {
		return fmt.Errorf("rendering email: %w", err)
	}

	// Send email
	msg := &email.Message{
		From:    s.getFromAddress(),
		To:      []string{user.Email},
		Subject: subject,
		Text:    body,
	}

	messageID, err := s.provider.Send(ctx, msg)

	// Log email (success or failure)
	logEntry := &EmailLog{
		UserID:            &user.ID,
		Recipient:         user.Email,
		EmailType:         "recovery",
		Provider:          s.provider.Name(),
		ProviderMessageID: &messageID,
		Status:            "sent",
	}

	if err != nil {
		logEntry.Status = "failed"
		errMsg := err.Error()
		logEntry.Error = &errMsg
		s.db.LogEmail(ctx, logEntry)
		return fmt.Errorf("sending email: %w", err)
	}

	s.db.LogEmail(ctx, logEntry)

	return nil
}

// getFromAddress returns the from address based on provider config
func (s *EmailService) getFromAddress() string {
	switch s.config.Provider {
	case "mailgun":
		return s.config.Mailgun.From
	case "resend":
		return s.config.Resend.From
	default:
		return "noreply@example.com"
	}
}
