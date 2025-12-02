package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Helper function that creates an environment with basil.public_dir set
func evalWithPublicDir(t *testing.T, input string, publicDir string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()

	// Set up basil.public_dir if provided
	if publicDir != "" {
		basilDict := &evaluator.Dictionary{
			Pairs: map[string]ast.Expression{
				"public_dir": &ast.StringLiteral{Value: publicDir},
			},
			Env: env,
		}
		env.SetProtected("basil", basilDict)
	}

	return evaluator.Eval(program, env)
}

// TestAssetFunction tests that url() transforms paths under public_dir to web URLs
func TestAssetFunction(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		// Basic transformation: asset(@./public/x) -> /x
		{
			name:      "simple_public_path",
			input:     `asset(@./public/images/foo.png)`,
			publicDir: "./public",
			expected:  `/images/foo.png`,
		},
		{
			name:      "public_root_file",
			input:     `asset(@./public/style.css)`,
			publicDir: "./public",
			expected:  `/style.css`,
		},
		{
			name:      "public_directory_itself",
			input:     `asset(@./public)`,
			publicDir: "./public",
			expected:  `/`,
		},

		// Nested public_dir
		{
			name:      "nested_public_dir",
			input:     `asset(@./dist/public/js/app.js)`,
			publicDir: "./dist/public",
			expected:  `/js/app.js`,
		},

		// Paths NOT under public_dir should remain unchanged
		{
			name:      "outside_public_dir",
			input:     `asset(@./data/config.json)`,
			publicDir: "./public",
			expected:  `./data/config.json`,
		},
		{
			name:      "different_prefix",
			input:     `asset(@./public_backup/images/foo.png)`,
			publicDir: "./public",
			expected:  `./public_backup/images/foo.png`,
		},

		// Absolute paths should not be transformed
		{
			name:      "absolute_path_unchanged",
			input:     `asset(@/var/www/public/images/foo.png)`,
			publicDir: "./public",
			expected:  `/var/www/public/images/foo.png`,
		},

		// Without public_dir set, paths remain unchanged
		{
			name:      "no_public_dir_set",
			input:     `asset(@./public/images/foo.png)`,
			publicDir: "",
			expected:  `./public/images/foo.png`,
		},

		// Various public_dir formats should work
		{
			name:      "public_dir_without_dot_slash",
			input:     `asset(@./public/images/foo.png)`,
			publicDir: "public",
			expected:  `/images/foo.png`,
		},
		{
			name:      "public_dir_with_trailing_slash",
			input:     `asset(@./public/images/foo.png)`,
			publicDir: "./public/",
			expected:  `/images/foo.png`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalWithPublicDir(t, tt.input, tt.publicDir)
			if result.Inspect() != tt.expected {
				t.Errorf("for input %s with public_dir=%q:\nexpected: %s\n     got: %s",
					tt.input, tt.publicDir, tt.expected, result.Inspect())
			}
		})
	}
}

// TestPathsWithoutUrlRemainUnchanged tests that paths without url() are NOT transformed
func TestPathsWithoutUrlRemainUnchanged(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		{
			name:      "path_string_not_transformed",
			input:     `let p = @./public/images/foo.png; p.string`,
			publicDir: "./public",
			expected:  `./public/images/foo.png`,
		},
		{
			name:      "path_in_html_not_transformed",
			input:     `let p = @./public/images/photo.jpg; <img src={p}/>`,
			publicDir: "./public",
			expected:  `<img src=./public/images/photo.jpg />`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalWithPublicDir(t, tt.input, tt.publicDir)
			if result.Inspect() != tt.expected {
				t.Errorf("for input %s with public_dir=%q:\nexpected: %s\n     got: %s",
					tt.input, tt.publicDir, tt.expected, result.Inspect())
			}
		})
	}
}

// TestAssetInHTMLAttributes tests url() function in HTML context
func TestAssetInHTMLAttributes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		{
			name:      "img_src_with_url",
			input:     `let p = @./public/images/photo.jpg; <img src={asset(p)}/>`,
			publicDir: "./public",
			expected:  `<img src=/images/photo.jpg />`,
		},
		{
			name:      "link_href_with_url",
			input:     `let css = @./public/css/style.css; <link rel=stylesheet href={asset(css)}/>`,
			publicDir: "./public",
			expected:  `<link rel=stylesheet href=/css/style.css />`,
		},
		{
			name:      "script_src_with_url",
			input:     `let js = @./public/js/app.js; <script src={asset(js)}/>`,
			publicDir: "./public",
			expected:  `<script src=/js/app.js />`,
		},
		{
			name:      "inline_url_call",
			input:     `<img src={asset(@./public/img/a.png)}/>`,
			publicDir: "./public",
			expected:  `<img src=/img/a.png />`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalWithPublicDir(t, tt.input, tt.publicDir)
			if result.Inspect() != tt.expected {
				t.Errorf("for input:\n%s\nwith public_dir=%q:\nexpected: %s\n     got: %s",
					tt.input, tt.publicDir, tt.expected, result.Inspect())
			}
		})
	}
}

// TestAssetWithAbsolutePaths tests that asset() works with absolute paths
// This simulates what happens when files(@~/public/images/*) returns absolute paths
func TestAssetWithAbsolutePaths(t *testing.T) {
	// Create a helper that sets up an environment with RootPath
	evalWithRootPath := func(t *testing.T, input string, publicDir string, rootPath string) evaluator.Object {
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		env.RootPath = rootPath

		if publicDir != "" {
			basilDict := &evaluator.Dictionary{
				Pairs: map[string]ast.Expression{
					"public_dir": &ast.StringLiteral{Value: publicDir},
				},
				Env: env,
			}
			env.SetProtected("basil", basilDict)
		}

		return evaluator.Eval(program, env)
	}

	tests := []struct {
		name      string
		input     string
		publicDir string
		rootPath  string
		expected  string
	}{
		{
			name:      "absolute_path_under_public_dir",
			input:     `asset(@/project/public/images/foo.png)`,
			publicDir: "./public",
			rootPath:  "/project",
			expected:  `/images/foo.png`,
		},
		{
			name:      "absolute_path_nested_under_public_dir",
			input:     `asset(@/project/app/public/images/tubs/tub1.png)`,
			publicDir: "./app/public",
			rootPath:  "/project",
			expected:  `/images/tubs/tub1.png`,
		},
		{
			name:      "absolute_path_not_under_public_dir",
			input:     `asset(@/other/path/file.txt)`,
			publicDir: "./public",
			rootPath:  "/project",
			expected:  `/other/path/file.txt`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalWithRootPath(t, tt.input, tt.publicDir, tt.rootPath)
			if result.Inspect() != tt.expected {
				t.Errorf("for input %s with public_dir=%q, rootPath=%q:\nexpected: %s\n     got: %s",
					tt.input, tt.publicDir, tt.rootPath, tt.expected, result.Inspect())
			}
		})
	}
}

// TestFileDictToString tests that file dictionaries stringify correctly
func TestFileDictToString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "absolute_path_concatenation",
			input:    `"path: " + @/usr/local/bin`,
			expected: `path: /usr/local/bin`,
		},
		{
			name:     "relative_path_concatenation",
			input:    `"path: " + @./some/file.txt`,
			expected: `path: ./some/file.txt`,
		},
		{
			name:     "home_path_concatenation",
			input:    `"path: " + @~/documents/file.txt`,
			expected: `path: ~/documents/file.txt`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result.Inspect() != tt.expected {
				t.Errorf("for input %s:\nexpected: %s\n     got: %s",
					tt.input, tt.expected, result.Inspect())
			}
		})
	}
}
