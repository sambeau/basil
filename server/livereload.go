package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Precompiled regex for case-insensitive tag matching
var (
	bodyTagRe = regexp.MustCompile(`(?i)</body>`)
	htmlTagRe = regexp.MustCompile(`(?i)</html>`)
)

// liveReloadScript is injected into HTML responses in dev mode
const liveReloadScript = `<script>
(function() {
  let lastSeq = -1;  // Use -1 to indicate "not yet initialized"
  const pollInterval = 1000;
  
  async function checkForChanges() {
    try {
      const resp = await fetch('/__livereload');
      const data = await resp.json();
      if (lastSeq === -1) {
        lastSeq = data.seq;
      } else if (data.seq !== lastSeq) {
        console.log('[LiveReload] Change detected, reloading...');
        location.reload();
      }
    } catch (e) {
      // Server might be restarting, retry
    }
    setTimeout(checkForChanges, pollInterval);
  }
  
  // Wait for page to fully load (including images) before starting live reload
  // This prevents reload from aborting in-flight resource requests
  if (document.readyState === 'complete') {
    checkForChanges();
    console.log('[LiveReload] Connected');
  } else {
    window.addEventListener('load', function() {
      checkForChanges();
      console.log('[LiveReload] Connected');
    });
  }
})();
</script>`

// liveReloadHandler serves the live reload polling endpoint
type liveReloadHandler struct {
	server *Server
}

func newLiveReloadHandler(s *Server) *liveReloadHandler {
	return &liveReloadHandler{server: s}
}

func (h *liveReloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	seq := uint64(0)
	if h.server.watcher != nil {
		seq = h.server.watcher.GetChangeSeq()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprintf(w, `{"seq":%d}`, seq)
}

// injectLiveReload wraps a handler to inject the live reload script into HTML responses
func injectLiveReload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer to intercept HTML
		lrw := &liveReloadResponseWriter{
			ResponseWriter: w,
			request:        r,
		}
		next.ServeHTTP(lrw, r)

		// Flush any buffered content
		lrw.flush()
	})
}

// liveReloadResponseWriter buffers HTML responses to inject the script
type liveReloadResponseWriter struct {
	http.ResponseWriter
	request     *http.Request
	buffer      []byte
	statusCode  int
	wroteHeader bool
	isHTML      bool
	checked     bool
}

func (w *liveReloadResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	// Don't write header yet - we need to check content type first
}

func (w *liveReloadResponseWriter) Write(b []byte) (int, error) {
	// Check content type on first write
	if !w.checked {
		w.checked = true
		contentType := w.Header().Get("Content-Type")
		w.isHTML = strings.Contains(contentType, "text/html")
	}

	if w.isHTML {
		// Buffer HTML content
		w.buffer = append(w.buffer, b...)
		return len(b), nil
	}

	// Non-HTML: write directly
	if !w.wroteHeader {
		w.wroteHeader = true
		if w.statusCode != 0 {
			w.ResponseWriter.WriteHeader(w.statusCode)
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *liveReloadResponseWriter) flush() {
	if !w.isHTML || len(w.buffer) == 0 {
		return
	}

	// Inject script before </body> or at end
	content := w.buffer
	injected := false

	// Use regex for case-insensitive search - this returns correct indices
	// into the original byte slice without any length-changing transformations

	// Try to inject before </body>
	if loc := bodyTagRe.FindIndex(content); loc != nil {
		idx := loc[0] // Start of the match
		newContent := make([]byte, 0, len(content)+len(liveReloadScript))
		newContent = append(newContent, content[:idx]...)
		newContent = append(newContent, []byte(liveReloadScript)...)
		newContent = append(newContent, content[idx:]...)
		content = newContent
		injected = true
	}

	// Fallback: inject before </html>
	if !injected {
		if loc := htmlTagRe.FindIndex(content); loc != nil {
			idx := loc[0]
			newContent := make([]byte, 0, len(content)+len(liveReloadScript))
			newContent = append(newContent, content[:idx]...)
			newContent = append(newContent, []byte(liveReloadScript)...)
			newContent = append(newContent, content[idx:]...)
			content = newContent
			injected = true
		}
	}

	// Fallback: append at end
	if !injected {
		content = append(content, []byte(liveReloadScript)...)
	}

	// Update content length and write
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	if !w.wroteHeader {
		w.wroteHeader = true
		if w.statusCode != 0 {
			w.ResponseWriter.WriteHeader(w.statusCode)
		}
	}
	w.ResponseWriter.Write(content)
}
