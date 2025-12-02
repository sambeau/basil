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

// TestPublicDirTransformation tests that paths under public_dir are rewritten to web URLs
func TestPublicDirTransformation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		// Basic transformation: ./public/x -> /x
		{
			name:      "simple_public_path",
			input:     `let p = @./public/images/foo.png; p.string`,
			publicDir: "./public",
			expected:  `/images/foo.png`,
		},
		{
			name:      "public_root_file",
			input:     `let p = @./public/style.css; p.string`,
			publicDir: "./public",
			expected:  `/style.css`,
		},
		{
			name:      "public_directory_itself",
			input:     `let p = @./public; p.string`,
			publicDir: "./public",
			expected:  `/`,
		},

		// Nested public_dir
		{
			name:      "nested_public_dir",
			input:     `let p = @./dist/public/js/app.js; p.string`,
			publicDir: "./dist/public",
			expected:  `/js/app.js`,
		},

		// Paths NOT under public_dir should remain unchanged
		{
			name:      "outside_public_dir",
			input:     `let p = @./data/config.json; p.string`,
			publicDir: "./public",
			expected:  `./data/config.json`,
		},
		{
			name:      "different_prefix",
			input:     `let p = @./public_backup/images/foo.png; p.string`,
			publicDir: "./public",
			expected:  `./public_backup/images/foo.png`,
		},

		// Absolute paths should not be transformed
		{
			name:      "absolute_path_unchanged",
			input:     `let p = @/var/www/public/images/foo.png; p.string`,
			publicDir: "./public",
			expected:  `/var/www/public/images/foo.png`,
		},

		// Without public_dir set, paths remain unchanged
		{
			name:      "no_public_dir_set",
			input:     `let p = @./public/images/foo.png; p.string`,
			publicDir: "",
			expected:  `./public/images/foo.png`,
		},

		// Various public_dir formats should work
		{
			name:      "public_dir_without_dot_slash",
			input:     `let p = @./public/images/foo.png; p.string`,
			publicDir: "public",
			expected:  `/images/foo.png`,
		},
		{
			name:      "public_dir_with_trailing_slash",
			input:     `let p = @./public/images/foo.png; p.string`,
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

// TestPublicDirInHTMLAttributes tests path transformation in HTML context
func TestPublicDirInHTMLAttributes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		{
			name:      "img_src_transformed",
			input:     `let p = @./public/images/photo.jpg; <img src={p}/>`,
			publicDir: "./public",
			expected:  `<img src=/images/photo.jpg />`,
		},
		{
			name:      "link_href_transformed",
			input:     `let css = @./public/css/style.css; <link rel=stylesheet href={css}/>`,
			publicDir: "./public",
			expected:  `<link rel=stylesheet href=/css/style.css />`,
		},
		{
			name:      "script_src_transformed",
			input:     `let js = @./public/js/app.js; <script src={js}/>`,
			publicDir: "./public",
			expected:  `<script src=/js/app.js />`,
		},
		{
			name:      "multiple_paths_transformed",
			input:     `let img = @./public/img/a.png; let css = @./public/style.css; <div><img src={img}/><link href={css}/></div>`,
			publicDir: "./public",
			expected:  `<div><img src=/img/a.png /><link href=/style.css /></div>`,
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
