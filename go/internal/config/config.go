// Package config loads fpack.yaml and resolves effective settings with the
// precedence: command-line flags > environment variables > fpack.yaml >
// built-in defaults. Every field is optional; zero config works.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileNames are the config file names looked up in the project root.
var FileNames = []string{"fpack.yaml", "fpack.yml"}

// Config is the schema of fpack.yaml.
type Config struct {
	App     App     `yaml:"app"`
	Flutter Flutter `yaml:"flutter"`
	Build   Build   `yaml:"build"`
	Output  Output  `yaml:"output"`
	Android Android `yaml:"android"`
	IOS     IOS     `yaml:"ios"`
	MacOS   MacOS   `yaml:"macos"`
	Windows Windows `yaml:"windows"`
	Linux   Linux   `yaml:"linux"`
	Web     Web     `yaml:"web"`

	// File is the path the config was loaded from ("" when none).
	File string `yaml:"-"`
	// UnsetEnv lists ${VARS} referenced in the file that are not set.
	UnsetEnv []string `yaml:"-"`
}

// App describes the application for installers and package metadata.
type App struct {
	Name        string `yaml:"name"`         // artifact base name (default: pubspec name)
	DisplayName string `yaml:"display_name"` // human name for installers/menus
	Description string `yaml:"description"`
	Publisher   string `yaml:"publisher"`  // Windows installer publisher, deb/rpm vendor
	Identifier  string `yaml:"identifier"` // reverse-DNS id (com.example.app)
	Homepage    string `yaml:"homepage"`
	Maintainer  string `yaml:"maintainer"` // "Name <email>" for .deb
}

// Flutter selects the SDK.
type Flutter struct {
	SDK string `yaml:"sdk"` // path to the Flutter SDK root (default: auto)
}

// Build holds options shared by every flutter build.
type Build struct {
	Targets            []string `yaml:"targets"` // default targets for `fpack build`
	Mode               string   `yaml:"mode"`    // release | profile | debug
	Flavor             string   `yaml:"flavor"`
	Target             string   `yaml:"target"` // entry point, e.g. lib/main_prod.dart
	DartDefine         Defines  `yaml:"dart_define"`
	DartDefineFromFile List     `yaml:"dart_define_from_file"`
	BuildName          Scalar   `yaml:"build_name"`
	BuildNumber        Scalar   `yaml:"build_number"`
	Obfuscate          *bool    `yaml:"obfuscate"`
	SplitDebugInfo     string   `yaml:"split_debug_info"`
	ExtraArgs          List     `yaml:"extra_args"` // appended to every flutter build
}

// Output controls where artifacts go and how they are named.
type Output struct {
	Dir       string `yaml:"dir"`       // default: dist/{version}+{build}
	Name      string `yaml:"name"`      // file name template (without extension)
	Overwrite *bool  `yaml:"overwrite"` // replace existing artifacts (default false)
	Checksums *bool  `yaml:"checksums"` // write SHA256SUMS (default true)
}

// Android options.
type Android struct {
	SplitPerABI ABIMode        `yaml:"split_per_abi"` // false | true | both
	ABIs        List           `yaml:"abis"`          // arm64-v8a, armeabi-v7a, x86_64
	Signing     AndroidSigning `yaml:"signing"`
	ExtraArgs   List           `yaml:"extra_args"`
}

// AndroidSigning injects a release signing config without editing Gradle files.
type AndroidSigning struct {
	StoreFile     string `yaml:"store_file"`
	StorePassword string `yaml:"store_password"`
	KeyAlias      string `yaml:"key_alias"`
	KeyPassword   string `yaml:"key_password"`
}

// IOS options.
type IOS struct {
	ExportMethod       string `yaml:"export_method"`        // app-store-connect | release-testing | ad-hoc | development | enterprise ...
	ExportOptionsPlist string `yaml:"export_options_plist"` // wins over export_method
	Codesign           *bool  `yaml:"codesign"`             // false = unsigned IPA
	ExtraArgs          List   `yaml:"extra_args"`
}

