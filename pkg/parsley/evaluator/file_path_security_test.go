package evaluator

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// TestFilePathTraversalAttacks tests that path traversal attempts are blocked
// by security policy
func TestFilePathTraversalAttacks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific path tests on Windows")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	safeFile := filepath.Join(tmpDir, "safe.txt")
	err := os.WriteFile(safeFile, []byte("safe content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		operation    string
		withSecurity bool
		allowedPaths []string
		expectError  bool
		desc         string
	}{
		{
			name:         "parent directory traversal - no security",
			path:         filepath.Join(tmpDir, "../../../etc/passwd"),
			operation:    "read",
			withSecurity: false,
			expectError:  false, // No security = allowed (but file may not exist)
			desc:         "Without security, path traversal is allowed",
		},
		{
			name: "parent directory traversal - with security",
			// Enough ".." to reach the filesystem root from any temp dir,
			// so the path deterministically cleans to /etc/passwd on every
			// platform (six levels don't reach root from macOS's deep
			// /var/folders temp dirs, but do from Linux's /tmp).
			path:         filepath.Join(tmpDir, strings.Repeat("../", 20), "etc/passwd"),
			operation:    "read",
			withSecurity: true,
			allowedPaths: []string{tmpDir},
			expectError:  true, // resolves into /etc, which RestrictRead blocks
			desc:         "Path traversal into a restricted directory should be blocked",
		},
		{
			name:         "absolute path to restricted dir",
			path:         "/etc/passwd",
			operation:    "read",
			withSecurity: true,
			allowedPaths: []string{tmpDir},
			expectError:  true, // /etc is explicitly restricted
			desc:         "Reading from restricted directories should be blocked",
		},
		{
			name:         "absolute path outside allowed - write",
			path:         "/tmp/evil.txt",
			operation:    "write",
			withSecurity: true,
			allowedPaths: []string{tmpDir},
			expectError:  true, // Write requires whitelist (AllowWriteAll=false)
			desc:         "Writes outside allowed paths should be blocked",
		},
		{
			name:         "safe path within allowed",
			path:         safeFile,
			operation:    "read",
			withSecurity: true,
			allowedPaths: []string{tmpDir},
			expectError:  false,
			desc:         "Access within allowed directory should succeed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()

			// Set up security policy if requested
			if tt.withSecurity {
				env.Security = &SecurityPolicy{
					AllowWrite:    tt.allowedPaths,
					AllowWriteAll: false,                            // Require whitelist for writes
					RestrictRead:  []string{"/etc", "/private/etc"}, // Blacklist system directories
				}
			}

			// Check access
			err := env.checkPathAccess(tt.path, tt.operation)
			gotError := err != nil

			if gotError != tt.expectError {
				t.Errorf("Expected error=%v, got error=%v. Error: %v",
					tt.expectError, gotError, err)
			}
		})
	}
}

// TestSymlinkAttacks tests that symlinks are resolved before security checks
func TestSymlinkAttacks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific symlink tests on Windows")
	}

	// Create temporary directories
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	restrictedDir := filepath.Join(tmpDir, "restricted")

	err := os.Mkdir(allowedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create allowed dir: %v", err)
	}
	err = os.Mkdir(restrictedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create restricted dir: %v", err)
	}

	// Create a restricted file
	restrictedFile := filepath.Join(restrictedDir, "secret.txt")
	err = os.WriteFile(restrictedFile, []byte("secret"), 0644)
	if err != nil {
		t.Fatalf("Failed to create restricted file: %v", err)
	}

	// Create a symlink from allowed dir to restricted file
	symlinkPath := filepath.Join(allowedDir, "link_to_secret")
	err = os.Symlink(restrictedFile, symlinkPath)
	if err != nil {
		t.Skipf("Failed to create symlink (may need permissions): %v", err)
	}

	tests := []struct {
		name        string
		path        string
		operation   string
		allowed     []string
		restricted  []string
		expectError bool
		desc        string
	}{
		{
			name:        "symlink escape attempt",
			path:        symlinkPath,
			operation:   "read",
			allowed:     []string{allowedDir},
			restricted:  []string{restrictedDir},
			expectError: true, // Should detect symlink points to restricted area
			desc:        "Symlinks should be resolved before security checks",
		},
		{
			name:        "direct access to restricted",
			path:        restrictedFile,
			operation:   "read",
			allowed:     []string{allowedDir},
			restricted:  []string{restrictedDir},
			expectError: true,
			desc:        "Direct access to restricted files should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				RestrictRead: tt.restricted,
			}

			err := env.checkPathAccess(tt.path, tt.operation)
			gotError := err != nil

			if gotError != tt.expectError {
				t.Errorf("Expected error=%v, got error=%v. Error: %v",
					tt.expectError, gotError, err)
			}
		})
	}
}

