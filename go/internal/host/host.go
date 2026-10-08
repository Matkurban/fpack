// Package host describes the machine fpack runs on and which Flutter
// platforms it can build. Flutter does not cross-compile desktop apps:
// Windows needs Windows, Linux needs Linux, iOS/macOS need macOS.
package host

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

// Host is the current machine.
type Host struct {
	OS     string // darwin, linux, windows
	Arch   string // amd64, arm64
	Distro string // linux only: ID from /etc/os-release (debian, ubuntu, fedora, arch...)
	Like   []string
}

// Current detects the running host.
func Current() Host {
	h := Host{OS: runtime.GOOS, Arch: runtime.GOARCH}
	if h.OS == "linux" {
		h.Distro, h.Like = readOSRelease("/etc/os-release")
	}
	return h
}

func readOSRelease(path string) (string, []string) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer f.Close()
	var id string
	var like []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			id = v
		case "ID_LIKE":
			like = strings.Fields(v)
		}
	}
	return id, like
}

// OSName returns a human friendly OS name.
func (h Host) OSName() string { return OSName(h.OS) }

// OSName maps a GOOS value to a display name.
func OSName(goos string) string {
	switch goos {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	}
	return goos
}

// FlutterArch is the architecture directory Flutter uses for desktop builds
// (build/linux/<arch>, build/windows/<arch>).
func (h Host) FlutterArch() string {
	if h.Arch == "arm64" {
		return "arm64"
	}
	return "x64"
}

// IsDebianLike reports apt based distributions.
func (h Host) IsDebianLike() bool { return h.is("debian", "ubuntu") }

// IsFedoraLike reports dnf/yum based distributions.
func (h Host) IsFedoraLike() bool { return h.is("fedora", "rhel", "centos") }

// IsArchLike reports pacman based distributions.
func (h Host) IsArchLike() bool { return h.is("arch") }

func (h Host) is(ids ...string) bool {
	for _, id := range ids {
		if h.Distro == id {
			return true
		}
		for _, l := range h.Like {
			if l == id {
				return true
			}
		}
	}
	return false
}

// Install returns a package-manager command that installs the given package
// names for this host; pkgs is keyed by manager: brew, apt, dnf, pacman, winget.
func (h Host) Install(pkgs map[string]string) string {
	switch h.OS {
	case "darwin":
		if p := pkgs["brew"]; p != "" {
			return "brew install " + p
		}
	case "windows":
		if p := pkgs["winget"]; p != "" {
			return "winget install --id " + p + " -e"
		}
	case "linux":
		switch {
		case h.IsDebianLike() && pkgs["apt"] != "":
			return "sudo apt-get install -y " + pkgs["apt"]
		case h.IsFedoraLike() && pkgs["dnf"] != "":
			return "sudo dnf install -y " + pkgs["dnf"]
		case h.IsArchLike() && pkgs["pacman"] != "":
			return "sudo pacman -S --needed " + pkgs["pacman"]
		case pkgs["apt"] != "":
			return "sudo apt-get install -y " + pkgs["apt"] + "   # (Debian/Ubuntu; use your distro's package manager otherwise)"
		}
	}
	return ""
}
