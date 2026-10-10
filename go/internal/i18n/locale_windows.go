//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

// windowsLocale returns the user's display (UI) language, e.g. "zh-CN", falling
// back to the user locale. Plain kernel32 calls, no subprocess.
func windowsLocale() string {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	if p := k32.NewProc("GetUserDefaultUILanguage"); p.Find() == nil {
		if id, _, _ := p.Call(); id != 0 {
			if name := lcidName(k32, uint32(id)); name != "" {
				return name
			}
			if id&0x3ff == 0x04 { // LANG_CHINESE
				return "zh"
			}
			return "en"
		}
	}
	p := k32.NewProc("GetUserDefaultLocaleName")
	if p.Find() != nil {
		return ""
	}
	buf := make([]uint16, 85)
	if r, _, _ := p.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func lcidName(k32 *syscall.LazyDLL, lcid uint32) string {
	p := k32.NewProc("LCIDToLocaleName")
	if p.Find() != nil {
		return ""
	}
	buf := make([]uint16, 85)
	if r, _, _ := p.Call(uintptr(lcid), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
