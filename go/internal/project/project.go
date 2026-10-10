// Package project inspects a Flutter project (read-only): pubspec, enabled
// platforms, flavors, signing hints, product names. fpack never edits these
// files.
package project

import (
	"errors"
	"fmt"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Matkurban/fpack/go/internal/host"
	"go.yaml.in/yaml/v3"
)

// Project is a detected Flutter application.
type Project struct {
	Root        string
	Name        string
	Version     string // "1.0.0" (without build number)
	BuildNumber string // "1"
	Description string
	Pubspec     map[string]any

	Platforms map[host.Platform]bool
	Deps      map[string]Dep

	AndroidFlavors []string
	IOSSchemes     []string
	MacOSSchemes   []string

	AndroidGradleFile         string // relative path
	AndroidApplicationID      string
	AndroidReleaseDebugSigned bool // release buildType uses signingConfigs.debug
	AndroidKeyProperties      bool // android/key.properties exists
	IOSBundleID               string
	IOSTeam                   string
	MacProductName            string
	MacBundleID               string // PRODUCT_BUNDLE_IDENTIFIER in macos/Runner/Configs/AppInfo.xcconfig
	MacDeploymentTarget       string // MACOSX_DEPLOYMENT_TARGET of macos/Runner.xcodeproj
	LinuxBinary               string
	LinuxAppID                string
	WindowsBinary             string
	WindowsCompany            string   // CompanyName in windows/runner/Runner.rc (unless it is the reverse-DNS org flutter create writes)
	DefineFiles               []string // candidate --dart-define-from-file files (config/, env/, …)
	LauncherIcon              string   // image_path from flutter_launcher_icons (relative)
}

// Dep is a pubspec dependency.
type Dep struct {
	Name string
	Dev  bool
	Path string // path dependency (as written)
}

// ErrNotFound means no Flutter project contains the start directory.
type ErrNotFound struct {
	Start      string
	Candidates []string // Flutter apps found below Start (monorepos)
	NonFlutter string   // a pubspec.yaml that exists but is not a Flutter app
}

func (e *ErrNotFound) Error() string {
	return i18n.S("no Flutter project found at ", "未找到 Flutter 项目：") + e.Start
}

