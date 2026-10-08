package ui

import (
	"fmt"
	"strings"
	"time"
)

// Spinner shows an animated status line with elapsed time and the latest
// output line of a running command. On non-interactive output it prints
// nothing; callers print start/finish lines themselves.
type Spinner struct {
	u      *UI
	label  string
	detail string
	start  time.Time
	frame  int
	stop   chan struct{}
	done   chan struct{}
}

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// StartSpinner starts a spinner (no-op when not interactive or verbose).
func (u *UI) StartSpinner(label string) *Spinner {
	s := &Spinner{u: u, label: label, start: time.Now(), stop: make(chan struct{}), done: make(chan struct{})}
	if !u.tty || u.Verbose {
		close(s.done)
		return s
	}
	u.mu.Lock()
	u.spin = s
	u.mu.Unlock()
	go s.loop()
	return s
}

func (s *Spinner) loop() {
	defer close(s.done)
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.u.mu.Lock()
			s.frame++
			s.u.redrawSpinLocked()
			s.u.mu.Unlock()
		}
	}
}

// Update sets the detail text (latest output line).
func (s *Spinner) Update(detail string) {
	s.u.mu.Lock()
	s.detail = detail
	s.u.mu.Unlock()
}

// Stop removes the spinner line.
func (s *Spinner) Stop() {
	select {
	case <-s.done:
		return
	default:
	}
	close(s.stop)
	<-s.done
	s.u.mu.Lock()
	s.u.clearSpinLocked()
	s.u.spin = nil
	s.u.mu.Unlock()
}

func (u *UI) clearSpinLocked() {
	if u.spin != nil {
		fmt.Fprint(u.w, "\r\x1b[2K")
	}
}

func (u *UI) redrawSpinLocked() {
	s := u.spin
	if s == nil {
		return
	}
	el := time.Since(s.start).Truncate(time.Second)
	head := fmt.Sprintf("%s %s %s", u.Cyan(frames[s.frame%len(frames)]), s.label, u.Dim(Duration(el)))
	width := termWidth(u.w)
	used := DisplayWidth(StripANSI(head)) + 3
	line := head
	if d := strings.TrimSpace(s.detail); d != "" && width-used > 10 {
		line += "  " + u.Dim(Truncate(d, width-used))
	}
	fmt.Fprint(u.w, "\r\x1b[2K"+line)
}
