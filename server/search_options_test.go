package server

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

func optsDict(pairs map[string]evaluator.Object) *evaluator.Dictionary {
	return evaluator.NewDictionaryFromObjects(pairs)
}

func TestParseSearchOptions(t *testing.T) {
	env := evaluator.NewEnvironment()

	t.Run("defaults", func(t *testing.T) {
		opts, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"path": &evaluator.String{Value: "search.db"},
		}), env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Path != "search.db" {
			t.Errorf("expected path search.db, got %q", opts.Path)
		}
		if opts.SnippetLen != 200 {
			t.Errorf("expected default snippetLength 200, got %d", opts.SnippetLen)
		}
		if opts.HighlightTag != "mark" {
			t.Errorf("expected default highlightTag mark, got %q", opts.HighlightTag)
		}
	})

	t.Run("path defaults from watch dir", func(t *testing.T) {
		opts, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"watch": &evaluator.String{Value: "content/docs"},
		}), env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Path != "content/docs_search.db" {
			t.Errorf("expected content/docs_search.db, got %q", opts.Path)
		}
	})

	t.Run("snippetLength and highlightTag", func(t *testing.T) {
		opts, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"path":          &evaluator.String{Value: ":memory:"},
			"snippetLength": &evaluator.Integer{Value: 100},
			"highlightTag":  &evaluator.String{Value: "em"},
		}), env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.SnippetLen != 100 {
			t.Errorf("expected snippetLength 100, got %d", opts.SnippetLen)
		}
		if opts.HighlightTag != "em" {
			t.Errorf("expected highlightTag em, got %q", opts.HighlightTag)
		}
	})

	t.Run("rejects non-positive snippetLength", func(t *testing.T) {
		_, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"snippetLength": &evaluator.Integer{Value: 0},
		}), env)
		if err == nil {
			t.Fatal("expected error for snippetLength 0")
		}
	})

	t.Run("rejects invalid highlightTag", func(t *testing.T) {
		_, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"highlightTag": &evaluator.String{Value: "<mark>"},
		}), env)
		if err == nil {
			t.Fatal("expected error for highlightTag with angle brackets")
		}
	})

	t.Run("backend suggests path", func(t *testing.T) {
		_, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"backend": &evaluator.String{Value: "search.db"},
		}), env)
		if err == nil || !strings.Contains(err.Error(), `did you mean "path"?`) {
			t.Fatalf("expected backend error suggesting path, got %v", err)
		}
	})

	t.Run("rejects unknown option", func(t *testing.T) {
		_, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"snipetLength": &evaluator.Integer{Value: 100},
		}), env)
		if err == nil || !strings.Contains(err.Error(), `"snipetLength"`) {
			t.Fatalf("expected unknown option error, got %v", err)
		}
		if !strings.Contains(err.Error(), "valid options") {
			t.Fatalf("expected valid options in error, got %v", err)
		}
	})

	t.Run("rejects unknown weights key", func(t *testing.T) {
		_, err := parseSearchOptions(optsDict(map[string]evaluator.Object{
			"weights": evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
				"titel": &evaluator.Float{Value: 10.0},
			}),
		}), env)
		if err == nil || !strings.Contains(err.Error(), `"titel"`) {
			t.Fatalf("expected unknown weights key error, got %v", err)
		}
	})
}
