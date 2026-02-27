package testenv

import (
	"io"
	"strings"
	"testing"
)

func TestSFTP_Connect(t *testing.T) {
	env := Start(t, WithSFTP())

	conn, err := dialSFTP(env.SFTPAddr, env.SFTPUser, env.SFTPPassword)
	if err != nil {
		t.Fatalf("dialSFTP failed: %v", err)
	}
	defer conn.Close()

	// Getwd should succeed on a live connection.
	_, err = conn.SFTP.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
}

func TestSFTP_BadPassword(t *testing.T) {
	env := Start(t, WithSFTP())

	_, err := dialSFTP(env.SFTPAddr, env.SFTPUser, "wrongpassword")
	if err == nil {
		t.Fatal("expected auth error with wrong password, got nil")
	}
}

func TestSFTP_WriteAndRead(t *testing.T) {
	env := Start(t, WithSFTP())

	env.SFTPWriteFile("/hello.txt", "hello basil")

	conn, err := dialSFTP(env.SFTPAddr, env.SFTPUser, env.SFTPPassword)
	if err != nil {
		t.Fatalf("dialSFTP failed: %v", err)
	}
	defer conn.Close()

	f, err := conn.SFTP.Open("/hello.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "hello basil" {
		t.Errorf("expected %q, got %q", "hello basil", string(data))
	}
}

func TestSFTP_List(t *testing.T) {
	env := Start(t, WithSFTP())

	env.SFTPWriteFile("/a.txt", "aaa")
	env.SFTPWriteFile("/b.txt", "bbb")

	conn, err := dialSFTP(env.SFTPAddr, env.SFTPUser, env.SFTPPassword)
	if err != nil {
		t.Fatalf("dialSFTP failed: %v", err)
	}
	defer conn.Close()

	entries, err := conn.SFTP.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names["a.txt"] {
		t.Error("expected a.txt in directory listing")
	}
	if !names["b.txt"] {
		t.Error("expected b.txt in directory listing")
	}
}

func TestSFTP_MissingFile(t *testing.T) {
	env := Start(t, WithSFTP())

	conn, err := dialSFTP(env.SFTPAddr, env.SFTPUser, env.SFTPPassword)
	if err != nil {
		t.Fatalf("dialSFTP failed: %v", err)
	}
	defer conn.Close()

	_, err = conn.SFTP.Open("/nonexistent.txt")
	if err == nil {
		t.Fatal("expected error opening nonexistent file, got nil")
	}
}

func TestSFTP_WriteViaClient(t *testing.T) {
	env := Start(t, WithSFTP())

	conn, err := dialSFTP(env.SFTPAddr, env.SFTPUser, env.SFTPPassword)
	if err != nil {
		t.Fatalf("dialSFTP failed: %v", err)
	}
	defer conn.Close()

	f, err := conn.SFTP.Create("/output.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := f.Write([]byte("written by client")); err != nil {
		_ = f.Close()
		t.Fatalf("Write failed: %v", err)
	}
	_ = f.Close()

	got := env.SFTPReadFile("/output.txt")
	if got != "written by client" {
		t.Errorf("expected %q, got %q", "written by client", got)
	}
}

func TestSFTP_SFTPReadFile_Panics_OnMissing(t *testing.T) {
	env := Start(t, WithSFTP())

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected SFTPReadFile to panic for missing file, but it did not")
		} else {
			msg := ""
			if s, ok := r.(string); ok {
				msg = s
			}
			if !strings.Contains(msg, "SFTPReadFile") {
				t.Errorf("expected panic message to contain 'SFTPReadFile', got %q", msg)
			}
		}
	}()

	env.SFTPReadFile("/does-not-exist.txt")
}