// MacOS options.
type MacOS struct {
	Sign      MacSign `yaml:"sign"`
	DMG       DMG     `yaml:"dmg"`
	Pkg       Pkg     `yaml:"pkg"`
	ExtraArgs List    `yaml:"extra_args"`
}

// MacSign configures Developer ID signing + notarization of the .app/.dmg.
type MacSign struct {
	Enabled       *bool  `yaml:"enabled"`
	Identity      string `yaml:"identity"`     // "Developer ID Application: Name (TEAMID)"
	Entitlements  string `yaml:"entitlements"` // default macos/Runner/Release.entitlements
	Notarize      *bool  `yaml:"notarize"`
	NotaryProfile string `yaml:"notary_profile"` // xcrun notarytool keychain profile
	// InstallerIdentity signs the .pkg: "Developer ID Installer: Name (TEAMID)".
	InstallerIdentity string `yaml:"installer_identity"`
}

// Pkg options (macOS installer package built with pkgbuild + productbuild).
type Pkg struct {
	Identifier      string `yaml:"identifier"`       // package id; default: macOS bundle id
	InstallLocation string `yaml:"install_location"` // default /Applications
	Title           string `yaml:"title"`            // installer window title; default: app name
	Welcome         string `yaml:"welcome"`          // .html/.rtf/.txt shown first
	Readme          string `yaml:"readme"`
	License         string `yaml:"license"` // the user must agree to it
	Conclusion      string `yaml:"conclusion"`
	Background      string `yaml:"background"` // image (png/jpg/tiff)
}

// DMG options.
type DMG struct {
	Tool       string `yaml:"tool"` // auto | hdiutil | create-dmg
	VolumeName string `yaml:"volume_name"`
	Background string `yaml:"background"` // create-dmg only
}

// Windows options.
type Windows struct {
	InnoSetup InnoSetup `yaml:"inno_setup"`
	Msix      Msix      `yaml:"msix"`
	ExtraArgs List      `yaml:"extra_args"`
}

// InnoSetup options for the installer .exe.
type InnoSetup struct {
	AppID  string `yaml:"app_id"` // stable GUID; default derived from the app identifier
	Script string `yaml:"script"` // custom .iss (fpack passes /D defines)
	ISCC   string `yaml:"iscc"`   // path to ISCC.exe
}

// Msix options (uses the `msix` pub package when present).
type Msix struct {
	ExtraArgs List `yaml:"extra_args"`
}

// Linux options.
type Linux struct {
	PackageName  string   `yaml:"package_name"` // deb/rpm name (default: app name with dashes)
	Icon         string   `yaml:"icon"`         // PNG for menus/AppImage
	Categories   string   `yaml:"categories"`   // freedesktop categories
	Deb          LinuxDeb `yaml:"deb"`
	Rpm          LinuxRpm `yaml:"rpm"`
	AppImageTool string   `yaml:"appimagetool"`
	ExtraArgs    List     `yaml:"extra_args"`
}

// LinuxDeb options.
type LinuxDeb struct {
	Depends List `yaml:"depends"`
}

// LinuxRpm options.
type LinuxRpm struct {
	Requires List `yaml:"requires"`
}

// Web options.
type Web struct {
	BaseHref  string `yaml:"base_href"`
	Wasm      *bool  `yaml:"wasm"`
	ExtraArgs List   `yaml:"extra_args"`
}

// ---- flexible YAML types ----

// List accepts a YAML list or a single scalar.
type List []string

// UnmarshalYAML implements yaml.Unmarshaler.
func (l *List) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if n.Tag == "!!null" || n.Value == "" {
			*l = nil
			return nil
		}
		*l = List{n.Value}
		return nil
	case yaml.SequenceNode:
		var out []string
		for _, c := range n.Content {
			if c.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: expected a plain value in the list", c.Line)
			}
			out = append(out, c.Value)
		}
		*l = out
		return nil
	}
	return fmt.Errorf("line %d: expected a list", n.Line)
}

// Scalar accepts strings and numbers (build_number: 42).
type Scalar string

// UnmarshalYAML implements yaml.Unmarshaler.
func (s *Scalar) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: expected a single value", n.Line)
	}
	if n.Tag == "!!null" {
		*s = ""
		return nil
	}
	*s = Scalar(n.Value)
	return nil
}