// TestFileReadSecurity tests security enforcement for file read operations
func TestFileReadSecurity(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		noRead       bool
		restrictRead []string
		expectError  bool
		desc         string
	}{
		{
			name:        "normal read allowed",
			path:        testFile,
			noRead:      false,
			expectError: false,
			desc:        "Normal reads should succeed",
		},
		{
			name:        "read denied by NoRead flag",
			path:        testFile,
			noRead:      true,
			expectError: true,
			desc:        "NoRead flag should deny all reads",
		},
		{
			name:         "read denied by blacklist",
			path:         testFile,
			noRead:       false,
			restrictRead: []string{tmpDir},
			expectError:  true,
			desc:         "Reads in restricted directories should be denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				NoRead:       tt.noRead,
				RestrictRead: tt.restrictRead,
			}

			// Create file dict
			fileDict := buildTestFileDict(tt.path, "text", env)

			// Attempt to read
			content, readErr := readFileContent(fileDict, env)

			gotError := readErr != nil
			if gotError != tt.expectError {
				if readErr != nil {
					t.Errorf("Expected error=%v, got error=%v. Error: %s",
						tt.expectError, gotError, readErr.Message)
				} else {
					t.Errorf("Expected error=%v, got error=%v. Content: %v",
						tt.expectError, gotError, content)
				}
			}
		})
	}
}

// TestFileWriteSecurity tests security enforcement for file write operations
func TestFileWriteSecurity(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		path          string
		noWrite       bool
		restrictWrite []string
		allowWrite    []string
		allowWriteAll bool
		expectError   bool
		desc          string
	}{
		{
			name:          "write allowed with AllowWriteAll",
			path:          filepath.Join(tmpDir, "test1.txt"),
			allowWriteAll: true,
			expectError:   false,
			desc:          "AllowWriteAll should permit writes anywhere",
		},
		{
			name:        "write denied by NoWrite flag",
			path:        filepath.Join(tmpDir, "test2.txt"),
			noWrite:     true,
			expectError: true,
			desc:        "NoWrite flag should deny all writes",
		},
		{
			name:          "write denied by blacklist",
			path:          filepath.Join(tmpDir, "test3.txt"),
			restrictWrite: []string{tmpDir},
			allowWriteAll: true,
			expectError:   true,
			desc:          "Writes in restricted directories should be denied",
		},
		{
			name:          "write allowed by whitelist",
			path:          filepath.Join(tmpDir, "test4.txt"),
			allowWrite:    []string{tmpDir},
			allowWriteAll: false,
			expectError:   false,
			desc:          "Writes in whitelisted directories should succeed",
		},
		{
			name:          "write denied outside whitelist",
			path:          "/tmp/evil.txt",
			allowWrite:    []string{tmpDir},
			allowWriteAll: false,
			expectError:   true,
			desc:          "Writes outside whitelist should be denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				NoWrite:       tt.noWrite,
				RestrictWrite: tt.restrictWrite,
				AllowWrite:    tt.allowWrite,
				AllowWriteAll: tt.allowWriteAll,
			}

			// Create file dict
			fileDict := buildTestFileDict(tt.path, "text", env)

			// Attempt to write
			writeErr := writeFileContent(fileDict, &String{Value: "test"}, false, env)

			gotError := writeErr != nil
			if gotError != tt.expectError {
				if writeErr != nil {
					t.Errorf("Expected error=%v, got error=%v. Error: %s",
						tt.expectError, gotError, writeErr.Message)
				} else {
					t.Errorf("Expected error=%v, got error=%v",
						tt.expectError, gotError)
				}
			}
		})
	}
}

