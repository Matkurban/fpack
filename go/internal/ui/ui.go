// Package ui renders human output: colored step lines and spinners on an
// interactive terminal, plain timestamp-free log lines everywhere else (CI,
// pipes, NO_COLOR). All human output goes through a single *UI.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// UI writes human readable output.
type UI struct {
	mu      sync.Mutex
	w       io.Writer
	color   bool
	tty     bool
	spin    *Spinner
	Verbose bool
}

// Options configure New.
type Options struct {
	Writer  io.Writer // defaults to stdout
	NoColor bool
	Verbose bool
}

// New creates a UI. Colors are enabled only on a terminal, and disabled by
// NO_COLOR, FPACK_NO_COLOR, TERM=dumb or Options.NoColor. FORCE_COLOR forces them.
func New(o Options) *UI {
	w := o.Writer
	if w == nil {
		w = os.Stdout
	}
	tty := false
	if f, ok := w.(*os.File); ok {
		tty = isTerminal(f)
		if tty {
			enableVT(f)
		}
	}
	color := tty && !o.NoColor && os.Getenv("NO_COLOR") == "" && os.Getenv("FPACK_NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if fc := os.Getenv("FORCE_COLOR"); fc != "" && fc != "0" && fc != "false" && !o.NoColor {
		color = true
	}
	return &UI{w: w, color: color, tty: tty && os.Getenv("TERM") != "dumb", Verbose: o.Verbose}
}

// Discard returns a UI that prints nothing (tests).
func Discard() *UI { return &UI{w: io.Discard} }

// Writer is the underlying writer.
func (u *UI) Writer() io.Writer { return u.w }

// Interactive reports a real terminal (spinners allowed).
func (u *UI) Interactive() bool { return u.tty }

// Color reports whether ANSI colors are emitted.
func (u *UI) Color() bool { return u.color }

const (
	cReset  = "\x1b[0m"
	cBold   = "\x1b[1m"
	cDim    = "\x1b[2m"
	cRed    = "\x1b[31m"
	cGreen  = "\x1b[32m"
	cYellow = "\x1b[33m"
	cBlue   = "\x1b[34m"
	cCyan   = "\x1b[36m"
)

func (u *UI) paint(code, s string) string {
	if !u.color || s == "" {
		return s
	}
	return code + s + cReset
}

// Bold, Dim, Red, Green, Yellow, Cyan style a string.
func (u *UI) Bold(s string) string   { return u.paint(cBold, s) }
func (u *UI) Dim(s string) string    { return u.paint(cDim, s) }
func (u *UI) Red(s string) string    { return u.paint(cRed, s) }
func (u *UI) Green(s string) string  { return u.paint(cGreen, s) }
func (u *UI) Yellow(s string) string { return u.paint(cYellow, s) }
func (u *UI) Cyan(s string) string   { return u.paint(cCyan, s) }
func (u *UI) Blue(s string) string   { return u.paint(cBlue, s) }

// Println prints a raw line (pausing any spinner).
func (u *UI) Println(s string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.clearSpinLocked()
	fmt.Fprintln(u.w, s)
	u.redrawSpinLocked()
}

// Printf prints formatted raw text followed by a newline.
func (u *UI) Printf(format string, a ...any) { u.Println(fmt.Sprintf(format, a...)) }

// Blank prints an empty line.
func (u *UI) Blank() { u.Println("") }

// Title prints a bold section header.
func (u *UI) Title(s string) { u.Println(u.Bold(s)) }

// Step prints the start of a major step.
func (u *UI) Step(s string) { u.Println(u.Cyan("▶") + " " + u.Bold(s)) }

// Success prints a success line.
func (u *UI) Success(s string) { u.Println("  " + u.Green("✓") + " " + s) }

// Warn prints a warning line.
func (u *UI) Warn(s string) { u.Println("  " + u.Yellow("!") + " " + s) }

// Fail prints an error line.
func (u *UI) Fail(s string) { u.Println("  " + u.Red("✗") + " " + s) }

// Skip prints a skipped line.
func (u *UI) Skip(s string) { u.Println("  " + u.Dim("–") + " " + s) }

// Info prints an indented informational line.
func (u *UI) Info(s string) { u.Println("  " + s) }

// Detail prints a dim, further indented line.
func (u *UI) Detail(s string) { u.Println("    " + u.Dim(s)) }

// Hint prints a highlighted suggestion.
func (u *UI) Hint(s string) {
	for i, line := range strings.Split(s, "\n") {
		prefix := "    "
		if i == 0 {
			prefix = "  " + u.Yellow("→") + " "
		}
		u.Println(prefix + line)
	}
}

// Errorf prints a top-level error.
func (u *UI) Errorf(format string, a ...any) {
	u.Println(u.Red(u.Bold("error:")) + " " + fmt.Sprintf(format, a...))
}

// Duration formats a duration compactly: 850ms, 12.3s, 3m05s, 1h02m.
func Duration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// Size formats bytes as B/KB/MB/GB (base 1024).
func Size(n int64) string {
	const k = 1024
	switch {
	case n < k:
		return fmt.Sprintf("%d B", n)
	case n < k*k:
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
	}
	return fmt.Sprintf("%.2f GB", float64(n)/(k*k*k))
}
