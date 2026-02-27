package evaluator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sambeau/basil/testenv"
)

// sftp integration tests exercise the full evaluator SFTP path against a real
// (fake) SSH/SFTP server provided by testenv. This addresses the gap noted in
// connection_cache_test.go and introspect_validation_test.go.

// makeSFTPSrc formats a Parsley source string, replacing placeholder tokens
// with real values from the testenv Env:
//   - SFTPURL → full sftp://user:pass@host:port URL
func makeSFTPSrc(env *testenv.Env, template string) string {
	host, port := sftpSplitHostPort(env.SFTPAddr)
	sftpURL := fmt.Sprintf("sftp://%s:%s@%s:%s", env.SFTPUser, env.SFTPPassword, host, port)
	return strings.ReplaceAll(template, "SFTPURL", sftpURL)
}

// sftpSplitHostPort splits "host:port" without importing net (avoids a cycle).
func sftpSplitHostPort(addr string) (host, port string) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, "22"
	}
	return addr[:idx], addr[idx+1:]
}

// TestSFTPEval_Connect verifies that evaluating an @sftp(...) literal returns
// an SFTP_CONNECTION_OBJ when the credentials are correct.
func TestSFTPEval_Connect(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `@sftp("SFTPURL")`)
	result := testEval(src)

	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	if result.Type() != SFTP_CONNECTION_OBJ {
		t.Fatalf("expected SFTP_CONNECTION_OBJ, got %s: %s", result.Type(), result.Inspect())
	}
}

// TestSFTPEval_BadPassword verifies that a wrong password returns an error
// with the "network" error class.
func TestSFTPEval_BadPassword(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	host, port := sftpSplitHostPort(env.SFTPAddr)
	src := fmt.Sprintf(`@sftp("sftp://%s:wrongpassword@%s:%s")`, env.SFTPUser, host, port)
	result := testEval(src)

	if !isError(result) {
		t.Fatalf("expected an error for bad password, got %s", result.Inspect())
	}
	err := result.(*Error)
	if string(err.Class) != "network" {
		t.Errorf("expected error class 'network', got %q", err.Class)
	}
}

// TestSFTPEval_ReadTextFile verifies reading a seeded text file via the fetch
// operator with the .text format accessor.
func TestSFTPEval_ReadTextFile(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/greeting.txt", "hello from sftp")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/greeting.txt).text
data
`)
	result := testEval(src)

	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	str, ok := result.(*String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "hello from sftp" {
		t.Errorf("expected %q, got %q", "hello from sftp", str.Value)
	}
}

// TestSFTPEval_ReadJSONFile verifies reading and parsing a JSON file over SFTP.
func TestSFTPEval_ReadJSONFile(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/data.json", `{"lang":"parsley","version":1}`)

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/data.json).json
data.lang
`)
	result := testEval(src)

	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	str, ok := result.(*String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "parsley" {
		t.Errorf("expected lang=%q, got %q", "parsley", str.Value)
	}
}

// TestSFTPEval_MissingFile verifies that fetching a non-existent file via SFTP
// returns an error in the error-capture pattern rather than panicking.
func TestSFTPEval_MissingFile(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/does-not-exist.txt).text
error
`)
	result := testEval(src)

	if isError(result) {
		t.Fatalf("unexpected hard evaluator error: %s", result.(*Error).Message)
	}
	// error field should be non-null for a missing file
	if result == NULL {
		t.Error("expected non-null error for missing SFTP file, got null")
	}
}

// TestSFTPEval_ConnectionCache verifies that connecting twice with the same
// credentials reuses the cached connection (cache size does not grow beyond 1).
func TestSFTPEval_ConnectionCache(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `@sftp("SFTPURL")`)

	r1 := testEval(src)
	if isError(r1) {
		t.Fatalf("first connect error: %s", r1.(*Error).Message)
	}
	if r1.Type() != SFTP_CONNECTION_OBJ {
		t.Fatalf("expected SFTP_CONNECTION_OBJ, got %s", r1.Type())
	}

	sizeAfterFirst := sftpCache.size()

	r2 := testEval(src)
	if isError(r2) {
		t.Fatalf("second connect error: %s", r2.(*Error).Message)
	}

	sizeAfterSecond := sftpCache.size()

	if sizeAfterSecond > sizeAfterFirst {
		t.Errorf("cache grew on second connection: size was %d, now %d — expected cache hit",
			sizeAfterFirst, sizeAfterSecond)
	}
}

// TestSFTPEval_CacheHealthCheck verifies that when the cached connection is
// dead, a new connection is established and the cache recovers.
func TestSFTPEval_CacheHealthCheck(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `@sftp("SFTPURL")`)

	// Establish a connection and cache it.
	r1 := testEval(src)
	if isError(r1) {
		t.Fatalf("first connect error: %s", r1.(*Error).Message)
	}

	// Forcibly close the underlying clients so the health check will fail.
	conn, ok := r1.(*SFTPConnection)
	if !ok {
		t.Fatalf("expected *SFTPConnection, got %T", r1)
	}
	if conn.Client != nil {
		_ = conn.Client.Close()
	}
	if conn.SSHClient != nil {
		_ = conn.SSHClient.Close()
	}
	conn.Connected = false

	// A second evaluation should detect the dead connection via the health
	// check (Getwd fails) and establish a new one.
	r2 := testEval(src)
	if isError(r2) {
		t.Fatalf("reconnect error after health check eviction: %s", r2.(*Error).Message)
	}

	conn2, ok := r2.(*SFTPConnection)
	if !ok {
		t.Fatalf("expected *SFTPConnection on reconnect, got %T", r2)
	}
	if !conn2.Connected {
		t.Error("expected new connection to be marked as connected")
	}
}