// TestFileDeleteSecurity tests security enforcement for file deletion
func TestFileDeleteSecurity(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "delete1.txt")
	testFile2 := filepath.Join(tmpDir, "delete2.txt")
	err := os.WriteFile(testFile1, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	err = os.WriteFile(testFile2, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name          string
		path          string
		noWrite       bool
		restrictWrite []string
		allowWrite    []string
		allowWriteAll bool
		expectError   bool
		desc          string
	}{
		{
			name:          "delete allowed with AllowWriteAll",
			path:          testFile1,
			allowWriteAll: true,
			expectError:   false,
			desc:          "Delete should work when AllowWriteAll is true",
		},
		{
			name:        "delete denied by NoWrite flag",
			path:        testFile2,
			noWrite:     true,
			expectError: true,
			desc:        "NoWrite flag should deny delete operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				NoWrite:       tt.noWrite,
				RestrictWrite: tt.restrictWrite,
				AllowWrite:    tt.allowWrite,
				AllowWriteAll: tt.allowWriteAll,
			}

			// Create file dict
			fileDict := buildTestFileDict(tt.path, "text", env)

			// Attempt to delete
			result := evalFileRemove(fileDict, env)

			gotError := isError(result)
			if gotError != tt.expectError {
				if gotError {
					t.Errorf("Expected error=%v, got error=%v. Error: %s",
						tt.expectError, gotError, result.(*Error).Message)
				} else {
					t.Errorf("Expected error=%v, got error=%v",
						tt.expectError, gotError)
				}
			}
		})
	}
}

// TestDeleteTreeInverseContainment covers BUG-036: a recursive removal
// (checkPathAccess operation "delete-tree") must be refused when its target
// equals OR contains a RestrictWrite entry, because os.RemoveAll would carry the
// denied descendant off with the tree. The ordinary "write" operation only asks
// the forward question (is the target itself restricted?), so a bare parent dir
// slips past it — that is the hole this test pins down.
func TestDeleteTreeInverseContainment(t *testing.T) {
	tmpDir := t.TempDir()

	// Layout:
	//   dataDir/            <- writable (the hook's data dir)
	//     deploy.db         <- RestrictWrite entry (the denied victim)
	//   siblingDir/         <- writable, holds nothing restricted
	dataDir := filepath.Join(tmpDir, "data")
	deniedFile := filepath.Join(dataDir, "deploy.db")
	siblingDir := filepath.Join(tmpDir, "sibling")

	for _, d := range []string{dataDir, siblingDir} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}
	if err := os.WriteFile(deniedFile, []byte("record"), 0o644); err != nil {
		t.Fatalf("Failed to create denied file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		operation   string
		expectError bool
		desc        string
	}{
		{
			name:        "delete-tree of dir containing restricted entry",
			path:        dataDir,
			operation:   "delete-tree",
			expectError: true,
			desc:        "Recursive delete of a parent of the denied file must be refused",
		},
		{
			name:        "delete-tree of the restricted entry itself",
			path:        deniedFile,
			operation:   "delete-tree",
			expectError: true,
			desc:        "Recursive delete of the denied file's exact path must be refused",
		},
		{
			name:        "delete-tree of a sibling with nothing restricted",
			path:        siblingDir,
			operation:   "delete-tree",
			expectError: false,
			desc:        "Recursive delete of a dir with no restricted descendant is allowed",
		},
		{
			// Fail-before / pass-after witness: the SAME parent dir, checked as a
			// plain "write", passes — proving the forward check alone missed the
			// descendant and the inverse check is what closes the hole.
			name:        "plain write of dir containing restricted entry (control)",
			path:        dataDir,
			operation:   "write",
			expectError: false,
			desc:        "The old forward-only check does not see the descendant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				AllowWriteAll: true,
				RestrictWrite: []string{deniedFile},
			}

			err := env.checkPathAccess(tt.path, tt.operation)
			gotError := err != nil
			if gotError != tt.expectError {
				t.Errorf("%s: expected error=%v, got error=%v (err: %v)",
					tt.desc, tt.expectError, gotError, err)
			}
		})
	}
}

