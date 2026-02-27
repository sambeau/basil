package testenv

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-smtp"
)

// smtpBackend implements smtp.Backend.
type smtpBackend struct {
	env *Env
}

func (b *smtpBackend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &smtpSession{env: b.env}, nil
}

// smtpSession implements smtp.Session and accumulates one message at a time.
type smtpSession struct {
	env  *Env
	from string
	to   []string
}

func (s *smtpSession) Mail(from string, _ *smtp.MailOptions) error {
	s.from = from
	s.to = nil
	return nil
}

func (s *smtpSession) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *smtpSession) Data(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	subject := parseSubject(raw)
	body := parseBody(raw)

	toCopy := make([]string, len(s.to))
	copy(toCopy, s.to)

	s.env.appendMessage(&Message{
		From:    s.from,
		To:      toCopy,
		Subject: subject,
		Body:    body,
	})
	return nil
}

func (s *smtpSession) Reset() {
	s.from = ""
	s.to = nil
}

func (s *smtpSession) Logout() error {
	return nil
}

// startSMTP starts the fake SMTP server and populates env.SMTPAddr.
func startSMTP(t testing.TB, env *Env) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("testenv: failed to listen for SMTP: %v", err)
	}

	be := &smtpBackend{env: env}
	s := smtp.NewServer(be)
	s.Domain = "localhost"
	s.AllowInsecureAuth = true

	env.SMTPAddr = ln.Addr().String()

	go func() {
		// Serve returns ErrServerClosed on s.Close(); ignore it.
		_ = s.Serve(ln)
	}()

	t.Cleanup(func() {
		_ = s.Close()
	})
}

// parseSubject extracts the Subject header value from a raw RFC 822 message.
func parseSubject(raw []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break // end of headers
		}
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			return strings.TrimSpace(line[len("subject:"):])
		}
	}
	return ""
}

// parseBody returns everything after the blank line separating headers from body.
func parseBody(raw []byte) string {
	idx := bytes.Index(raw, []byte("\r\n\r\n"))
	if idx >= 0 {
		return string(raw[idx+4:])
	}
	// Fallback: some clients use bare \n
	idx = bytes.Index(raw, []byte("\n\n"))
	if idx >= 0 {
		return string(raw[idx+2:])
	}
	return string(raw)
}