// Defines accepts a map (KEY: value) or a list of KEY=VALUE strings.
type Defines []string

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Defines) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.MappingNode:
		var out []string
		for i := 0; i+1 < len(n.Content); i += 2 {
			out = append(out, n.Content[i].Value+"="+n.Content[i+1].Value)
		}
		*d = out
		return nil
	case yaml.SequenceNode, yaml.ScalarNode:
		var l List
		if err := l.UnmarshalYAML(n); err != nil {
			return err
		}
		for _, v := range l {
			if !strings.Contains(v, "=") {
				return fmt.Errorf("line %d: dart_define entry %q must be KEY=VALUE", n.Line, v)
			}
		}
		*d = Defines(l)
		return nil
	}
	return fmt.Errorf("line %d: dart_define must be a map or a list of KEY=VALUE", n.Line)
}

// ABIMode is the APK split mode.
type ABIMode string

// APK split modes.
const (
	ABIUniversal ABIMode = "false" // one universal (fat) APK
	ABISplit     ABIMode = "true"  // one APK per ABI
	ABIBoth      ABIMode = "both"  // universal + per-ABI APKs
)

// UnmarshalYAML implements yaml.Unmarshaler.
func (m *ABIMode) UnmarshalYAML(n *yaml.Node) error {
	v, err := ParseABIMode(n.Value)
	if err != nil {
		return fmt.Errorf("line %d: %v", n.Line, err)
	}
	*m = v
	return nil
}

// ParseABIMode parses false/true/both (and yes/no/universal/split).
func ParseABIMode(v string) (ABIMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "false", "no", "off", "universal", "fat":
		return ABIUniversal, nil
	case "true", "yes", "on", "split", "per-abi":
		return ABISplit, nil
	case "both", "all":
		return ABIBoth, nil
	}
	return "", fmt.Errorf("split_per_abi must be false, true or both (got %q)", v)
}

// ---- loading ----

// Find returns the config path for a project root, honoring an explicit
// path. Returns "" when no file exists (zero-config).
func Find(root, explicit string) (string, error) {
	if explicit != "" {
		p := explicit
		if !filepath.IsAbs(p) {
			if abs, err := filepath.Abs(p); err == nil {
				p = abs
			}
		}
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("config file %s not found", explicit)
		}
		return p, nil
	}
	for _, n := range FileNames {
		p := filepath.Join(root, n)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", nil
}

// Load reads and validates a config file. path "" returns an empty config.
func Load(path string, getenv func(string) string) (*Config, error) {
	c := &Config{}
	if path == "" {
		return c, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := Parse(data, c); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	c.File = path
	c.UnsetEnv = interpolate(reflect.ValueOf(c).Elem(), getenv)
	return c, nil
}

var unknownField = regexp.MustCompile(`line (\d+): field (\S+) not found in type config\.(\w+)`)

// Parse decodes YAML strictly (unknown keys are errors with a suggestion).
func Parse(data []byte, c *Config) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return friendlyYAMLError(err)
	}
	return nil
}

func friendlyYAMLError(err error) error {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "yaml: unmarshal errors:\n")
	msg = strings.TrimPrefix(msg, "yaml: ")
	var lines []string
	for _, l := range strings.Split(msg, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if m := unknownField.FindStringSubmatch(l); m != nil {
			section := sectionName(m[3])
			s := fmt.Sprintf("line %s: unknown key %q", m[1], m[2])
			if section != "" {
				s += fmt.Sprintf(" in %q", section)
			}
			if sug := suggest(m[2], keysOf(m[3])); sug != "" {
				s += fmt.Sprintf(" (did you mean %q?)", sug)
			}
			l = s
		}
		lines = append(lines, l)
	}
	return errors.New(strings.Join(lines, "; "))
}

var typeSections = map[string]reflect.Type{}

func init() {
	var walk func(prefix string, t reflect.Type)
	walk = func(prefix string, t reflect.Type) {
		typeSections[t.Name()] = t
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Type.Kind() == reflect.Struct {
				walk(prefix, f.Type)
			}
		}
	}
	walk("", reflect.TypeOf(Config{}))
}

