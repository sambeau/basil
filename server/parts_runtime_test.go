package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPageNeedsPartsRuntime guards BUG-047.
//
// Injection used to be gated on env.ContainsParts, which evalPartTag sets on
// whichever environment is evaluating the tag — never the handler's.
// NewEnclosedEnvironment copies the flag outer→inner and nothing carries it
// back, so a Part anywhere below the top of the program set it on a child the
// handler could not see, and the runtime was never injected. Verified with
// pointers at the time: the tag set it on 0xaf1a7da4000 while the handler read
// 0xaf1a602c840.
//
// The gate now reads the rendered HTML, which is the question actually being
// asked and cannot go out of step with what was rendered.
func TestPageNeedsPartsRuntime(t *testing.T) {
	withPart := `<html><body><footer><div data-part-src="/shows.part" data-part-view="default" data-part-refresh="1000"></div></footer></body></html>`
	if !pageNeedsPartsRuntime(withPart) {
		t.Error("a page containing a Part must get the runtime")
	}

	withoutPart := `<html><body><h1>no parts here</h1></body></html>`
	if pageNeedsPartsRuntime(withoutPart) {
		t.Error("a page with no Part must not get the runtime")
	}

	// The injected runtime must land inside the document, not after it.
	injected := injectPartsRuntime(withPart)
	if !strings.Contains(injected, "refreshIntervals") {
		t.Fatal("injectPartsRuntime did not add the runtime")
	}
	if strings.Index(injected, "refreshIntervals") > strings.Index(injected, "</body>") {
		t.Error("the runtime was injected after </body>")
	}
}

// TestIsPartFragmentRequest guards the second half of BUG-047.
//
// A Part fragment has no </body> and no </html>, so the live-reload
// middleware's final fallback appended its script to the end of the fragment.
// The runtime assigns the response straight into el.innerHTML, so every refresh
// planted a dead script inside the Part — once a second at part-refresh={1000}.
func TestIsPartFragmentRequest(t *testing.T) {
	part := httptest.NewRequest("GET", "/shows.part?_view=default", nil)
	if !isPartFragmentRequest(part) {
		t.Error("a ?_view= request is a Part fragment and must not be given the live-reload script")
	}

	for _, path := range []string{"/", "/shows.part", "/about?x=1"} {
		if isPartFragmentRequest(httptest.NewRequest("GET", path, nil)) {
			t.Errorf("%s is a page and must keep live reload", path)
		}
	}
}
