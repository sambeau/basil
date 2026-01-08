package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// ResendProvider sends emails via Resend API
type ResendProvider struct {
	client *resend.Client
	from   string
}

// NewResendProvider creates a new Resend email provider
func NewResendProvider(apiKey, from string) (*ResendProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("resend API key is required")
	}
	if from == "" {
		return nil, fmt.Errorf("from address is required")
	}

	client := resend.NewClient(apiKey)

	return &ResendProvider{
		client: client,
		from:   from,
	}, nil
}

// Send sends an email via Resend
func (p *ResendProvider) Send(ctx context.Context, msg *Message) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message cannot be nil")
	}
	if len(msg.To) == 0 {
		return "", fmt.Errorf("at least one recipient is required")
	}
	if msg.Subject == "" {
		return "", fmt.Errorf("subject is required")
	}
	if msg.Text == "" && msg.HTML == "" {
		return "", fmt.Errorf("text or HTML body is required")
	}

	params := &resend.SendEmailRequest{
		From:    msg.From,
		To:      msg.To,
		Subject: msg.Subject,
		Text:    msg.Text,
		Html:    msg.HTML,
	}

	sent, err := p.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	return sent.Id, nil
}

// Name returns the provider name
func (p *ResendProvider) Name() string {
	return "resend"
}
