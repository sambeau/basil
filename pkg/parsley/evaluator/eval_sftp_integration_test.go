package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// TestSFTPEval_ListDirectory verifies that fetching a directory path with the
// .dir format accessor returns an array of entry dictionaries, each with a
// "name" key matching the seeded filenames.
func TestSFTPEval_ListDirectory(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/listtest/alpha.txt", "a")
	env.SFTPWriteFile("/listtest/beta.txt", "b")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/listtest).dir
data
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	arr, ok := result.(*Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	names := make(map[string]bool)
	for _, elem := range arr.Elements {
		dict, ok := elem.(*Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary element, got %T", elem)
		}
		nameExpr, exists := dict.Pairs["name"]
		if !exists {
			t.Fatal("directory entry missing 'name' key")
		}
		nameObj := Eval(nameExpr, NewEnvironment())
		if s, ok := nameObj.(*String); ok {
			names[s.Value] = true
		}
	}

	if !names["alpha.txt"] {
		t.Error("expected alpha.txt in directory listing")
	}
	if !names["beta.txt"] {
		t.Error("expected beta.txt in directory listing")
	}
}

// TestSFTPEval_WriteFile verifies that the remote write operator (=/=>) can
// write content to a file on the SFTP server.
func TestSFTPEval_WriteFile(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
"hello from parsley" =/=> conn(@/written.txt).text
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	// Verify the file was written by reading it directly from the temp dir.
	got := env.SFTPReadFile("/written.txt")
	if got != "hello from parsley" {
		t.Errorf("expected file content %q, got %q", "hello from parsley", got)
	}
}

// TestSFTPEval_PermissionDenied verifies that attempting to read a file the
// SFTP server cannot open due to OS permissions returns an error rather than
// panicking or returning null.
func TestSFTPEval_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permission checks cannot fail when running as root")
	}
	if runtime.GOOS == "windows" {
		t.Skip("permission denied semantics differ on Windows")
	}

	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/secret.txt", "classified")

	fullPath := filepath.Join(env.SFTPRoot(), "secret.txt")
	if err := os.Chmod(fullPath, 0o000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(fullPath, 0o644) })

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/secret.txt).text
error
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected hard evaluator error: %s", result.(*Error).Message)
	}
	if result == NULL {
		t.Error("expected non-null error for permission denied, got null")
	}
}

// TestSFTPEval_Mkdir verifies that mkdir() creates a directory on the SFTP server.
func TestSFTPEval_Mkdir(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/newdir).mkdir()
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	info, err := os.Stat(filepath.Join(env.SFTPRoot(), "newdir"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory, got a file")
	}
}

// TestSFTPEval_MkdirParents verifies that mkdir({parents: true}) creates
// nested directories on the SFTP server.
func TestSFTPEval_MkdirParents(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/deep/nested/path).mkdir({parents: true})
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	info, err := os.Stat(filepath.Join(env.SFTPRoot(), "deep", "nested", "path"))
	if err != nil {
		t.Fatalf("nested directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory, got a file")
	}
}

// TestSFTPEval_Rmdir verifies that rmdir() removes an empty directory from
// the SFTP server.
func TestSFTPEval_Rmdir(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	dirPath := filepath.Join(env.SFTPRoot(), "toremove")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/toremove).rmdir()
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Error("directory still exists after rmdir")
	}
}

// TestSFTPEval_RmdirRecursive_Ignored verifies that rmdir({recursive: true}) is
// parsed but behaves the same as non-recursive rmdir (the recursive option is
// not yet implemented — see TODO in eval_network_io.go).
func TestSFTPEval_RmdirRecursive_Ignored(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	// Create a directory with a file inside it.
	dirPath := filepath.Join(env.SFTPRoot(), "notempty")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("setup mkdir failed: %v", err)
	}
	env.SFTPWriteFile("/notempty/child.txt", "content")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/notempty).rmdir({recursive: true})
