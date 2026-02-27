package testenv

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

// Option configures which fake servers are started by Start.
type Option func(*options)

type options struct {
	wantHTTPS bool
	wantSMTP  bool
	wantSFTP  bool
}

// WithHTTPS requests a fake HTTPS server.
func WithHTTPS() Option { return func(o *options) { o.wantHTTPS = true } }

// WithSMTP requests a fake SMTP server.
func WithSMTP() Option { return func(o *options) { o.wantSMTP = true } }

// WithSFTP requests a fake SFTP server.
func WithSFTP() Option { return func(o *options) { o.wantSFTP = true } }

// Message is an email message captured by the fake SMTP server.
type Message struct {
	From    string
	To      []string
	Subject string
	Body    string
}

// Env holds addresses and credentials for all running fake servers.
// Fields are populated only for the servers that were requested via options.
type Env struct {
	// HTTPS
	HTTPSURL    string       // e.g. "https://127.0.0.1:PORT"
	HTTPSClient *http.Client // pre-configured to trust the self-signed cert

	// SMTP
	SMTPAddr string // e.g. "127.0.0.1:PORT"

	// SFTP
	SFTPAddr     string // e.g. "127.0.0.1:PORT"
	SFTPUser     string
	SFTPPassword string

	// internal state
	httpsMux interface{ Handle(string, http.Handler) } // *http.ServeMux
	sftpRoot string                                    // temp dir served by fake SFTP

	smtpMu   sync.Mutex
	messages []*Message
}

// Messages returns a copy of all email messages captured so far.
func (e *Env) Messages() []*Message {
	e.smtpMu.Lock()
	defer e.smtpMu.Unlock()
	out := make([]*Message, len(e.messages))
	copy(out, e.messages)
	return out
}

// LastMessage returns the most recently captured email message, or nil.
func (e *Env) LastMessage() *Message {
	e.smtpMu.Lock()
	defer e.smtpMu.Unlock()
	if len(e.messages) == 0 {
		return nil
	}
	return e.messages[len(e.messages)-1]
}

// WaitForMessage polls until at least one message has arrived or the timeout
// elapses. It calls t.Fatal on timeout.
func (e *Env) WaitForMessage(t testing.TB, timeout time.Duration) *Message {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m := e.LastMessage(); m != nil {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("testenv: timed out after %s waiting for an SMTP message", timeout)
	return nil
}

// appendMessage is called by the SMTP backend to store a captured message.
func (e *Env) appendMessage(m *Message) {
	e.smtpMu.Lock()
	defer e.smtpMu.Unlock()
	e.messages = append(e.messages, m)
}

// Start launches the requested fake servers and registers t.Cleanup to stop
// them when the test ends.
func Start(t testing.TB, opts ...Option) *Env {
	t.Helper()

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	env := &Env{}

	if o.wantHTTPS {
		startHTTPS(t, env)
	}
	if o.wantSMTP {
		startSMTP(t, env)
	}
	if o.wantSFTP {
		startSFTP(t, env)
	}

	return env
}