func sectionName(typeName string) string {
	var find func(prefix string, t reflect.Type) string
	find = func(prefix string, t reflect.Type) string {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("yaml"), ",")[0]
			if tag == "" || tag == "-" || f.Type.Kind() != reflect.Struct {
				continue
			}
			p := tag
			if prefix != "" {
				p = prefix + "." + tag
			}
			if f.Type.Name() == typeName {
				return p
			}
			if r := find(p, f.Type); r != "" {
				return r
			}
		}
		return ""
	}
	return find("", reflect.TypeOf(Config{}))
}

func keysOf(typeName string) []string {
	t, ok := typeSections[typeName]
	if !ok {
		return nil
	}
	var out []string
	for i := 0; i < t.NumField(); i++ {
		tag := strings.Split(t.Field(i).Tag.Get("yaml"), ",")[0]
		if tag != "" && tag != "-" {
			out = append(out, tag)
		}
	}
	return out
}

// suggest returns the closest candidate within edit distance 3.
func suggest(s string, candidates []string) string {
	best, bestD := "", 4
	for _, c := range candidates {
		if d := levenshtein(strings.ToLower(s), strings.ToLower(c)); d < bestD {
			best, bestD = c, d
		}
	}
	return best
}

// Suggest is exported for CLI "did you mean" messages.
func Suggest(s string, candidates []string) string { return suggest(s, candidates) }

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// envRef matches an innermost ${VAR} / ${VAR:-default} (the default holds no
// further references), so nested defaults like ${A:-${B}} expand inside out.
var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^${}]*))?\}`)

// interpolate expands ${VAR} and ${VAR:-default} in every string field and
// returns the names of referenced variables that are not set.
func interpolate(v reflect.Value, getenv func(string) string) []string {
	unset := map[string]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.String:
			if v.CanSet() && strings.Contains(v.String(), "${") {
				// Values substituted in one pass are marked so that their
				// content is never expanded again (only defaults nest).
				protect := strings.NewReplacer("$", "\x00", "{", "\x01", "}", "\x02")
				restore := strings.NewReplacer("\x00", "$", "\x01", "{", "\x02", "}")
				s := v.String()
				for pass := 0; pass < 8 && envRef.MatchString(s); pass++ {
					s = envRef.ReplaceAllStringFunc(s, func(m string) string {
						sm := envRef.FindStringSubmatch(m)
						if val := getenv(sm[1]); val != "" {
							return protect.Replace(val)
						}
						if sm[2] != "" {
							return sm[3]
						}
						unset[sm[1]] = true
						return ""
					})
				}
				v.SetString(restore.Replace(s))
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		case reflect.Ptr:
			if !v.IsNil() {
				walk(v.Elem())
			}
		}
	}
	walk(v)
	var out []string
	for k := range unset {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---- environment overrides ----

// EnvVars documents every environment variable fpack reads (for --help and README).
var EnvVars = [][2]string{
	{"FPACK_FLUTTER", "Flutter SDK root"},
	{"FPACK_CONFIG", "config file path"},
	{"FPACK_MODE", "release | profile | debug"},
	{"FPACK_FLAVOR", "build flavor"},
	{"FPACK_ENTRY", "entry point (flutter -t)"},
	{"FPACK_BUILD_NAME", "override version name"},
	{"FPACK_BUILD_NUMBER", "override build number"},
	{"FPACK_OUTPUT_DIR", "artifact directory"},
	{"FPACK_OVERWRITE", "true = replace existing artifacts"},
	{"FPACK_SPLIT_PER_ABI", "false | true | both"},
	{"FPACK_OBFUSCATE", "true = --obfuscate"},
	{"FPACK_ANDROID_KEYSTORE", "Android keystore path"},
	{"FPACK_ANDROID_KEYSTORE_BASE64", "Android keystore content, base64 (CI)"},
	{"FPACK_ANDROID_KEYSTORE_PASSWORD", "keystore password"},
	{"FPACK_ANDROID_KEY_ALIAS", "key alias"},
	{"FPACK_ANDROID_KEY_PASSWORD", "key password (defaults to keystore password)"},
	{"FPACK_IOS_EXPORT_METHOD", "IPA export method"},
	{"FPACK_IOS_EXPORT_OPTIONS_PLIST", "ExportOptions.plist path"},
	{"FPACK_IOS_CODESIGN", "false = unsigned IPA"},
	{"FPACK_MACOS_SIGN", "true/false: codesign .app/.dmg/.pkg"},
	{"FPACK_MACOS_SIGN_IDENTITY", "codesign identity (Developer ID Application)"},
	{"FPACK_MACOS_INSTALLER_IDENTITY", "pkg signing identity (Developer ID Installer)"},
	{"FPACK_MACOS_NOTARIZE", "true/false: notarize the zip/.dmg/.pkg"},
	{"FPACK_MACOS_NOTARY_PROFILE", "notarytool keychain profile"},
	{"FPACK_DMG_TOOL", "auto | hdiutil | create-dmg"},
	{"FPACK_LANG", "zh | en"},
	{"NO_COLOR", "disable colors"},
}

// ApplyEnv overlays FPACK_* environment variables onto c.
func ApplyEnv(c *Config, getenv func(string) string) error {
	str := func(k string, dst *string) {
		if v := getenv(k); v != "" {
			*dst = v
		}
	}
	var errs []string
	boolean := func(k string, dst **bool) {
		if v := getenv(k); v != "" {
			b, err := ParseBool(v)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", k, err))
				return
			}
			*dst = &b
		}
	}
	str("FPACK_FLUTTER", &c.Flutter.SDK)
	str("FPACK_MODE", &c.Build.Mode)
	str("FPACK_FLAVOR", &c.Build.Flavor)
	str("FPACK_ENTRY", &c.Build.Target)
	if v := getenv("FPACK_BUILD_NAME"); v != "" {
		c.Build.BuildName = Scalar(v)
	}
	if v := getenv("FPACK_BUILD_NUMBER"); v != "" {
		c.Build.BuildNumber = Scalar(v)
	}
	str("FPACK_OUTPUT_DIR", &c.Output.Dir)
	boolean("FPACK_OVERWRITE", &c.Output.Overwrite)
	boolean("FPACK_OBFUSCATE", &c.Build.Obfuscate)
	if v := getenv("FPACK_SPLIT_PER_ABI"); v != "" {
		m, err := ParseABIMode(v)
		if err != nil {
			errs = append(errs, "FPACK_SPLIT_PER_ABI: "+err.Error())
		} else {
			c.Android.SplitPerABI = m
		}
	}
	str("FPACK_ANDROID_KEYSTORE", &c.Android.Signing.StoreFile)
	str("FPACK_ANDROID_KEYSTORE_PASSWORD", &c.Android.Signing.StorePassword)
	str("FPACK_ANDROID_KEY_ALIAS", &c.Android.Signing.KeyAlias)
	str("FPACK_ANDROID_KEY_PASSWORD", &c.Android.Signing.KeyPassword)
	str("FPACK_IOS_EXPORT_METHOD", &c.IOS.ExportMethod)
	str("FPACK_IOS_EXPORT_OPTIONS_PLIST", &c.IOS.ExportOptionsPlist)
	boolean("FPACK_IOS_CODESIGN", &c.IOS.Codesign)
	boolean("FPACK_MACOS_SIGN", &c.MacOS.Sign.Enabled)
	str("FPACK_MACOS_SIGN_IDENTITY", &c.MacOS.Sign.Identity)
	str("FPACK_MACOS_INSTALLER_IDENTITY", &c.MacOS.Sign.InstallerIdentity)
	boolean("FPACK_MACOS_NOTARIZE", &c.MacOS.Sign.Notarize)
	str("FPACK_MACOS_NOTARY_PROFILE", &c.MacOS.Sign.NotaryProfile)
	str("FPACK_DMG_TOOL", &c.MacOS.DMG.Tool)
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// ParseBool accepts true/false/yes/no/1/0/on/off.
func ParseBool(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b, nil
	}
	return false, fmt.Errorf("expected true or false, got %q", v)
}

// ---- effective values with defaults ----

// Default values.
const (
	DefaultMode         = "release"
	DefaultOutputDir    = "dist/{version}{+build}"
	DefaultNameTemplate = "{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}"
)

// DefaultABIs are Flutter's default Android target ABIs.
var DefaultABIs = []string{"armeabi-v7a", "arm64-v8a", "x86_64"}

// Mode returns the build mode.
func (c *Config) Mode() string {
	if c.Build.Mode == "" {
		return DefaultMode
	}
	return strings.ToLower(c.Build.Mode)
}

// OutputDir returns the output directory template.
func (c *Config) OutputDir() string {
	if c.Output.Dir == "" {
		return DefaultOutputDir
	}
	return c.Output.Dir
}

// NameTemplate returns the artifact name template.
func (c *Config) NameTemplate() string {
	if c.Output.Name == "" {
		return DefaultNameTemplate
	}
	return c.Output.Name
}

// Overwrite reports whether existing artifacts may be replaced.
func (c *Config) Overwrite() bool { return c.Output.Overwrite != nil && *c.Output.Overwrite }

// Checksums reports whether SHA256SUMS should be written.
func (c *Config) Checksums() bool { return c.Output.Checksums == nil || *c.Output.Checksums }

// Obfuscate reports whether Dart code is obfuscated.
func (c *Config) Obfuscate() bool { return c.Build.Obfuscate != nil && *c.Build.Obfuscate }

// ABIs returns the configured Android ABIs.
func (c *Config) ABIs() []string {
	if len(c.Android.ABIs) == 0 {
		return DefaultABIs
	}
	return c.Android.ABIs
}

// SplitPerABI returns the APK split mode.
func (c *Config) SplitPerABI() ABIMode {
	if c.Android.SplitPerABI == "" {
		return ABIUniversal
	}
	return c.Android.SplitPerABI
}

// IOSCodesign reports whether the IPA is signed.
func (c *Config) IOSCodesign() bool { return c.IOS.Codesign == nil || *c.IOS.Codesign }

// Known value sets.
var (
	Modes         = []string{"release", "profile", "debug"}
	KnownABIs     = []string{"armeabi-v7a", "arm64-v8a", "x86_64"}
	ExportMethods = []string{"app-store-connect", "app-store", "release-testing", "ad-hoc", "development", "debugging", "enterprise"}
	DMGTools      = []string{"auto", "hdiutil", "create-dmg"}
)

// Validate checks value ranges and returns human readable problems.
func (c *Config) Validate() []string {
	var p []string
	if !contains(Modes, c.Mode()) {
		p = append(p, fmt.Sprintf("build.mode must be one of %s (got %q)", strings.Join(Modes, ", "), c.Build.Mode))
	}
	for _, a := range c.Android.ABIs {
		if !contains(KnownABIs, a) {
			p = append(p, fmt.Sprintf("android.abis: unknown ABI %q (use %s)", a, strings.Join(KnownABIs, ", ")))
		}
	}
	if m := c.IOS.ExportMethod; m != "" && !contains(ExportMethods, m) {
		p = append(p, fmt.Sprintf("ios.export_method must be one of %s (got %q)", strings.Join(ExportMethods, ", "), m))
	}
	if l := c.MacOS.Pkg.InstallLocation; l != "" && !strings.HasPrefix(l, "/") {
		p = append(p, fmt.Sprintf("macos.pkg.install_location must be an absolute path (got %q)", l))
	}
	if t := c.MacOS.DMG.Tool; t != "" && !contains(DMGTools, t) {
		p = append(p, fmt.Sprintf("macos.dmg.tool must be one of %s (got %q)", strings.Join(DMGTools, ", "), t))
	}
	s := c.Android.Signing
	if s.StoreFile != "" && (s.StorePassword == "" || s.KeyAlias == "") {
		var missing []string
		if s.StorePassword == "" {
			missing = append(missing, "store_password (FPACK_ANDROID_KEYSTORE_PASSWORD)")
		}
		if s.KeyAlias == "" {
			missing = append(missing, "key_alias (FPACK_ANDROID_KEY_ALIAS)")
		}
		p = append(p, "android.signing: store_file is set but missing "+strings.Join(missing, ", "))
	}
	return p
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
