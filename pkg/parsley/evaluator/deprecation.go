// deprecation.go - Deprecation warning infrastructure
//
// Provides a mechanism for emitting deprecation warnings to stderr.
// Each warning is emitted only once per program execution to avoid spam.

package evaluator

import (
	"fmt"
	"os"
	"sync"
)

// deprecationWarnings tracks which warnings have been emitted
var deprecationWarnings sync.Map

// emitDeprecationWarning prints a deprecation warning to stderr.
// Each unique code is only emitted once per program execution.
func emitDeprecationWarning(code, message string) {
	// Check if we've already emitted this warning
	if _, loaded := deprecationWarnings.LoadOrStore(code, true); loaded {
		return // Already emitted
	}

	fmt.Fprintf(os.Stderr, "DEPRECATION WARNING: %s [%s]\n", message, code)
}

// resetDeprecationWarnings clears all tracked warnings (for testing)
func resetDeprecationWarnings() {
	deprecationWarnings = sync.Map{}
}

// ResetDeprecationWarningsForTesting is the exported version for tests in other packages
func ResetDeprecationWarningsForTesting() {
	resetDeprecationWarnings()
}
