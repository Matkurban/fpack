package host

// Platform is a Flutter target platform.
type Platform string

// Flutter platforms.
const (
	Android Platform = "android"
	IOS     Platform = "ios"
	MacOS   Platform = "macos"
	Windows Platform = "windows"
	Linux   Platform = "linux"
	Web     Platform = "web"
)

// AllPlatforms in display order.
var AllPlatforms = []Platform{Android, IOS, MacOS, Windows, Linux, Web}

// RequiredOS returns the GOOS a platform must be built on, or "" when any
// host works.
func RequiredOS(p Platform) string {
	switch p {
	case IOS, MacOS:
		return "darwin"
	case Windows:
		return "windows"
	case Linux:
		return "linux"
	}
	return ""
}

// CanBuild reports whether host h can build platform p. When it cannot, the
// returned string is the GOOS that is required.
func (h Host) CanBuild(p Platform) (bool, string) {
	req := RequiredOS(p)
	if req == "" || req == h.OS {
		return true, ""
	}
	return false, req
}
