package server

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Embed the prelude directory
//
//go:embed prelude/js/* prelude/css/* prelude/public/* prelude/errors/* prelude/devtools/* prelude/components/*
var preludeFS embed.FS

// preludeASTs stores parsed ASTs for all .pars files in the prelude
var preludeASTs map[string]*ast.Program

// jsAssetHash is the version hash for basil.js (commit or content hash)
var jsAssetHash string

// initPrelude parses all .pars files in the prelude directory at server startup.
// Returns an error if any parse fails (fail-fast).
func initPrelude(commit string) error {
	preludeASTs = make(map[string]*ast.Program)

	// Initialize JS asset hash
	initJSHash(commit)

	// Register the prelude loader with the evaluator for @std/html module
	evaluator.PreludeLoader = GetPreludeAST

	// Walk the embedded filesystem and parse all .pars files
	return fs.WalkDir(preludeFS, "prelude", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(filePath, ".pars") {
			return nil
		}

		source, err := preludeFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", filePath, err)
		}

		l := lexer.New(string(source))
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			return fmt.Errorf("parse error in %s: %v", filePath, p.Errors())
		}

		// Store with relative path as key: "components/text_field.pars"
		key := strings.TrimPrefix(filePath, "prelude/")
		preludeASTs[key] = program

		return nil
	})
}

// initJSHash initializes the version hash for basil.js
func initJSHash(commit string) {
	if commit != "" && commit != "unknown" {
		// Use git commit short hash (first 7 chars)
		if len(commit) > 7 {
			jsAssetHash = commit[:7]
		} else {
			jsAssetHash = commit
		}
	} else {
		// Development fallback: hash the content
		data, err := preludeFS.ReadFile("prelude/js/basil.js")
		if err != nil {
			// Fallback to a default hash if file doesn't exist
			jsAssetHash = "dev"
			return
		}
		h := sha256.Sum256(data)
		jsAssetHash = hex.EncodeToString(h[:])[:7]
	}
}

// JSAssetURL returns the versioned URL for basil.js
func JSAssetURL() string {
	return fmt.Sprintf("/__/js/basil.%s.js", jsAssetHash)
}

// GetPreludeAST returns the parsed AST for a prelude file, or nil if not found
func GetPreludeAST(relativePath string) *ast.Program {
	return preludeASTs[relativePath]
}

// HasPreludeAST checks if a prelude AST exists for the given path
func HasPreludeAST(relativePath string) bool {
	_, exists := preludeASTs[relativePath]
	return exists
}

// handlePreludeAsset serves static assets from the prelude
func (s *Server) handlePreludeAsset(w http.ResponseWriter, r *http.Request) {
	// Determine asset type from path
	var dir string
	switch {
	case strings.HasPrefix(r.URL.Path, "/__/js/"):
		dir = "js"
	case strings.HasPrefix(r.URL.Path, "/__/css/"):
		dir = "css"
	case strings.HasPrefix(r.URL.Path, "/__/public/"):
		dir = "public"
	default:
		http.NotFound(w, r)
		return
	}

	// Extract filename and build path
	filename := strings.TrimPrefix(r.URL.Path, "/__/"+dir+"/")

	// Security: prevent directory traversal
	if strings.Contains(filename, "..") {
		http.NotFound(w, r)
		return
	}

	// Handle versioned basil.js requests
	if dir == "js" && strings.HasPrefix(filename, "basil.") && strings.HasSuffix(filename, ".js") {
		// Strip version hash: basil.{hash}.js -> basil.js
		filename = "basil.js"
	}

	filepath := "prelude/" + dir + "/" + filename

	// Read file from embedded filesystem
	data, err := preludeFS.ReadFile(filepath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set Content-Type
	contentType := mime.TypeByExtension(path.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Override for .js files (mime package returns text/javascript on some systems)
	if path.Ext(filename) == ".js" {
		contentType = "application/javascript"
	}
	w.Header().Set("Content-Type", contentType)

	// Set caching headers
	// For basil.js with hash in original request path, use immutable caching
	originalFilename := strings.TrimPrefix(r.URL.Path, "/__/"+dir+"/")
	if isVersionedAsset(originalFilename) {
		// Versioned assets: cache forever (immutable)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// Unversioned assets: cache for 1 hour
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	w.Write(data)
}

// isVersionedAsset checks if a filename appears to be versioned (has hash)
func isVersionedAsset(filename string) bool {
	// Look for pattern: name.{hash}.ext where hash is 7+ chars
	parts := strings.Split(filename, ".")
	if len(parts) >= 3 {
		// Check if middle part looks like a hash (7+ alphanumeric chars)
		hashPart := parts[len(parts)-2]
		if len(hashPart) >= 7 {
			return true
		}
	}
	return false
}
