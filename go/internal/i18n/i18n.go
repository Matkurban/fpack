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
	"path/filepath"
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

// Detect resolves the language: explicit override (--lang), then FPACK_LANG,
// then the operating system's language. Chinese (any zh variant) gives
// Chinese, everything else English.
//
// The OS language is:
//   - macOS: the first entry of the system UI language list (System Settings >
//     General > Language & Region, `defaults read -g AppleLanguages`), then
//     AppleLocale. Terminal sets LANG from the region format (often
//     en_US.UTF-8 even with a Chinese UI), so LANG/LANGUAGE are only a
//     fallback; LC_ALL / LC_MESSAGES set explicitly still win.
//   - Windows: the user's display language (GetUserDefaultUILanguage), then
//     the user locale. Git Bash/MSYS set LANG themselves, so as on macOS only
//     LC_ALL / LC_MESSAGES override it.
//   - Linux and others: LC_ALL, LC_MESSAGES, LANGUAGE (first entry), LANG.
func Detect(override string, getenv func(string) string) Lang {
	return detect(override, getenv, runtime.GOOS, osLanguage)
}

// detect is Detect with the OS and its language lookup injectable for tests.
func detect(override string, getenv func(string) string, goos string, osLang func(string) string) Lang {
	if l, ok := Parse(override); ok {
		return l
	}
	if l, ok := Parse(getenv("FPACK_LANG")); ok {
		return l
	}
	explicit := []string{"LC_ALL", "LC_MESSAGES"}
	fallback := []string{"LANGUAGE", "LANG"}
	if goos != "darwin" && goos != "windows" {
		explicit, fallback = append(explicit, fallback...), nil
	}
	if l, ok := fromEnv(explicit, getenv); ok {
		return l
	}
	if fallback != nil {
		if v := osLang(goos); v != "" {
			return langOf(v)
		}
		if l, ok := fromEnv(fallback, getenv); ok {
			return l
		}
	}
	return EN
}

// fromEnv returns the language of the first set, meaningful locale variable.
func fromEnv(keys []string, getenv func(string) string) (Lang, bool) {
	for _, k := range keys {
		v := strings.TrimSpace(getenv(k))
		if k == "LANGUAGE" {
			v, _, _ = strings.Cut(v, ":")
		}
		if v == "" || v == "C" || v == "POSIX" || strings.HasPrefix(v, "C.") {
			continue
		}
		return langOf(v), true
	}
	return EN, false
}

// langOf maps any locale/language tag to ZH for Chinese, else EN.
func langOf(v string) Lang {
	if l, ok := Parse(v); ok && l == ZH {
		return ZH
	}
	return EN
}

// osLanguage returns the OS UI language tag ("zh-Hans-CN", "en-US"...) or "".
func osLanguage(goos string) string {
	switch goos {
	case "darwin":
		return macLanguage()
	case "windows":
		return windowsLocale()
	}
	return ""
}

// macLanguage reads AppleLanguages/AppleLocale with `defaults` (≈30 ms), cached
// in the user cache dir and invalidated when the global preferences change.
func macLanguage() string {
	home, _ := os.UserHomeDir()
	cacheDir, _ := os.UserCacheDir()
	var stamp, cache string
	if home != "" && cacheDir != "" {
		if fi, err := os.Stat(filepath.Join(home, "Library", "Preferences", ".GlobalPreferences.plist")); err == nil {
			stamp = fmt.Sprint(fi.ModTime().UnixMicro())
			cache = filepath.Join(cacheDir, "fpack", "os-language")
			if b, err := os.ReadFile(cache); err == nil {
				if s, v, ok := strings.Cut(strings.TrimSpace(string(b)), " "); ok && s == stamp {
					return v
				}
			}
		}
	}
	v := firstAppleLanguage(quick("defaults", "read", "-g", "AppleLanguages"))
	if v == "" {
		v = quick("defaults", "read", "-g", "AppleLocale")
	}
	if cache != "" && v != "" {
		_ = os.MkdirAll(filepath.Dir(cache), 0o755)
		_ = os.WriteFile(cache, []byte(stamp+" "+v+"\n"), 0o644)
	}
	return v
}

// firstAppleLanguage extracts the first entry of `defaults read -g AppleLanguages`
// output, e.g. `(\n    "zh-Hans-CN",\n    "en-CN"\n)`.
func firstAppleLanguage(out string) string {
	for _, f := range strings.FieldsFunc(out, func(r rune) bool {
		return r == '(' || r == ')' || r == ',' || r == '"' || r == '\n' || r == ' ' || r == '\t'
	}) {
		return f
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
