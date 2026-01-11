package email

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// TemplateData holds variables for email templates
type TemplateData struct {
	DisplayName     string
	VerificationURL string
	RecoveryURL     string
	TTL             string
	SiteName        string
	SiteURL         string
}

// Default email templates (text-only for V1)
const (
	verificationEmailTemplate = `Subject: Verify your email address

Hi {{.DisplayName}},

Please verify your email address by clicking the link below:

{{.VerificationURL}}

This link expires in {{.TTL}}.

If you didn't create an account, you can safely ignore this email.

---
{{.SiteName}}
{{.SiteURL}}`

	recoveryEmailTemplate = `Subject: Account recovery link

Hi {{.DisplayName}},

You requested to recover your account. Click the link below:

{{.RecoveryURL}}

This link expires in {{.TTL}}.

If you didn't request this, please ignore this email.

---
{{.SiteName}}
{{.SiteURL}}`
)

var (
	verificationTmpl *template.Template
	recoveryTmpl     *template.Template
)

func init() {
	var err error
	verificationTmpl, err = template.New("verification").Parse(verificationEmailTemplate)
	if err != nil {
		panic(fmt.Sprintf("failed to parse verification template: %v", err))
	}

	recoveryTmpl, err = template.New("recovery").Parse(recoveryEmailTemplate)
	if err != nil {
		panic(fmt.Sprintf("failed to parse recovery template: %v", err))
	}
}

// RenderVerificationEmail renders the verification email template
func RenderVerificationEmail(data TemplateData) (subject, body string, err error) {
	var buf bytes.Buffer
	if err := verificationTmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("rendering verification email: %w", err)
	}

	text := buf.String()
	// Extract subject line (first line after "Subject: ")
	lines := bytes.SplitN([]byte(text), []byte("\n"), 3)
	if len(lines) < 3 {
		return "", "", fmt.Errorf("invalid email template format")
	}

	subject = string(bytes.TrimPrefix(lines[0], []byte("Subject: ")))
	body = string(lines[2]) // Skip the blank line

	return subject, body, nil
}

// RenderRecoveryEmail renders the recovery email template
func RenderRecoveryEmail(data TemplateData) (subject, body string, err error) {
	var buf bytes.Buffer
	if err := recoveryTmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("rendering recovery email: %w", err)
	}

	text := buf.String()
	// Extract subject line (first line after "Subject: ")
	lines := bytes.SplitN([]byte(text), []byte("\n"), 3)
	if len(lines) < 3 {
		return "", "", fmt.Errorf("invalid email template format")
	}

	subject = string(bytes.TrimPrefix(lines[0], []byte("Subject: ")))
	body = string(lines[2]) // Skip the blank line

	return subject, body, nil
}

// FormatDuration formats a duration for email templates (e.g., "1 hour", "24 hours")
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour"
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}

	minutes := int(d.Minutes())
	if minutes == 1 {
		return "1 minute"
	}
	if minutes > 0 {
		return fmt.Sprintf("%d minutes", minutes)
	}

	return "a few moments"
}
