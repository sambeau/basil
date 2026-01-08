package email

import (
	"context"
	"fmt"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

// MailgunProvider sends emails via Mailgun API
type MailgunProvider struct {
	client *mailgun.MailgunImpl
	from   string
}

// NewMailgunProvider creates a new Mailgun email provider
func NewMailgunProvider(apiKey, domain, from, region string) (*MailgunProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("mailgun API key is required")
	}
	if domain == "" {
		return nil, fmt.Errorf("mailgun domain is required")
	}
	if from == "" {
		return nil, fmt.Errorf("from address is required")
	}

	mg := mailgun.NewMailgun(domain, apiKey)

	// Set EU region if specified
	if region == "eu" {
		mg.SetAPIBase("https://api.eu.mailgun.net/v3")
	}

	return &MailgunProvider{
		client: mg,
		from:   from,
	}, nil
}

// Send sends an email via Mailgun
func (p *MailgunProvider) Send(ctx context.Context, msg *Message) (string, error) {
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

	// Create message
	m := p.client.NewMessage(msg.From, msg.Subject, msg.Text, msg.To...)

	// Add HTML if provided
	if msg.HTML != "" {
		m.SetHtml(msg.HTML)
	}

	// Send with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, id, err := p.client.Send(ctx, m)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	return id, nil
}

// Name returns the provider name
func (p *MailgunProvider) Name() string {
	return "mailgun"
}
