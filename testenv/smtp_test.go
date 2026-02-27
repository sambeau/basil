package testenv

import (
	"fmt"
	"net/smtp"
	"testing"
	"time"
)

// sendTestMail sends a single plain-text email to the fake SMTP server using
// the standard library net/smtp client. No auth required (AllowInsecureAuth is
// set, but the fake backend doesn't demand it).
func sendTestMail(addr, from string, to []string, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		from, to[0], subject, body)
	return smtp.SendMail(addr, nil, from, to, []byte(msg))
}

func TestSMTP_ReceiveMessage(t *testing.T) {
	env := Start(t, WithSMTP())

	err := sendTestMail(
		env.SMTPAddr,
		"sender@example.com",
		[]string{"recipient@example.com"},
		"Hello Basil",
		"This is the body.",
	)
	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}

	msg := env.WaitForMessage(t, 2*time.Second)

	if msg.From != "sender@example.com" {
		t.Errorf("expected From=sender@example.com, got %q", msg.From)
	}
	if len(msg.To) == 0 || msg.To[0] != "recipient@example.com" {
		t.Errorf("expected To=[recipient@example.com], got %v", msg.To)
	}
	if msg.Subject != "Hello Basil" {
		t.Errorf("expected Subject=%q, got %q", "Hello Basil", msg.Subject)
	}
}

func TestSMTP_MultipleMessages(t *testing.T) {
	env := Start(t, WithSMTP())

	for i := range 3 {
		err := sendTestMail(
			env.SMTPAddr,
			"sender@example.com",
			[]string{"recipient@example.com"},
			fmt.Sprintf("Message %d", i),
			"body",
		)
		if err != nil {
			t.Fatalf("SendMail %d failed: %v", i, err)
		}
	}

	// Wait for the last message to arrive.
	env.WaitForMessage(t, 2*time.Second)

	// Poll briefly to let all three settle.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(env.Messages()) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	msgs := env.Messages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestSMTP_WaitForMessage_Async(t *testing.T) {
	env := Start(t, WithSMTP())

	// Send asynchronously after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = sendTestMail(
			env.SMTPAddr,
			"async@example.com",
			[]string{"dest@example.com"},
			"Async",
			"async body",
		)
	}()

	msg := env.WaitForMessage(t, 2*time.Second)
	if msg.From != "async@example.com" {
		t.Errorf("expected From=async@example.com, got %q", msg.From)
	}
}

func TestSMTP_NoMessages(t *testing.T) {
	env := Start(t, WithSMTP())

	if env.LastMessage() != nil {
		t.Error("expected LastMessage to be nil before any messages are sent")
	}
	if len(env.Messages()) != 0 {
		t.Errorf("expected 0 messages, got %d", len(env.Messages()))
	}
}

func TestSMTP_MessagesReturnsCopy(t *testing.T) {
	env := Start(t, WithSMTP())

	err := sendTestMail(
		env.SMTPAddr,
		"a@example.com",
		[]string{"b@example.com"},
		"Copy test",
		"body",
	)
	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
	env.WaitForMessage(t, 2*time.Second)

	// Mutating the returned slice must not affect the internal store.
	msgs := env.Messages()
	msgs[0] = nil

	if env.LastMessage() == nil {
		t.Error("mutating returned slice corrupted internal message store")
	}
}