`)
	result := testEval(src)

	// The recursive flag is parsed but ignored — RemoveDirectory only removes
	// empty directories, so this should return an error.
	if !isError(result) {
		t.Fatal("expected error removing non-empty directory, got success")
	}
}

// TestSFTPEval_Remove verifies that remove() deletes a file from the SFTP server.
func TestSFTPEval_Remove(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/deleteme.txt", "temporary content")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn(@/deleteme.txt).remove()
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	filePath := filepath.Join(env.SFTPRoot(), "deleteme.txt")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file still exists after remove")
	}
}

// TestSFTPEval_Close verifies that close() does not return an error.
func TestSFTPEval_Close(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
conn.close()
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("close failed: %s", result.(*Error).Message)
	}
}

// TestSFTPEval_ReadLines verifies reading a file with the .lines format
// returns an array of strings.
func TestSFTPEval_ReadLines(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/lines.txt", "line1\nline2\nline3")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/lines.txt).lines
data
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	arr, ok := result.(*Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}
	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 lines, got %d", len(arr.Elements))
	}
}

// TestSFTPEval_ReadCSV verifies reading a CSV file with the .csv format
// returns a table with the expected number of rows and header columns.
func TestSFTPEval_ReadCSV(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/data.csv", "name,age\nAlice,30\nBob,25")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/data.csv).csv
data
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	table, ok := result.(*Table)
	if !ok {
		t.Fatalf("expected Table, got %T: %s", result, result.Inspect())
	}
	if len(table.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.Rows))
	}
	// Verify header columns are present
	if len(table.Columns) < 2 || table.Columns[0] != "name" || table.Columns[1] != "age" {
		t.Errorf("expected columns [name age], got %v", table.Columns)
	}
}

// TestSFTPEval_ReadBytes verifies reading a file with the .bytes format
// returns an array of integers.
func TestSFTPEval_ReadBytes(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/binary.bin", "ABC") // bytes 65, 66, 67

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/binary.bin).bytes
data[0]
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	num, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if num.Value != 65 { // 'A' = 65
		t.Errorf("expected 65, got %d", num.Value)
	}
}

// TestSFTPEval_ReadFileAutoDetect verifies reading a .json file with the .file
// format auto-detects and parses the JSON content.
func TestSFTPEval_ReadFileAutoDetect(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/auto.json", `{"key": "value"}`)

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/auto.json).file
data.key
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	str, ok := result.(*String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "value" {
		t.Errorf("expected 'value', got %q", str.Value)
	}
}

// TestSFTPEval_WriteJSON verifies writing a dictionary with the .json format
// produces valid JSON on the SFTP server.
func TestSFTPEval_WriteJSON(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
{name: "test", count: 42} =/=> conn(@/output.json).json
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	got := env.SFTPReadFile("/output.json")
	if !strings.Contains(got, `"name"`) || !strings.Contains(got, `"test"`) {
		t.Errorf("unexpected JSON content: %s", got)
	}
}

// TestSFTPEval_WriteLines verifies writing an array with the .lines format
// produces a newline-delimited file on the SFTP server.
func TestSFTPEval_WriteLines(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let items = ["first", "second", "third"]
items =/=> conn(@/lines.txt).lines
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	got := env.SFTPReadFile("/lines.txt")
	expected := "first\nsecond\nthird\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestSFTPEval_WriteBytes verifies writing an integer array with the .bytes
// format produces the correct binary file on the SFTP server.
func TestSFTPEval_WriteBytes(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let data = [72, 105, 33]
data =/=> conn(@/bytes.bin).bytes
`)
	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	got := env.SFTPReadFile("/bytes.bin")
	if got != "Hi!" { // 72='H', 105='i', 33='!'
		t.Errorf("expected 'Hi!', got %q", got)
	}
}

// TestSFTPEval_WriteCSV_NotImplemented verifies that writing with the .csv
// format returns a SFTP-0003 error.
func TestSFTPEval_WriteCSV_NotImplemented(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let rows = [{a: 1}]
rows =/=> conn(@/data.csv).csv
`)
	result := testEval(src)
	if !isError(result) {
		t.Fatal("expected error for CSV write, got success")
	}
	err := result.(*Error)
	if !strings.Contains(err.Message, "CSV") {
		t.Errorf("expected CSV not implemented error, got: %s", err.Message)
	}
}

// TestSFTPEval_UnknownReadFormat verifies that reading with an unknown format
// captures an error rather than panicking or returning null.
func TestSFTPEval_UnknownReadFormat(t *testing.T) {
	env := testenv.Start(t, testenv.WithSFTP())
	env.SFTPWriteFile("/test.txt", "content")

	src := makeSFTPSrc(env, `
let conn = @sftp("SFTPURL")
let {data, error} <=/= conn(@/test.txt).unknownformat
error
`)
	result := testEval(src)
	if result == NULL {
		t.Error("expected non-null error for unknown SFTP format, got null")
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
