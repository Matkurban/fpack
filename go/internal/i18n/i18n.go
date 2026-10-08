// Package i18n provides tiny bilingual (English / Simplified Chinese) text
// helpers. Strings live next to the code that prints them:
//
//	ui.Info(i18n.S("Building", "正在构建"))
//	i18n.F("%d targets", "%d 个目标", n)
package i18n

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Lang is an output language.
type Lang string

// Supported languages.
const (
	EN Lang = "en"
	ZH Lang = "zh"
)

var current = EN

// Set sets the active language.
func Set(l Lang) { current = l }

// Current returns the active language.
func Current() Lang { return current }

// IsZH reports whether Chinese output is active.
func IsZH() bool { return current == ZH }

// S picks the string for the active language.
func S(en, zh string) string {
	if current == ZH && zh != "" {
		return zh
	}
	return en
}

// F is S followed by fmt.Sprintf.
func F(en, zh string, args ...any) string { return fmt.Sprintf(S(en, zh), args...) }

// Parse normalizes a user supplied language value ("zh", "zh_CN.UTF-8",
// "en-US", "chinese"...). ok is false for unknown values.
func Parse(v string) (Lang, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	switch {
	case v == "":
		return EN, false
	case strings.HasPrefix(v, "zh"), v == "cn", v == "chinese", strings.HasPrefix(v, "中文"):
		return ZH, true
	case strings.HasPrefix(v, "en"), v == "english":
		return EN, true
	}
	return EN, false
}

// Detect resolves the language: explicit override, then FPACK_LANG, then the
// POSIX locale variables, then the OS preference (macOS AppleLanguages /
// AppleLocale, Windows user locale). Defaults to English.
func Detect(override string, getenv func(string) string) Lang {
	if l, ok := Parse(override); ok {
		return l
	}
	if l, ok := Parse(getenv("FPACK_LANG")); ok {
		return l
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := getenv(k)
		if v == "" || v == "C" || v == "POSIX" || strings.HasPrefix(v, "C.") {
			continue
		}
		if l, ok := Parse(v); ok {
			return l
		}
		return EN // a real, non-Chinese locale is set
	}
	if l, ok := Parse(osLocale()); ok {
		return l
	}
	return EN
}

func osLocale() string {
	switch runtime.GOOS {
	case "darwin":
		// AppleLanguages reflects the UI language order, AppleLocale the region format.
		if out := quick("defaults", "read", "-g", "AppleLanguages"); out != "" {
			for _, f := range strings.FieldsFunc(out, func(r rune) bool { return r == '(' || r == ')' || r == ',' || r == '"' || r == '\n' || r == ' ' }) {
				if f != "" {
					return f
				}
			}
		}
		return quick("defaults", "read", "-g", "AppleLocale")
	case "windows":
		return windowsLocale()
	}
	return ""
}

func quick(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	done := make(chan struct{})
	var out []byte
	go func() { out, _ = cmd.Output(); close(done) }()
	select {
	case <-done:
		return strings.TrimSpace(string(out))
	case <-time.After(1500 * time.Millisecond):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ""
	}
}

// Getenv is os.Getenv, exposed for callers that pass an environment lookup.
func Getenv(k string) string { return os.Getenv(k) }
