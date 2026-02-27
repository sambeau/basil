package testenv

import (
	"testing"
)

func TestStart_NoOptions(t *testing.T) {
	env := Start(t)
	if env == nil {
		t.Fatal("Start returned nil")
	}
	if env.HTTPSURL != "" {
		t.Error("HTTPSURL should be empty when WithHTTPS not requested")
	}
	if env.SMTPAddr != "" {
		t.Error("SMTPAddr should be empty when WithSMTP not requested")
	}
	if env.SFTPAddr != "" {
		t.Error("SFTPAddr should be empty when WithSFTP not requested")
	}
}

func TestStart_WithHTTPS(t *testing.T) {
	env := Start(t, WithHTTPS())
	if env.HTTPSURL == "" {
		t.Fatal("HTTPSURL should be set when WithHTTPS requested")
	}
	if env.HTTPSClient == nil {
		t.Fatal("HTTPSClient should be set when WithHTTPS requested")
	}
}

func TestStart_WithSMTP(t *testing.T) {
	env := Start(t, WithSMTP())
	if env.SMTPAddr == "" {
		t.Fatal("SMTPAddr should be set when WithSMTP requested")
	}
}

func TestStart_WithSFTP(t *testing.T) {
	env := Start(t, WithSFTP())
	if env.SFTPAddr == "" {
		t.Fatal("SFTPAddr should be set when WithSFTP requested")
	}
	if env.SFTPUser == "" {
		t.Fatal("SFTPUser should be set when WithSFTP requested")
	}
	if env.SFTPPassword == "" {
		t.Fatal("SFTPPassword should be set when WithSFTP requested")
	}
}

func TestStart_AllOptions(t *testing.T) {
	env := Start(t, WithHTTPS(), WithSMTP(), WithSFTP())
	if env.HTTPSURL == "" {
		t.Error("HTTPSURL should be set")
	}
	if env.HTTPSClient == nil {
		t.Error("HTTPSClient should be set")
	}
	if env.SMTPAddr == "" {
		t.Error("SMTPAddr should be set")
	}
	if env.SFTPAddr == "" {
		t.Error("SFTPAddr should be set")
	}
}

func TestMessages_Empty(t *testing.T) {
	env := Start(t, WithSMTP())
	if env.LastMessage() != nil {
		t.Error("LastMessage should be nil when no messages sent")
	}
	msgs := env.Messages()
	if len(msgs) != 0 {
		t.Errorf("Messages should be empty, got %d", len(msgs))
	}
}
