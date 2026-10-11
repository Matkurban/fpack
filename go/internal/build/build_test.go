package build

import (
	"strings"
	"testing"
)

func TestToolError(t *testing.T) {
	tail := []string{"Verifying: app.exe", "SignTool Error: A certificate chain processed, but terminated in a root certificate which is not trusted by the trust provider.", "Number of errors: 1"}
	if got := toolError(tail); !strings.Contains(got, "not trusted") {
		t.Fatal(got)
	}
	if toolError([]string{"ok"}) != "" {
		t.Fatal("no error line")
	}
}
