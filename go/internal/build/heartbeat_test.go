package build

import (
	"strings"
	"testing"
	"time"
)

func TestHeartbeatMarksStaleOutput(t *testing.T) {
	fresh := heartbeat(2*time.Minute, "Running Gradle task 'bundleRelease'...", 5*time.Second)
	if strings.Contains(fresh, "no output") || !strings.Contains(fresh, "bundleRelease") {
		t.Fatal(fresh)
	}
	stale := heartbeat(3*time.Minute, "Running pod install...   1,379ms", 2*time.Minute+30*time.Second)
	if !strings.Contains(stale, "no output for 2m30s") || !strings.Contains(stale, "pod install") {
		t.Fatal(stale)
	}
	if got := heartbeat(time.Minute, "", time.Minute); strings.Contains(got, "last") {
		t.Fatal(got)
	}
}
