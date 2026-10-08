//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

func windowsLocale() string {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	p := k32.NewProc("GetUserDefaultLocaleName")
	buf := make([]uint16, 85)
	r, _, _ := p.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