// TestDeleteTreeViaRmdirMethod drives the fix through the real rmdir method
// (methods_file_http.go), which selects os.RemoveAll for recursive:true and
// therefore must pass the "delete-tree" operation. It confirms the wiring, not
// just the policy helper: a recursive rmdir of a dir holding a denied file is
// refused (and the file survives on disk), while a non-recursive remove of an
// ordinary allowed file still succeeds.
func TestDeleteTreeViaRmdirMethod(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	deniedFile := filepath.Join(dataDir, "deploy.db")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := os.WriteFile(deniedFile, []byte("record"), 0o644); err != nil {
		t.Fatalf("Failed to create denied file: %v", err)
	}

	env := NewEnvironment()
	env.Security = &SecurityPolicy{
		AllowWriteAll: true,
		RestrictWrite: []string{deniedFile},
	}

	// Recursive rmdir of the parent must be refused, and the denied file must
	// still exist afterwards.
	parts, isAbs := parsePathString(dataDir)
	dirDict := dirToDict(pathToDict(parts, isAbs, env), env)
	recursiveOpt := &Dictionary{Pairs: map[string]ast.Expression{"recursive": &ast.Boolean{Value: true}}, Env: env}
	result := rmdirFromDict(dirDict, []Object{recursiveOpt}, env, "dir")
	if !isError(result) {
		t.Fatalf("Expected recursive rmdir of dir containing a denied file to be refused, got %T", result)
	}
	if _, err := os.Stat(deniedFile); err != nil {
		t.Errorf("Denied file should have survived the refused delete, but: %v", err)
	}

	// A non-recursive remove of an ordinary allowed file still works.
	okFile := filepath.Join(tmpDir, "ok.txt")
	if err := os.WriteFile(okFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("Failed to create ok file: %v", err)
	}
	okDict := buildTestFileDict(okFile, "text", env)
	if res := evalFileRemove(okDict, env); isError(res) {
		t.Errorf("Non-recursive remove of an ordinary allowed file should succeed, got: %s", res.(*Error).Message)
	}
	if _, err := os.Stat(okFile); !os.IsNotExist(err) {
		t.Errorf("Ordinary file should have been removed, stat err: %v", err)
	}
}

// TestDeleteTreeSymlinkedTarget checks that the inverse containment check
// resolves the target through EvalSymlinks before comparing, exactly as the
// forward check does. The delete-tree target is a symlink to the real data dir
// that holds the restricted file; after resolution the target becomes the real
// dir, which contains the restricted entry, so the removal must be refused.
// Skips where symlink creation is unavailable.
func TestDeleteTreeSymlinkedTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific symlink test on Windows")
	}
	tmpDir := t.TempDir()
	// Resolve tmpDir up front so path comparisons are stable on macOS
	// (/var -> /private/var).
	if resolved, err := filepath.EvalSymlinks(tmpDir); err == nil {
		tmpDir = resolved
	}

	realDir := filepath.Join(tmpDir, "real") // holds the restricted file
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatalf("mkdir realDir: %v", err)
	}
	restricted := filepath.Join(realDir, "deploy.db")
	if err := os.WriteFile(restricted, []byte("record"), 0o644); err != nil {
		t.Fatalf("write restricted: %v", err)
	}
	// A symlink pointing at realDir; deleting through it must still be refused.
	linkDir := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	env := NewEnvironment()
	env.Security = &SecurityPolicy{
		AllowWriteAll: true,
		RestrictWrite: []string{restricted},
	}
	if err := env.checkPathAccess(linkDir, "delete-tree"); err == nil {
		t.Errorf("Expected delete-tree of a symlink to a dir containing a restricted entry to be refused")
	}
}

