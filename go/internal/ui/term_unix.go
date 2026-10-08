//go:build linux || darwin

package ui

import (
	"io"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

type winsize struct{ Row, Col, X, Y uint16 }

func isTerminal(f *os.File) bool {
	var ws winsize
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	return e == 0
}

func termWidth(w io.Writer) int {
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 20 {
		return c
	}
	if f, ok := w.(*os.File); ok {
		var ws winsize
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws))); e == 0 && ws.Col > 20 {
			return int(ws.Col)
		}
	}
	return 100
}

func enableVT(*os.File) {}
