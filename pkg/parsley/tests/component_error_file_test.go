package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestComponentError_NamesTheFileItIsIn guards BUG-041.
//
// An error raised inside an imported component used to carry a line and column
// but no file. The server then fell back to the handler it was serving, so the
// error was reported against the page that *used* the component, at the
// component's line number — a precise position in the wrong file, pointing at
// whatever innocent code happened to be on that line. The reported case sent
// its author to a <header> tag in a file that was entirely fine.
func TestComponentError_NamesTheFileItIsIn(t *testing.T) {
	dir := t.TempDir()

	// The component's fault is on line 3. The page that uses it is only two
	// lines long, so a line number attributed to the wrong file would be
	// obviously, checkably wrong rather than coincidentally plausible.
	component := filepath.Join(dir, "component.pars")
	if err := os.WriteFile(component, []byte(
		"export let Widget = fn() {\n"+
			"  <div>\n"+
			"    <NoSuchThing/>\n"+
			"  </div>\n"+
			"}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	page := filepath.Join(dir, "page.pars")
	src := "let {Widget} = import @./component.pars\n<Widget/>\n"
	if err := os.WriteFile(page, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	env.Filename = page
	env.RootPath = dir
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected an error for the undefined component, got %T: %s", result, result.Inspect())
	}
	if !strings.Contains(errObj.Message, "NoSuchThing") {
		t.Fatalf("wrong error: %s", errObj.Message)
	}

	if errObj.File == "" {
		t.Fatal("the error carries no file, so the server will blame whichever handler it was serving")
	}
	if filepath.Base(errObj.File) != "component.pars" {
		t.Errorf("error blamed %q; the fault is in component.pars", errObj.File)
	}
	if errObj.Line != 3 {
		t.Errorf("error reported line %d; <NoSuchThing/> is on line 3 of component.pars", errObj.Line)
	}
}
