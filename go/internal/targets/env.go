package targets

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/Matkurban/fpack/go/internal/flutter"
)

// AndroidEnv describes the Android toolchain as Flutter would see it.
type AndroidEnv struct {
	SDK         string // Android SDK root
	SDKSource   string
	Java        string // java executable
	JavaHome    string
	JavaSource  string
	JavaVersion int // major version, 0 if unknown
	Licenses    bool
	BuildTools  string // newest build-tools dir
}

// DetectAndroidEnv mirrors Flutter's lookup order: flutter config values,
// ANDROID_HOME/ANDROID_SDK_ROOT, default install locations; for Java:
// flutter config jdk-dir, Android Studio's bundled JBR, JAVA_HOME, PATH.
func DetectAndroidEnv(t Tools) *AndroidEnv {
	e := &AndroidEnv{}
	settings := flutter.Settings()
	home, _ := os.UserHomeDir()
	sdkCands := [][2]string{
		{settings["android-sdk"], "flutter config --android-sdk"},
		{os.Getenv("ANDROID_HOME"), "ANDROID_HOME"},
		{os.Getenv("ANDROID_SDK_ROOT"), "ANDROID_SDK_ROOT"},
	}
	switch runtime.GOOS {
	case "darwin":
		sdkCands = append(sdkCands, [2]string{filepath.Join(home, "Library", "Android", "sdk"), "~/Library/Android/sdk"})
	case "windows":
		sdkCands = append(sdkCands, [2]string{filepath.Join(os.Getenv("LOCALAPPDATA"), "Android", "Sdk"), `%LOCALAPPDATA%\Android\Sdk`})
	default:
		sdkCands = append(sdkCands, [2]string{filepath.Join(home, "Android", "Sdk"), "~/Android/Sdk"})
	}
	for _, c := range sdkCands {
		if c[0] != "" && exists(filepath.Join(c[0], "platform-tools")) || c[0] != "" && exists(filepath.Join(c[0], "build-tools")) {
			e.SDK, e.SDKSource = c[0], c[1]
			break
		}
	}
	if e.SDK != "" {
		e.Licenses = exists(filepath.Join(e.SDK, "licenses", "android-sdk-license"))
		e.BuildTools = newestDir(filepath.Join(e.SDK, "build-tools"))
	}

	javaCands := [][2]string{{settings["jdk-dir"], "flutter config --jdk-dir"}}
	switch runtime.GOOS {
	case "darwin":
		javaCands = append(javaCands,
			[2]string{"/Applications/Android Studio.app/Contents/jbr/Contents/Home", "Android Studio JBR"},
			[2]string{filepath.Join(home, "Applications", "Android Studio.app", "Contents", "jbr", "Contents", "Home"), "Android Studio JBR"})
	case "windows":
		javaCands = append(javaCands, [2]string{`C:\Program Files\Android\Android Studio\jbr`, "Android Studio JBR"})
	default:
		javaCands = append(javaCands, [2]string{"/opt/android-studio/jbr", "Android Studio JBR"}, [2]string{filepath.Join(home, "android-studio", "jbr"), "Android Studio JBR"})
	}
	javaCands = append(javaCands, [2]string{os.Getenv("JAVA_HOME"), "JAVA_HOME"})
	for _, c := range javaCands {
		if c[0] == "" {
			continue
		}
		j := filepath.Join(c[0], "bin", exe("java"))
		if exists(j) {
			e.Java, e.JavaHome, e.JavaSource = j, c[0], c[1]
			break
		}
	}
	if e.Java == "" {
		if j := t.Find("java"); j != "" {
			e.Java, e.JavaSource = j, "PATH"
			if real, err := filepath.EvalSymlinks(j); err == nil {
				e.JavaHome = filepath.Dir(filepath.Dir(real))
			}
		}
	}
	if e.Java != "" {
		if out, ok := t.Probe(e.Java, "-version"); ok || out != "" {
			e.JavaVersion = ParseJavaMajor(out)
		}
	}
	return e
}

var javaVer = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?`)

// ParseJavaMajor extracts the major version from `java -version` output.
func ParseJavaMajor(out string) int {
	m := javaVer.FindStringSubmatch(out)
	if m == nil {
		return 0
	}
	major, _ := strconv.Atoi(m[1])
	if major == 1 && m[2] != "" { // 1.8 -> 8
		major, _ = strconv.Atoi(m[2])
	}
	return major
}

// Apksigner returns the apksigner path from the newest build-tools.
func (e *AndroidEnv) Apksigner() string {
	if e.BuildTools == "" {
		return ""
	}
	name := "apksigner"
	if runtime.GOOS == "windows" {
		name = "apksigner.bat"
	}
	p := filepath.Join(e.BuildTools, name)
	if exists(p) {
		return p
	}
	return ""
}

// Keytool returns the keytool next to the detected java.
func (e *AndroidEnv) Keytool(t Tools) string {
	if e.Java != "" {
		p := filepath.Join(filepath.Dir(e.Java), exe("keytool"))
		if exists(p) {
			return p
		}
	}
	return t.Find("keytool")
}

func newestDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Slice(names, func(i, j int) bool { return versionLess(names[i], names[j]) })
	return filepath.Join(dir, names[len(names)-1])
}

func versionLess(a, b string) bool {
	pa, pb := strings.FieldsFunc(a, func(r rune) bool { return r == '.' || r == '-' }), strings.FieldsFunc(b, func(r rune) bool { return r == '.' || r == '-' })
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, ex := strconv.Atoi(pa[i])
		y, ey := strconv.Atoi(pb[i])
		if ex == nil && ey == nil {
			if x != y {
				return x < y
			}
			continue
		}
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return len(pa) < len(pb)
}

func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// knownLocations lists install paths of tools that are often not in PATH.
func knownLocations(name string) []string {
	home, _ := os.UserHomeDir()
	switch strings.ToLower(name) {
	case "iscc", "iscc.exe":
		var out []string
		for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs")} {
			if base != "" {
				out = append(out, filepath.Join(base, "Inno Setup 6", "ISCC.exe"))
			}
		}
		return out
	case "appimagetool":
		return []string{filepath.Join(home, ".local", "bin", "appimagetool"), filepath.Join(home, "Applications", "appimagetool-x86_64.AppImage"), "/usr/local/bin/appimagetool-x86_64.AppImage", filepath.Join(home, ".local", "bin", "appimagetool-x86_64.AppImage")}
	case "create-dmg":
		return []string{"/opt/homebrew/bin/create-dmg", "/usr/local/bin/create-dmg"}
	case "pod":
		return []string{"/opt/homebrew/bin/pod", "/usr/local/bin/pod", filepath.Join(home, ".gem", "bin", "pod")}
	case "makeappx", "makeappx.exe":
		return nil
	case "signtool", "signtool.exe":
		var out []string
		for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles")} {
			if base == "" {
				continue
			}
			m, _ := filepath.Glob(filepath.Join(base, "Windows Kits", "10", "bin", "10.*", "x64", "signtool.exe"))
			sort.Sort(sort.Reverse(sort.StringSlice(m)))
			out = append(out, m...)
			out = append(out, filepath.Join(base, "Windows Kits", "10", "App Certification Kit", "signtool.exe"))
		}
		return out
	}
	return nil
}
