package email

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrProviderNotConfigured = errors.New("email provider not configured")
	ErrInvalidProvider       = errors.New("invalid email provider")
	ErrSendFailed            = errors.New("failed to send email")
)

// Provider sends transactional emails
type Provider interface {
	Send(ctx context.Context, msg *Message) (messageID string, err error)
	Name() string
}

// Message is a provider-agnostic email message
type Message struct {
	From    string
	To      []string
	Subject string
	Text    string // Plain text version
	HTML    string // HTML version (optional)
}
