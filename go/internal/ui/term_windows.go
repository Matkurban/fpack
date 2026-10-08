//go:build windows

package ui

import (
	"io"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

var (
	k32                = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = k32.NewProc("GetConsoleMode")
	procSetConsoleMode = k32.NewProc("SetConsoleMode")
	procGetInfo        = k32.NewProc("GetConsoleScreenBufferInfo")
)

func isTerminal(f *os.File) bool {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(f.Fd(), uintptr(unsafe.Pointer(&mode)))
	return r != 0
}

// enableVT turns on ANSI escape processing (Windows 10+).
func enableVT(f *os.File) {
	var mode uint32
	if r, _, _ := procGetConsoleMode.Call(f.Fd(), uintptr(unsafe.Pointer(&mode))); r != 0 {
		procSetConsoleMode.Call(f.Fd(), uintptr(mode|0x0004))
	}
}

type coord struct{ X, Y int16 }
type smallRect struct{ L, T, R, B int16 }
type csbi struct {
	Size, Cursor coord
	Attr         uint16
	Window       smallRect
	Max          coord
}

func termWidth(w io.Writer) int {
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 20 {
		return c
	}
	if f, ok := w.(*os.File); ok {
		var info csbi
		if r, _, _ := procGetInfo.Call(f.Fd(), uintptr(unsafe.Pointer(&info))); r != 0 {
			if c := int(info.Window.R-info.Window.L) + 1; c > 20 {
				return c
			}
		}
	}
	return 100
}
