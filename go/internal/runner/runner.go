// Package runner executes external tools (flutter, xcrun, hdiutil, dpkg-deb…).
//
// Every command's combined output is written to a log file, the last lines
// are kept in memory for error reports, secrets are redacted everywhere, and
// cancellation (Ctrl-C) stops the whole child process tree.
package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Cmd describes one external command.
type Cmd struct {
	Name string   // executable (absolute path or name in PATH)
	Args []string // arguments
	Dir  string   // working directory
	// Env holds extra environment variables (KEY=VALUE) added to os.Environ().
	Env []string
	// Secret lists values that must never be printed (passwords, tokens).
	// Any env var whose value is a secret is shown as KEY=***.
	Secret []string
	// Capture keeps the full output in Result.Output (for small outputs that
	// need parsing, e.g. notarytool JSON).
	Capture bool
}

// String renders the command as a copy-pasteable shell line with secrets
// redacted. Only Env entries are shown, not the inherited environment.
func (c Cmd) String() string {
	var parts []string
	for _, e := range c.Env {
		k, v, _ := strings.Cut(e, "=")
		parts = append(parts, k+"="+quoteRedacted(c.redact(v)))
	}
	parts = append(parts, Quote(displayName(c.Name)))
	for _, a := range c.Args {
		parts = append(parts, quoteRedacted(c.redact(a)))
	}
	return strings.Join(parts, " ")
}

// quoteRedacted quotes s but keeps the *** redaction marker readable.
func quoteRedacted(s string) string {
	return strings.ReplaceAll(Quote(strings.ReplaceAll(s, "***", "\x00")), "\x00", "***")
}

func displayName(n string) string {
	// Show "flutter" instead of a long absolute path when it is the SDK tool.
	base := filepath.Base(n)
	switch strings.TrimSuffix(strings.TrimSuffix(base, ".bat"), ".exe") {
	case "flutter", "dart":
		return n
	}
	return n
}

func (c Cmd) redact(s string) string { return Redact(s, c.Secret) }

// Redact replaces every secret occurrence in s with ***.
func Redact(s string, secrets []string) string {
	// Longest first so overlapping secrets are fully hidden.
	sec := append([]string(nil), secrets...)
	sort.Slice(sec, func(i, j int) bool { return len(sec[i]) > len(sec[j]) })
	for _, v := range sec {
		if len(v) >= 3 {
			s = strings.ReplaceAll(s, v, "***")
		} else if v != "" && s == v {
			s = "***"
		}
	}
	return s
}

// Quote quotes s for display in a POSIX shell when needed.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`!*?[](){}<>|&;#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Result of a command.
type Result struct {
	ExitCode int
	Duration time.Duration
	LogPath  string
	Tail     []string // last output lines (redacted)
	Output   string   // full output when Cmd.Capture
}

// ErrInterrupted is returned when the context was cancelled (Ctrl-C).
var ErrInterrupted = errors.New("interrupted")

// ExitError is returned for a non-zero exit status.
type ExitError struct {
	Cmd  Cmd
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("%s exited with code %d", filepath.Base(e.Cmd.Name), e.Code)
}

// Options control a Run.
type Options struct {
	// LogPath is the file that receives the full output (appended).
	LogPath string
	// OnLine is called for every output line (already redacted).
	OnLine func(string)
	// Stream writes every line to this writer as well (--verbose).
	Stream io.Writer
	// TailLines is the number of lines kept for error reports (default 400).
	TailLines int
}

