package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWidth(t *testing.T) {
	if DisplayWidth("abc") != 3 || DisplayWidth("构建") != 4 || DisplayWidth(StripANSI("\x1b[32m✓\x1b[0m")) != 1 {
		t.Fatal("width")
	}
	if got := Truncate("hello world", 6); got != "hello…" {
		t.Fatal(got)
	}
	if got := Pad("构建", 6); got != "构建  " {
		t.Fatalf("%q", got)
	}
}

func TestFormat(t *testing.T) {
	if Duration(1500*time.Millisecond) != "1.5s" || Duration(125*time.Second) != "2m05s" || Duration(300*time.Millisecond) != "300ms" {
		t.Fatal("duration")
	}
	if Size(512) != "512 B" || Size(1536) != "1.5 KB" || Size(20*1024*1024) != "20.0 MB" {
		t.Fatal("size")
	}
}

func TestPlainOutputHasNoANSI(t *testing.T) {
	t.Setenv("FORCE_COLOR", "")
	var b bytes.Buffer
	u := New(Options{Writer: &b})
	u.Step("Build")
	u.Success("ok")
	u.Table([]string{"A", "B"}, [][]string{{"目标", "x"}, {"apk", "yy"}})
	s := b.String()
	if strings.Contains(s, "\x1b[") {
		t.Fatalf("unexpected ANSI in %q", s)
	}
	if !strings.Contains(s, "  目标  x") || !strings.Contains(s, "  apk   yy") {
		t.Fatalf("table misaligned:\n%s", s)
	}
}
