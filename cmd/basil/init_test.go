package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "testproject"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run init
	var stdout, stderr bytes.Buffer
	err := runInitCommand(projectPath, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	// Verify structure
	assertFileExists(t, filepath.Join(projectPath, "basil.yaml"))
	assertFileExists(t, filepath.Join(projectPath, ".gitignore"))
	assertFileExists(t, filepath.Join(projectPath, "site", "index.pars"))
	assertDirExists(t, filepath.Join(projectPath, "public"))
	assertDirExists(t, filepath.Join(projectPath, "logs"))
	assertDirExists(t, filepath.Join(projectPath, "db"))

	// Verify YAML content
	yamlContent := readFile(t, filepath.Join(projectPath, "basil.yaml"))
	if !strings.Contains(yamlContent, "site: ./site") {
		t.Error("YAML missing site config")
	}
	if !strings.Contains(yamlContent, "public_dir: ./public") {
		t.Error("YAML missing public_dir config")
	}
	if !strings.Contains(yamlContent, "output: ./logs/basil.log") {
		t.Error("YAML missing logs directory in logging config")
	}
	if !strings.Contains(yamlContent, "sqlite: ./db/data.db") {
		t.Error("YAML missing db directory in sqlite config")
	}

	// Verify .gitignore content
	gitignoreContent := readFile(t, filepath.Join(projectPath, ".gitignore"))
	if !strings.Contains(gitignoreContent, "logs/") {
		t.Error(".gitignore missing logs/ entry")
	}
	if !strings.Contains(gitignoreContent, "db/") {
		t.Error(".gitignore missing db/ entry")
	}
	if !strings.Contains(gitignoreContent, "*.db") {
		t.Error(".gitignore missing *.db entry")
	}

	// Verify index.pars content
	indexContent := readFile(t, filepath.Join(projectPath, "site", "index.pars"))
	if indexContent != "<h1>🌿 Hello from Basil 👋</h1>\n" {
		t.Errorf("unexpected index.pars content: %q", indexContent)
	}

	// Verify success message
	output := stdout.String()
	if !strings.Contains(output, "Created new Basil project") {
		t.Error("success message not printed")
	}
	if !strings.Contains(output, "cd "+projectPath) {
		t.Error("success message missing instructions")
	}
}

func TestInitCommand_FolderIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile")

	// Create a file at the path
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Try to init at this path
	var stdout, stderr bytes.Buffer
	err := runInitCommand(filePath, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when path is a file")
	}
	if !strings.Contains(err.Error(), "is a file, not a folder") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitCommand_FolderNotEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "existing")

	// Create folder with a file
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "existing.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Try to init
	var stdout, stderr bytes.Buffer
	err := runInitCommand(projectPath, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when folder is not empty")
	}
	if !strings.Contains(err.Error(), "is not empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitCommand_FolderEmptyOK(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "empty")

	// Create empty folder
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	// Init should succeed
	var stdout, stderr bytes.Buffer
	err := runInitCommand(projectPath, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInitCommand failed on empty folder: %v", err)
	}

	// Verify files were created
	assertFileExists(t, filepath.Join(projectPath, "basil.yaml"))
}

func TestInitCommand_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Change to tmpDir so relative path works
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Use relative path
	projectPath := "./myproject"

	var stdout, stderr bytes.Buffer
	err = runInitCommand(projectPath, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInitCommand failed with relative path: %v", err)
	}

	// Verify files created
	assertFileExists(t, filepath.Join(tmpDir, "myproject", "basil.yaml"))
}

func TestInitCommand_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "absolute")

	var stdout, stderr bytes.Buffer
	err := runInitCommand(projectPath, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInitCommand failed with absolute path: %v", err)
	}

	// Verify files created
	assertFileExists(t, filepath.Join(projectPath, "basil.yaml"))
}

func TestInitCommand_YAMLContent(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "yamltest")

	var stdout, stderr bytes.Buffer
	err := runInitCommand(projectPath, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	yamlContent := readFile(t, filepath.Join(projectPath, "basil.yaml"))

	// Check all required fields
	requiredFields := []string{
		"server:",
		"host: localhost",
		"port: 8080",
		"site: ./site",
		"public_dir: ./public",
		"logging:",
		"level: info",
	}

	for _, field := range requiredFields {
		if !strings.Contains(yamlContent, field) {
			t.Errorf("YAML missing required field: %s", field)
		}
	}

	// Check for helpful comments
	if !strings.Contains(yamlContent, "Generated by: basil --init") {
		t.Error("YAML missing generation comment")
	}
}

func TestInitCommand_IndexContent(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "indextest")

	var stdout, stderr bytes.Buffer
	err := runInitCommand(projectPath, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	indexContent := readFile(t, filepath.Join(projectPath, "site", "index.pars"))
	expected := "<h1>🌿 Hello from Basil 👋</h1>\n"

	if indexContent != expected {
		t.Errorf("index.pars content incorrect.\nExpected: %q\nGot: %q", expected, indexContent)
	}
}

// Test helpers

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("file does not exist: %s (error: %v)", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("path is a directory, not a file: %s", path)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("directory does not exist: %s (error: %v)", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("path is a file, not a directory: %s", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}