// Run executes cmd and waits. A cancelled ctx stops the process tree
// gracefully (SIGINT, then SIGKILL after a grace period).
func Run(ctx context.Context, c Cmd, o Options) (Result, error) {
	start := time.Now()
	res := Result{LogPath: o.LogPath}
	if o.TailLines == 0 {
		o.TailLines = 400
	}
	var logf *os.File
	if o.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(o.LogPath), 0o755); err == nil {
			logf, _ = os.OpenFile(o.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		}
	}
	if logf != nil {
		defer logf.Close()
		fmt.Fprintf(logf, "$ %s\n# cwd: %s\n# started: %s\n", c.String(), c.Dir, start.Format(time.RFC3339))
	}

	cmd := exec.Command(c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), c.Env...)
	cmd.Stdin = nil
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		pw.Close()
		res.ExitCode = 127
		res.Duration = time.Since(start)
		if logf != nil {
			fmt.Fprintf(logf, "# failed to start: %v\n", err)
		}
		return res, fmt.Errorf("cannot start %s: %w", c.Name, err)
	}

	var (
		mu   sync.Mutex
		tail []string
		full strings.Builder
		wg   sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
		sc.Split(scanLinesCR)
		for sc.Scan() {
			line := Redact(strings.TrimRight(sc.Text(), "\r"), c.Secret)
			if logf != nil {
				fmt.Fprintln(logf, line)
			}
			if o.Stream != nil {
				fmt.Fprintln(o.Stream, line)
			}
			if o.OnLine != nil && strings.TrimSpace(line) != "" {
				o.OnLine(line)
			}
			mu.Lock()
			tail = append(tail, line)
			if len(tail) > o.TailLines {
				tail = tail[len(tail)-o.TailLines:]
			}
			if c.Capture {
				full.WriteString(line)
				full.WriteByte('\n')
			}
			mu.Unlock()
		}
		io.Copy(io.Discard, pr)
	}()

	track(cmd)
	defer untrack(cmd)

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	var waitErr error
	interrupted := false
	select {
	case waitErr = <-waitCh:
	case <-ctx.Done():
		interrupted = true
		interruptTree(cmd)
		select {
		case waitErr = <-waitCh:
		case <-time.After(8 * time.Second):
			killTree(cmd)
			waitErr = <-waitCh
		}
	}
	pw.Close()
	wg.Wait()

	res.Duration = time.Since(start)
	res.Tail = tail
	res.Output = full.String()
	res.ExitCode = exitCode(cmd, waitErr)
	if logf != nil {
		fmt.Fprintf(logf, "# exit code: %d (%s)\n\n", res.ExitCode, res.Duration.Round(time.Millisecond))
	}
	if interrupted {
		return res, ErrInterrupted
	}
	if res.ExitCode != 0 {
		return res, &ExitError{Cmd: c, Code: res.ExitCode}
	}
	return res, nil
}

func exitCode(cmd *exec.Cmd, err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if c := ee.ExitCode(); c >= 0 {
			return c
		}
		return 1
	}
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return 1
}

// scanLinesCR splits on \n, and also on lone \r used by progress bars.
func scanLinesCR(data []byte, atEOF bool) (int, []byte, error) {
	for i, b := range data {
		if b == '\n' {
			return i + 1, data[:i], nil
		}
		if b == '\r' {
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + 2, data[:i], nil
				}
				return i + 1, data[:i], nil
			}
			if atEOF {
				return i + 1, data[:i], nil
			}
			return 0, nil, nil // need more data to decide
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// Which finds an executable in PATH (adding .exe/.bat/.cmd on Windows).
func Which(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	if runtime.GOOS == "windows" {
		for _, ext := range []string{".exe", ".bat", ".cmd"} {
			if p, err := exec.LookPath(name + ext); err == nil {
				return p
			}
		}
	}
	return ""
}

// Output runs a short command and returns its trimmed combined output. It
// never fails loudly; ok is false on error or timeout.
func Output(timeout time.Duration, name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err == nil
}

var (
	activeMu sync.Mutex
	active   = map[*exec.Cmd]struct{}{}
)

func track(c *exec.Cmd)   { activeMu.Lock(); active[c] = struct{}{}; activeMu.Unlock() }
func untrack(c *exec.Cmd) { activeMu.Lock(); delete(active, c); activeMu.Unlock() }

// KillAll force-kills every running child process tree (second Ctrl-C).
func KillAll() {
	activeMu.Lock()
	defer activeMu.Unlock()
	for c := range active {
		killTree(c)
	}
}
