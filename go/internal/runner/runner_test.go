package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStringRedactsSecrets(t *testing.T) {
	c := Cmd{Name: "flutter", Args: []string{"build", "apk", "--dart-define=TOKEN=s3cret!"},
		Env:    []string{"ORG_GRADLE_PROJECT_android.injected.signing.store.password=hunter22"},
		Secret: []string{"hunter22", "s3cret!"}}
	s := c.String()
	if strings.Contains(s, "hunter22") || strings.Contains(s, "s3cret!") {
		t.Fatalf("secret leaked: %s", s)
	}
	if !strings.Contains(s, "store.password=***") {
		t.Fatalf("missing redaction marker: %s", s)
	}
}

func TestQuote(t *testing.T) {
	if Quote("abc") != "abc" || Quote("a b") != "'a b'" || Quote("it's") != `'it'\''s'` || Quote("") != "''" {
		t.Fatal("quote")
	}
}

func TestRunCapturesTailLogAndExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "x.log")
	var lines []string
	res, err := Run(context.Background(), Cmd{Name: "sh", Args: []string{"-c", "echo one; echo pw=topsecret >&2; printf 'a\\rb\\n'; exit 3"}, Secret: []string{"topsecret"}},
		Options{LogPath: log, OnLine: func(l string) { lines = append(lines, l) }})
	if err == nil || res.ExitCode != 3 {
		t.Fatalf("want exit 3, got %d %v", res.ExitCode, err)
	}
	data, _ := os.ReadFile(log)
	if strings.Contains(string(data), "topsecret") {
		t.Fatal("secret in log")
	}
	if !strings.Contains(strings.Join(res.Tail, "\n"), "pw=***") {
		t.Fatalf("tail: %v", res.Tail)
	}
	if len(lines) != 4 { // one, pw=***, a, b
		t.Fatalf("lines: %q", lines)
	}
}

func TestRunCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell")
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	start := time.Now()
	_, err := Run(ctx, Cmd{Name: "sh", Args: []string{"-c", "sleep 30"}}, Options{})
	if err != ErrInterrupted {
		t.Fatalf("want ErrInterrupted, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancel too slow")
	}
}

func TestRunMissingBinary(t *testing.T) {
	res, err := Run(context.Background(), Cmd{Name: "definitely-not-a-real-tool-xyz"}, Options{})
	if err == nil || res.ExitCode != 127 {
		t.Fatalf("want 127, got %d %v", res.ExitCode, err)
	}
}
