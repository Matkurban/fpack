package targets

import (
	"runtime"
	"testing"

	"github.com/Matkurban/fpack/go/internal/runner"
)

func runnerResult(out string) runner.Result    { return runner.Result{Output: out} }
func runnerResultOut(out string) runner.Result { return runner.Result{Output: out} }

// posixPaths skips tests that compare exact command lines containing
// POSIX paths (they model macOS/Linux tools; quoting differs on Windows).
func posixPaths(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("exact POSIX command lines")
	}
}