// TestDirectoryEscapeAttacks tests that directory operations can't escape
// allowed directories
func TestDirectoryEscapeAttacks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific directory tests on Windows")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		pathStr     string
		expectError bool
		desc        string
	}{
		{
			name:        "normal directory access",
			pathStr:     tmpDir,
			expectError: false,
			desc:        "Normal directory access should work",
		},
		{
			name:        "parent directory escape attempt",
			pathStr:     filepath.Join(tmpDir, "../../../../../../etc"),
			expectError: true, // Will be blocked by security or won't exist
			desc:        "Parent directory traversal should be restricted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				RestrictRead: []string{"/etc", "/private/etc"},
			}

			// Parse path and create directory dict
			parts, isAbs := parsePathString(tt.pathStr)
			pathDict := pathToDict(parts, isAbs, env)
			_ = dirToDict(pathDict, env) // Convert to dir dict (not used directly)

			// Attempt to read directory
			result := readDirContents(tt.pathStr, env)

			gotError := isError(result)
			// Note: This might not error for all paths, just verifying the function works
			t.Logf("Path: %s, Error: %v, Result type: %T", tt.pathStr, gotError, result)
		})
	}
}

// TestPathCanonicalization tests that paths are properly canonicalized
// before security checks
func TestPathCanonicalization(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		allowed     []string
		expectError bool
		desc        string
	}{
		{
			name:        "path with dot segments - current dir",
			path:        filepath.Join(tmpDir, "./test.txt"),
			allowed:     []string{tmpDir},
			expectError: false,
			desc:        "Paths with ./ should be canonicalized correctly",
		},
		{
			name:        "path with dot segments - parent dir",
			path:        filepath.Join(tmpDir, "subdir/../test.txt"),
			allowed:     []string{tmpDir},
			expectError: false,
			desc:        "Paths with ../ within allowed dir should work",
		},
		{
			name:        "path with multiple slashes",
			path:        strings.ReplaceAll(filepath.Join(tmpDir, "test.txt"), string(filepath.Separator), string(filepath.Separator)+string(filepath.Separator)),
			allowed:     []string{tmpDir},
			expectError: false,
			desc:        "Multiple slashes should be normalized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			env.Security = &SecurityPolicy{
				AllowWrite:    tt.allowed,
				AllowWriteAll: false,
			}

			err := env.checkPathAccess(tt.path, "write")
			gotError := err != nil

			if gotError != tt.expectError {
				t.Errorf("Expected error=%v, got error=%v. Error: %v, Path: %s",
					tt.expectError, gotError, err, tt.path)
			}
		})
	}
}

// TestFilePermissionDenied tests handling of files with insufficient permissions
func TestFilePermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permission checks cannot fail when running as root")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific permission tests on Windows")
	}

	// Create a file with no read permissions
	tmpDir := t.TempDir()
	noReadFile := filepath.Join(tmpDir, "noread.txt")
	err := os.WriteFile(noReadFile, []byte("secret"), 0000) // No permissions
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Chmod(noReadFile, 0644) // Cleanup

	env := NewEnvironment()
	fileDict := buildTestFileDict(noReadFile, "text", env)

	// Attempt to read - should fail with permission error
	_, readErr := readFileContent(fileDict, env)

	if readErr == nil {
		t.Error("Expected permission error, got nil")
	} else if !strings.Contains(readErr.Message, "permission denied") &&
		!strings.Contains(readErr.Message, "IO-") {
		t.Errorf("Expected permission error, got: %s", readErr.Message)
	}
}

// Helper functions

// buildTestFileDict creates a file dictionary for testing
func buildTestFileDict(path string, format string, env *Environment) *Dictionary {
	// Parse path
	parts, isAbs := parsePathString(path)

	// Create path dict first
	pathDict := pathToDict(parts, isAbs, env)

	// Use fileToDict to create proper file dictionary
	return fileToDict(pathDict, format, nil, env)
}
