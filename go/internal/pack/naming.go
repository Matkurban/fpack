// Package pack names artifacts, writes archives and checksums.
package pack

import (
	"fmt"
	"regexp"
	"strings"
)

// Fields are the values available to name templates.
type Fields struct {
	App      string // artifact base name
	Version  string // 1.0.0
	Build    string // build number
	Platform string // android, ios, macos, windows, linux, web
	Arch     string // universal, arm64-v8a, x64, arm64...
	Variant  string // setup, portable, unsigned...
	Mode     string // release/profile/debug
	Flavor   string
}

var placeholder = regexp.MustCompile(`\{([-_.+]?)([a-z]+)\}`)

// Placeholders lists valid template keys.
var Placeholders = []string{"app", "version", "build", "platform", "arch", "variant", "mode", "flavor"}

func (f Fields) value(key string) (string, bool) {
	switch key {
	case "app":
		return f.App, true
	case "version":
		return f.Version, true
	case "build":
		return f.Build, true
	case "platform":
		return f.Platform, true
	case "arch":
		return f.Arch, true
	case "variant":
		return f.Variant, true
	case "mode":
		// release is the default and is omitted from names.
		if f.Mode == "release" {
			return "", true
		}
		return f.Mode, true
	case "flavor":
		return f.Flavor, true
	}
	return "", false
}

// Render expands a template. {key} inserts the value; {-key}, {_key},
// {.key}, {+key} insert the separator followed by the value only when the
// value is not empty. Unsafe filename characters are replaced.
func Render(tmpl string, f Fields) (string, error) {
	var bad []string
	out := placeholder.ReplaceAllStringFunc(tmpl, func(m string) string {
		sm := placeholder.FindStringSubmatch(m)
		v, ok := f.value(sm[2])
		if !ok {
			bad = append(bad, m)
			return m
		}
		if v == "" {
			return ""
		}
		return sm[1] + v
	})
	if len(bad) > 0 {
		return "", fmt.Errorf("unknown placeholder %s (available: %s)", strings.Join(bad, ", "), strings.Join(Placeholders, ", "))
	}
	return SanitizeName(out), nil
}

// SanitizeName replaces characters that are invalid or awkward in file names.
func SanitizeName(s string) string {
	r := strings.NewReplacer("/", "-", `\`, "-", ":", "-", "*", "-", "?", "-", `"`, "", "<", "", ">", "", "|", "-", " ", "_")
	s = r.Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-_.")
}

// RenderDir expands the output directory template (same placeholders, no
// sanitizing of path separators).
func RenderDir(tmpl string, f Fields) (string, error) {
	var bad []string
	out := placeholder.ReplaceAllStringFunc(tmpl, func(m string) string {
		sm := placeholder.FindStringSubmatch(m)
		v, ok := f.value(sm[2])
		if !ok {
			bad = append(bad, m)
			return m
		}
		if v == "" {
			return ""
		}
		return sm[1] + v
	})
	if len(bad) > 0 {
		return "", fmt.Errorf("output.dir: unknown placeholder %s", strings.Join(bad, ", "))
	}
	return out, nil
}