// Find locates the Flutter project containing dir (walking up), or reports
// Flutter apps found in subdirectories (monorepo roots).
func Find(dir string) (*Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return nil, fmt.Errorf(i18n.S("directory %s does not exist", "目录 %s 不存在"), dir)
	}
	nf := &ErrNotFound{Start: abs}
	for d := abs; ; {
		p := filepath.Join(d, "pubspec.yaml")
		if data, err := os.ReadFile(p); err == nil {
			if isFlutterApp(data) {
				return Load(d)
			}
			if nf.NonFlutter == "" {
				nf.NonFlutter = p
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	nf.Candidates = scanCandidates(abs, 3)
	return nil, nf
}

func isFlutterApp(pubspec []byte) bool {
	var m map[string]any
	if yaml.Unmarshal(pubspec, &m) != nil {
		return false
	}
	deps, _ := m["dependencies"].(map[string]any)
	f, ok := deps["flutter"].(map[string]any)
	return ok && f["sdk"] == "flutter"
}

func scanCandidates(root string, depth int) []string {
	var out []string
	var walk func(d string, n int)
	walk = func(d string, n int) {
		if n > depth {
			return
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || strings.HasPrefix(name, ".") || name == "build" || name == "node_modules" || name == "ios" || name == "android" {
				continue
			}
			sub := filepath.Join(d, name)
			if data, err := os.ReadFile(filepath.Join(sub, "pubspec.yaml")); err == nil && isFlutterApp(data) && hasAnyPlatform(sub) {
				rel, _ := filepath.Rel(root, sub)
				out = append(out, rel)
				continue
			}
			walk(sub, n+1)
		}
	}
	walk(root, 1)
	sort.Strings(out)
	return out
}

func hasAnyPlatform(dir string) bool {
	for _, p := range host.AllPlatforms {
		if st, err := os.Stat(filepath.Join(dir, string(p))); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

// Load reads the project at root.
func Load(root string) (*Project, error) {
	data, err := os.ReadFile(filepath.Join(root, "pubspec.yaml"))
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("pubspec.yaml: %v", err)
	}
	p := &Project{Root: root, Pubspec: m, Platforms: map[host.Platform]bool{}, Deps: map[string]Dep{}}
	p.Name, _ = m["name"].(string)
	if p.Name == "" {
		return nil, errors.New(i18n.S("pubspec.yaml has no name", "pubspec.yaml 中没有 name"))
	}
	p.Description, _ = m["description"].(string)
	p.Version, p.BuildNumber = SplitVersion(fmt.Sprint(valueOr(m["version"], "")))
	for _, sec := range []string{"dependencies", "dev_dependencies"} {
		deps, _ := m[sec].(map[string]any)
		for name, v := range deps {
			d := Dep{Name: name, Dev: sec == "dev_dependencies"}
			if mv, ok := v.(map[string]any); ok {
				if path, ok := mv["path"].(string); ok {
					d.Path = path
				}
			}
			if _, exists := p.Deps[name]; !exists || !d.Dev {
				p.Deps[name] = d
			}
		}
	}
	for _, pl := range host.AllPlatforms {
		if st, err := os.Stat(filepath.Join(root, string(pl))); err == nil && st.IsDir() {
			p.Platforms[pl] = true
		}
	}
	p.inspectAndroid()
	p.IOSSchemes = schemes(filepath.Join(root, "ios", "Runner.xcodeproj"))
	p.MacOSSchemes = schemes(filepath.Join(root, "macos", "Runner.xcodeproj"))
	if pbx := readFile(filepath.Join(root, "ios", "Runner.xcodeproj", "project.pbxproj")); pbx != "" {
		p.IOSTeam = firstMatch(pbx, `DEVELOPMENT_TEAM = "?([A-Z0-9]+)"?;`)
		p.IOSBundleID = firstMatch(pbx, `PRODUCT_BUNDLE_IDENTIFIER = "?([A-Za-z0-9.\-]+)"?;`)
	}
	appInfo := readFile(filepath.Join(root, "macos", "Runner", "Configs", "AppInfo.xcconfig"))
	if pbx := readFile(filepath.Join(root, "macos", "Runner.xcodeproj", "project.pbxproj")); pbx != "" {
		p.MacDeploymentTarget = firstMatch(pbx, `MACOSX_DEPLOYMENT_TARGET = "?([0-9.]+)"?;`)
	}
	p.MacProductName = firstMatch(appInfo, `(?m)^\s*PRODUCT_NAME\s*=\s*(.+?)\s*$`)
	p.MacBundleID = firstMatch(appInfo, `(?m)^\s*PRODUCT_BUNDLE_IDENTIFIER\s*=\s*([A-Za-z0-9.\-]+)\s*$`)
	linuxCM := readFile(filepath.Join(root, "linux", "CMakeLists.txt"))
	p.LinuxBinary = firstMatch(linuxCM, `set\(BINARY_NAME\s+"([^"]+)"\)`)
	p.LinuxAppID = firstMatch(linuxCM, `set\(APPLICATION_ID\s+"([^"]+)"\)`)
	p.WindowsBinary = firstMatch(readFile(filepath.Join(root, "windows", "CMakeLists.txt")), `set\(BINARY_NAME\s+"([^"]+)"\)`)
	if c := firstMatch(readFile(filepath.Join(root, "windows", "runner", "Runner.rc")), `VALUE "CompanyName", "([^"]*)"`); c != "" && !orgLike.MatchString(c) {
		p.WindowsCompany = c
	}
	p.DefineFiles = defineFiles(root)
	p.LauncherIcon = p.launcherIcon()
	return p, nil
}

// orgLike matches the reverse-DNS org flutter create puts into CompanyName
// (e.g. "com.example").
var orgLike = regexp.MustCompile(`^[a-z0-9_]+(\.[A-Za-z0-9_\-]+)+$`)

// defineFiles lists likely --dart-define-from-file files.
func defineFiles(root string) []string {
	var out []string
	for _, dir := range []string{"config", "configs", "env", "envs", "environments", "dart_defines", "defines"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() && (strings.HasSuffix(n, ".json") || strings.HasSuffix(n, ".env")) {
				out = append(out, dir+"/"+n)
			}
		}
	}
	return out
}

func valueOr(v any, d any) any {
	if v == nil {
		return d
	}
	return v
}

// SplitVersion splits "1.2.3+45" into "1.2.3" and "45". Flutter's defaults
// (1.0.0 / 1) apply when missing.
func SplitVersion(v string) (string, string) {
	v = strings.TrimSpace(v)
	if v == "" || v == "<nil>" {
		return "1.0.0", "1"
	}
	name, build, ok := strings.Cut(v, "+")
	if !ok || build == "" {
		return name, "1"
	}
	return name, build
}

func (p *Project) inspectAndroid() {
	for _, f := range []string{"android/app/build.gradle.kts", "android/app/build.gradle"} {
		txt := readFile(filepath.Join(p.Root, f))
		if txt == "" {
			continue
		}
		p.AndroidGradleFile = f
		p.AndroidFlavors = AndroidFlavors(txt)
		p.AndroidApplicationID = firstMatch(txt, `applicationId\s*=?\s*["']([^"']+)["']`)
		p.AndroidReleaseDebugSigned = ReleaseUsesDebugSigning(txt)
		break
	}
	if _, err := os.Stat(filepath.Join(p.Root, "android", "key.properties")); err == nil {
		p.AndroidKeyProperties = true
	}
}

var reDebugSigning = regexp.MustCompile(`signingConfig\s*=?\s*signingConfigs\.(getByName\(\s*"debug"\s*\)|debug\b)`)

// ReleaseUsesDebugSigning reports whether the release buildType in a Gradle
// file is signed with the debug key (Flutter's template default).
func ReleaseUsesDebugSigning(gradle string) bool {
	block := blockAfter(gradle, "buildTypes")
	rel := blockAfter(block, "release")
	if rel == "" {
		rel = blockAfter(block, `getByName("release")`)
	}
	return reDebugSigning.MatchString(rel)
}

var (
	reKtsFlavor = regexp.MustCompile(`(?:create|register|maybeCreate)\(\s*"([A-Za-z0-9_]+)"\s*\)`)
	reGroovyID  = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*$`)
)

// AndroidFlavors extracts productFlavors names from a Gradle (Groovy or
// Kotlin DSL) build file.
func AndroidFlavors(gradle string) []string {
	block := blockAfter(gradle, "productFlavors")
	if block == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	depth := 0
	start := 0
	for i, ch := range block {
		switch ch {
		case '{':
			if depth == 0 {
				head := strings.TrimSpace(block[start:i])
				name := ""
				if m := reKtsFlavor.FindStringSubmatch(head); m != nil {
					name = m[1]
				} else if m := reGroovyID.FindStringSubmatch(head); m != nil {
					name = m[1]
				}
				if name != "" && !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
			depth++
		case '}':
			depth--
			if depth == 0 {
				start = i + 1
			}
		case '\n':
			if depth == 0 {
				start = i + 1
			}
		}
	}
	return out
}

// blockAfter returns the contents of the {...} block following keyword.
func blockAfter(s, keyword string) string {
	idx := 0
	for {
		i := strings.Index(s[idx:], keyword)
		if i < 0 {
			return ""
		}
		i += idx
		rest := s[i+len(keyword):]
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t' || rest[j] == '\n' || rest[j] == '\r') {
			j++
		}
		if j < len(rest) && rest[j] == '{' {
			depth := 0
			for k := j; k < len(rest); k++ {
				switch rest[k] {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						return rest[j+1 : k]
					}
				}
			}
			return rest[j+1:]
		}
		idx = i + len(keyword)
	}
}

// schemes lists the custom (non-Runner) Xcode schemes like `xcodebuild
// -list` does: shared and per-user schemes of the project and workspace.
func schemes(xcodeproj string) []string {
	ws := strings.TrimSuffix(xcodeproj, ".xcodeproj") + ".xcworkspace"
	dirs := []string{filepath.Join(xcodeproj, "xcshareddata", "xcschemes"), filepath.Join(ws, "xcshareddata", "xcschemes")}
	for _, base := range []string{xcodeproj, ws} {
		users, _ := filepath.Glob(filepath.Join(base, "xcuserdata", "*.xcuserdatad", "xcschemes"))
		dirs = append(dirs, users...)
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := strings.TrimSuffix(e.Name(), ".xcscheme")
			if n != e.Name() && n != "Runner" && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (p *Project) launcherIcon() string {
	var cfg map[string]any
	if data, err := os.ReadFile(filepath.Join(p.Root, "flutter_launcher_icons.yaml")); err == nil {
		var m map[string]any
		if yaml.Unmarshal(data, &m) == nil {
			cfg, _ = m["flutter_launcher_icons"].(map[string]any)
			if cfg == nil {
				cfg = m
			}
		}
	}
	if cfg == nil {
		for _, k := range []string{"flutter_launcher_icons", "flutter_icons"} {
			if c, ok := p.Pubspec[k].(map[string]any); ok {
				cfg = c
				break
			}
		}
	}
	for _, k := range []string{"image_path", "image_path_android", "image_path_ios"} {
		if v, ok := cfg[k].(string); ok && v != "" {
			if _, err := os.Stat(filepath.Join(p.Root, v)); err == nil {
				return v
			}
		}
	}
	return ""
}

// HasDep reports whether pubspec lists a (dev) dependency.
func (p *Project) HasDep(name string) bool { _, ok := p.Deps[name]; return ok }

// MissingPathDeps returns path dependencies whose directory does not exist.
func (p *Project) MissingPathDeps() []Dep {
	var out []Dep
	for _, d := range p.Deps {
		if d.Path == "" {
			continue
		}
		path := d.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(p.Root, path)
		}
		if _, err := os.Stat(filepath.Join(path, "pubspec.yaml")); err != nil {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Section returns a top-level pubspec map (e.g. "msix_config").
func (p *Project) Section(name string) map[string]any {
	m, _ := p.Pubspec[name].(map[string]any)
	return m
}

// Identifier returns the best reverse-DNS app id.
func (p *Project) Identifier() string {
	for _, v := range []string{p.LinuxAppID, p.AndroidApplicationID, p.IOSBundleID} {
		if v != "" && !strings.Contains(v, "$") {
			return v
		}
	}
	return "com.example." + p.Name
}

// Abs resolves a project-relative path.
func (p *Project) Abs(rel string) string {
	if rel == "" || filepath.IsAbs(rel) {
		return rel
	}
	if strings.HasPrefix(rel, "~/") || strings.HasPrefix(rel, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rel[2:])
		}
	}
	return filepath.Join(p.Root, rel)
}

// Flavors returns flavors detected for a platform.
func (p *Project) Flavors(pl host.Platform) []string {
	switch pl {
	case host.Android:
		return p.AndroidFlavors
	case host.IOS:
		return p.IOSSchemes
	case host.MacOS:
		return p.MacOSSchemes
	}
	return nil
}

func readFile(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

func firstMatch(s, pattern string) string {
	if s == "" {
		return ""
	}
	m := regexp.MustCompile(pattern).FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(m[1]), `"`)
}
