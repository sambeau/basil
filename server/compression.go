package server

import (
	"compress/gzip"
	"net/http"

	"github.com/klauspost/compress/gzhttp"
	"github.com/sambeau/basil/server/config"
)

// newCompressionHandler wraps an HTTP handler with gzip/zstd compression middleware.
// Returns the original handler if compression is disabled or level is "none".
func newCompressionHandler(h http.Handler, cfg config.CompressionConfig) http.Handler {
	// Compression disabled
	if !cfg.Enabled || cfg.Level == "none" {
		return h
	}

	// Map level string to gzip compression level
	var level int
	switch cfg.Level {
	case "fastest":
		level = gzip.BestSpeed
	case "best":
		level = gzip.BestCompression
	case "default":
		level = gzip.DefaultCompression
	default:
		// Unknown level - use default
		level = gzip.DefaultCompression
	}

	// Create wrapper with options
	// Note: option type is unexported, so we call the option constructors directly
	wrapper, err := gzhttp.NewWrapper(
		gzhttp.MinSize(cfg.MinSize),
		gzhttp.CompressionLevel(level),
	)
	if err != nil {
		// Should not happen with valid options, but return unwrapped if it does
		return h
	}

	return wrapper(h)
}
