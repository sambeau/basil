package server

import (
	"fmt"
	"net/http"
	"strings"
)

// liveReloadScript is injected into HTML responses in dev mode
const liveReloadScript = `<script>
(function() {
  let lastSeq = 0;
  let initializing = true;
  const pollInterval = 1000;
  const initGracePeriod = 2000; // Don't reload for 2s after page load
  
  async function checkForChanges() {
    try {
      const resp = await fetch('/__livereload');
      const data = await resp.json();
      if (lastSeq === 0) {
        lastSeq = data.seq;
        // Start grace period - don't reload for changes that happen right after page load
        setTimeout(function() { initializing = false; }, initGracePeriod);
      } else if (data.seq !== lastSeq && !initializing) {
        console.log('[LiveReload] Change detected, reloading...');
        location.reload();
      } else if (data.seq !== lastSeq) {
        // During grace period, just update lastSeq without reloading
        lastSeq = data.seq;
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
	content := string(w.buffer)
	injected := false
	
	// Try to inject before </body>
	if idx := strings.LastIndex(strings.ToLower(content), "</body>"); idx != -1 {
		content = content[:idx] + liveReloadScript + content[idx:]
		injected = true
	}
	
	// Fallback: inject before </html>
	if !injected {
		if idx := strings.LastIndex(strings.ToLower(content), "</html>"); idx != -1 {
			content = content[:idx] + liveReloadScript + content[idx:]
			injected = true
		}
	}
	
	// Fallback: append at end
	if !injected {
		content = content + liveReloadScript
	}
	
	// Update content length and write
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	if !w.wroteHeader {
		w.wroteHeader = true
		if w.statusCode != 0 {
			w.ResponseWriter.WriteHeader(w.statusCode)
		}
	}
	w.ResponseWriter.Write([]byte(content))
}
