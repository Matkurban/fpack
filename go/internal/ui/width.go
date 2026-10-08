package ui

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// StripANSI removes ANSI escape sequences.
func StripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// RuneWidth returns the terminal column width of r (CJK and fullwidth are 2).
func RuneWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 32 || (r >= 0x7f && r < 0xa0):
		return 0
	case r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe30 && r <= 0xfe4f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1f64f) ||
		(r >= 0x20000 && r <= 0x3fffd)):
		return 2
	}
	return 1
}

// DisplayWidth is the column width of s (no ANSI codes expected).
func DisplayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += RuneWidth(r)
	}
	return w
}

// Truncate shortens s to at most max columns, adding an ellipsis.
func Truncate(s string, max int) string {
	if max <= 1 || DisplayWidth(s) <= max {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := RuneWidth(r)
		if w+rw > max-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	b.WriteString("…")
	return b.String()
}

// Pad right-pads s (which may contain ANSI codes) to width columns.
func Pad(s string, width int) string {
	n := DisplayWidth(StripANSI(s))
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

var _ = utf8.RuneLen
