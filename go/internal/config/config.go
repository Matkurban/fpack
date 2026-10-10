// Package config loads fpack.yaml and resolves effective settings with the
// precedence: command-line flags > environment variables > fpack.yaml >
// built-in defaults. Every field is optional; zero config works.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Matkurban/fpack/go/internal/i18n"
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
	Hooks   Hooks   `yaml:"hooks"`
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
	SupportURL  string `yaml:"support_url"`
	Maintainer  string `yaml:"maintainer"` // "Name <email>" for .deb
	Copyright   string `yaml:"copyright"`
	License     string `yaml:"license"` // SPDX id, e.g. MIT
}

// Flutter selects the SDK.
type Flutter struct {
	SDK string `yaml:"sdk"` // path to the Flutter SDK root (default: auto)
}

// Build holds options shared by every flutter build.
type Build struct {
	Targets            List    `yaml:"targets"` // default targets for `fpack build`
	Mode               string  `yaml:"mode"`    // release | profile | debug
	Flavor             string  `yaml:"flavor"`
	Target             string  `yaml:"target"` // entry point, e.g. lib/main_prod.dart
	DartDefine         Defines `yaml:"dart_define"`
	DartDefineFromFile List    `yaml:"dart_define_from_file"`
	BuildName          Scalar  `yaml:"build_name"`
	BuildNumber        Scalar  `yaml:"build_number"`
	Obfuscate          *bool   `yaml:"obfuscate"`
	SplitDebugInfo     string  `yaml:"split_debug_info"`
	TreeShakeIcons     *bool   `yaml:"tree_shake_icons"`
	ExtraArgs          List    `yaml:"extra_args"` // appended to every flutter build
}

// Output controls where artifacts go and how they are named.
type Output struct {
	Dir               string            `yaml:"dir"`   // default: dist/{version}+{build}
	Name              string            `yaml:"name"`  // file name template (without extension)
	Names             map[string]string `yaml:"names"` // per-target name templates
	Overwrite         *bool             `yaml:"overwrite"`
	Checksums         *bool             `yaml:"checksums"`
	ChecksumAlgorithm string            `yaml:"checksum_algorithm"` // sha256 | sha512
}

// Hooks are shell commands run around the build (cwd: project root).
type Hooks struct {
	PreBuild    List            `yaml:"pre_build"`    // once, before the first flutter build
	PostBuild   List            `yaml:"post_build"`   // once, after all targets
	PrePackage  map[string]List `yaml:"pre_package"`  // per target, before packaging
	PostPackage map[string]List `yaml:"post_package"` // per target, after its artifacts exist
}

// Android options.
type Android struct {
	SplitPerABI ABIMode        `yaml:"split_per_abi"` // false | true | both
	ABIs        List           `yaml:"abis"`          // arm64-v8a, armeabi-v7a, x86_64
	Signing     AndroidSigning `yaml:"signing"`
	// ProjectArgs are Gradle project properties (flutter -P key=value).
	ProjectArgs map[string]string `yaml:"project_args"`
	ExtraArgs   List              `yaml:"extra_args"`
}

// AndroidSigning injects a release signing config without editing Gradle files.
type AndroidSigning struct {
	StoreFile     string `yaml:"store_file"`
	StorePassword string `yaml:"store_password"`
	KeyAlias      string `yaml:"key_alias"`
	KeyPassword   string `yaml:"key_password"`
	V1            *bool  `yaml:"v1"` // APK signature schemes (unset = Gradle default)
	V2            *bool  `yaml:"v2"`
	V3            *bool  `yaml:"v3"`
	V4            *bool  `yaml:"v4"`
}

// IOS options.
type IOS struct {
	ExportMethod         string            `yaml:"export_method"`        // app-store-connect | release-testing | ad-hoc | development | enterprise ...
	ExportOptionsPlist   string            `yaml:"export_options_plist"` // wins over everything below
	Codesign             *bool             `yaml:"codesign"`             // false = unsigned IPA
	TeamID               string            `yaml:"team_id"`
	SigningStyle         string            `yaml:"signing_style"` // automatic | manual
	SigningCertificate   string            `yaml:"signing_certificate"`
	ProvisioningProfiles map[string]string `yaml:"provisioning_profiles"` // bundle id → profile name/UUID
	UploadSymbols        *bool             `yaml:"upload_symbols"`
	ManageVersion        *bool             `yaml:"manage_app_version_and_build_number"`
	Destination          string            `yaml:"destination"` // export | upload
	Thinning             string            `yaml:"thinning"`
	StripSwiftSymbols    *bool             `yaml:"strip_swift_symbols"`
	ExportOptions        map[string]any    `yaml:"export_options"` // extra ExportOptions.plist keys
	ExtraArgs            List              `yaml:"extra_args"`
}

// MacOS options.
type MacOS struct {
	Sign      MacSign  `yaml:"sign"`
	Notarize  Notarize `yaml:"notarize"`
	DMG       DMG      `yaml:"dmg"`
	Pkg       Pkg      `yaml:"pkg"`
	ExtraArgs List     `yaml:"extra_args"`
}

// MacSign configures Developer ID signing + notarization of the .app/.dmg/.pkg.
type MacSign struct {
	Enabled         *bool  `yaml:"enabled"`
	Identity        string `yaml:"identity"`     // "Developer ID Application: Name (TEAMID)"
	Entitlements    string `yaml:"entitlements"` // default macos/Runner/Release.entitlements
	HardenedRuntime *bool  `yaml:"hardened_runtime"`
	Notarize        *bool  `yaml:"notarize"`
	NotaryProfile   string `yaml:"notary_profile"` // xcrun notarytool keychain profile
	NotaryAppleID   string `yaml:"notary_apple_id"`
	NotaryTeamID    string `yaml:"notary_team_id"`
	NotaryPassword  string `yaml:"notary_password"` // app-specific password
	NotaryAPIKey    string `yaml:"notary_api_key"`  // App Store Connect API key (.p8)
	NotaryAPIKeyID  string `yaml:"notary_api_key_id"`
	NotaryAPIIssuer string `yaml:"notary_api_issuer"`
	// InstallerIdentity signs the .pkg: "Developer ID Installer: Name (TEAMID)".
	InstallerIdentity string `yaml:"installer_identity"`
}

// Notarize controls how fpack waits for Apple's notary service.
type Notarize struct {
	// Wait keeps fpack attached until Apple answers (default true); false
	// submits, records the submission and finishes ("submitted").
	Wait *bool `yaml:"wait"`
}

// NotarizeWait reports whether fpack waits for notarization results.
func (c *Config) NotarizeWait() bool { return c.MacOS.Notarize.Wait == nil || *c.MacOS.Notarize.Wait }

// DMG options.
type DMG struct {
	Tool                 string `yaml:"tool"` // auto | hdiutil | create-dmg
	VolumeName           string `yaml:"volume_name"`
	VolumeIcon           string `yaml:"volume_icon"` // .icns
	Background           string `yaml:"background"`
	WindowPosition       Pair   `yaml:"window_position"`
	WindowSize           Pair   `yaml:"window_size"`
	IconSize             int    `yaml:"icon_size"`
	AppPosition          Pair   `yaml:"app_position"`
	ApplicationsPosition Pair   `yaml:"applications_position"`
	Format               string `yaml:"format"`     // UDZO | UDBZ | ULFO | ULMO | UDRO
	Filesystem           string `yaml:"filesystem"` // HFS+ | APFS
	License              string `yaml:"license"`    // EULA shown when the DMG is opened
}

// Pkg options (macOS installer package built with pkgbuild + productbuild).
type Pkg struct {
	Identifier      string `yaml:"identifier"` // package id; default: macOS bundle id
	Version         string `yaml:"version"`    // default: build name
	InstallLocation string `yaml:"install_location"`
	Title           string `yaml:"title"`
	Welcome         string `yaml:"welcome"`
	Readme          string `yaml:"readme"`
	License         string `yaml:"license"`
	Conclusion      string `yaml:"conclusion"`
	Background      string `yaml:"background"`
	MinOS           string `yaml:"min_os"` // e.g. 10.15
	Preinstall      string `yaml:"preinstall"`
	Postinstall     string `yaml:"postinstall"`
	Relocatable     *bool  `yaml:"relocatable"`
	RequireRestart  *bool  `yaml:"require_restart"`
}

// Windows options.
type Windows struct {
	InnoSetup InnoSetup `yaml:"inno_setup"`
	Msix      Msix      `yaml:"msix"`
	Sign      WinSign   `yaml:"sign"`
	ExtraArgs List      `yaml:"extra_args"`
}

// InnoSetup options for the installer .exe.
type InnoSetup struct {
	AppID            string `yaml:"app_id"` // stable GUID; default derived from the app identifier
	Script           string `yaml:"script"` // custom .iss (fpack passes /D defines)
	ISCC             string `yaml:"iscc"`   // path to ISCC.exe
	Publisher        string `yaml:"publisher"`
	PublisherURL     string `yaml:"publisher_url"`
	SupportURL       string `yaml:"support_url"`
	UpdatesURL       string `yaml:"updates_url"`
	DefaultDir       string `yaml:"default_dir"`
	GroupName        string `yaml:"group_name"`
	DesktopIcon      string `yaml:"desktop_icon"` // none | unchecked | checked
	RunAfterInstall  *bool  `yaml:"run_after_install"`
	LicenseFile      string `yaml:"license_file"`
	InfoBefore       string `yaml:"info_before"`
	InfoAfter        string `yaml:"info_after"`
	SetupIcon        string `yaml:"setup_icon"`
	WizardImage      string `yaml:"wizard_image"`
	WizardSmallImage string `yaml:"wizard_small_image"`
	WizardStyle      string `yaml:"wizard_style"` // modern | classic
	Languages        List   `yaml:"languages"`    // en, zh-CN, zh-TW, ja, ...
	Privileges       string `yaml:"privileges"`   // user | admin | ask
	Compression      string `yaml:"compression"`
	MinVersion       string `yaml:"min_version"` // MinVersion=, e.g. 10.0
}

// WinSign signs executables and installers with signtool.
type WinSign struct {
	Certificate  string `yaml:"certificate"` // .pfx
	Password     string `yaml:"password"`
	Thumbprint   string `yaml:"thumbprint"` // certificate in the Windows store (instead of a .pfx)
	TimestampURL string `yaml:"timestamp_url"`
	Signtool     string `yaml:"signtool"`
	Description  string `yaml:"description"`
}

// Msix options (uses the `msix` pub package).
type Msix struct {
	DisplayName          string `yaml:"display_name"`
	PublisherDisplayName string `yaml:"publisher_display_name"`
	IdentityName         string `yaml:"identity_name"`
	Publisher            string `yaml:"publisher"` // CN=... of the signing certificate
	Version              string `yaml:"version"`   // a.b.c.d
	Logo                 string `yaml:"logo"`
	Description          string `yaml:"description"`
	Capabilities         List   `yaml:"capabilities"`
	Languages            List   `yaml:"languages"`
	FileExtensions       List   `yaml:"file_extensions"`
	ProtocolActivation   List   `yaml:"protocol_activation"`
	ExecutionAlias       string `yaml:"execution_alias"`
	StartAtLogin         *bool  `yaml:"start_at_login"`
	OSMinVersion         string `yaml:"os_min_version"`
	Store                *bool  `yaml:"store"`
	Sign                 *bool  `yaml:"sign"`
	Certificate          string `yaml:"certificate"`
	CertificatePassword  string `yaml:"certificate_password"`
	ExtraArgs            List   `yaml:"extra_args"`
}

// Linux options.
type Linux struct {
	PackageName    string        `yaml:"package_name"` // deb/rpm name (default: app name with dashes)
	Prefix         string        `yaml:"prefix"`       // install directory, default /opt/<package>
	Icon           string        `yaml:"icon"`         // PNG for menus/AppImage
	IconSizes      []int         `yaml:"icon_sizes"`
	Categories     List          `yaml:"categories"` // freedesktop categories
	GenericName    string        `yaml:"generic_name"`
	Keywords       List          `yaml:"keywords"`
	MimeTypes      List          `yaml:"mime_types"`
	StartupWMClass string        `yaml:"startup_wm_class"`
	Metainfo       string        `yaml:"metainfo"` // AppStream .metainfo.xml
	Deb            LinuxDeb      `yaml:"deb"`
	Rpm            LinuxRpm      `yaml:"rpm"`
	AppImage       LinuxAppImage `yaml:"appimage"`
	AppImageTool   string        `yaml:"appimagetool"`
	ExtraArgs      List          `yaml:"extra_args"`
}

// LinuxDeb options.
type LinuxDeb struct {
	Depends    List   `yaml:"depends"`
	Recommends List   `yaml:"recommends"`
	Suggests   List   `yaml:"suggests"`
	Conflicts  List   `yaml:"conflicts"`
	Section    string `yaml:"section"`
	Priority   string `yaml:"priority"`
	Preinst    string `yaml:"preinst"`
	Postinst   string `yaml:"postinst"`
	Prerm      string `yaml:"prerm"`
	Postrm     string `yaml:"postrm"`
}

// LinuxRpm options.
type LinuxRpm struct {
	Requires List   `yaml:"requires"`
	Group    string `yaml:"group"`
	License  string `yaml:"license"`
	Pre      string `yaml:"pre"`
	Post     string `yaml:"post"`
	Preun    string `yaml:"preun"`
	Postun   string `yaml:"postun"`
}

// LinuxAppImage options.
type LinuxAppImage struct {
	UpdateInformation string `yaml:"update_information"`
	ExtraArgs         List   `yaml:"extra_args"` // appended to appimagetool
}

// Web options.
type Web struct {
	BaseHref          string            `yaml:"base_href"`
	Wasm              *bool             `yaml:"wasm"`
	SourceMaps        *bool             `yaml:"source_maps"`
	CSP               *bool             `yaml:"csp"`
	OptimizationLevel *int              `yaml:"optimization_level"` // 0-4
	StaticAssetsURL   string            `yaml:"static_assets_url"`
	WebResourcesCDN   *bool             `yaml:"web_resources_cdn"`
	WebDefine         map[string]string `yaml:"web_define"`
	ExtraArgs         List              `yaml:"extra_args"`
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
				return fmt.Errorf(i18n.S("line %d: expected a plain value in the list", "第 %d 行：列表中应为普通值"), c.Line)
			}
			out = append(out, c.Value)
		}
		*l = out
		return nil
	}
	return fmt.Errorf(i18n.S("line %d: expected a list", "第 %d 行：应为列表"), n.Line)
}

// Pair is an [x, y] / [width, height] value ("x,y" is accepted too).
type Pair []int

// UnmarshalYAML implements yaml.Unmarshaler.
func (p *Pair) UnmarshalYAML(n *yaml.Node) error {
	var parts []string
	switch n.Kind {
	case yaml.SequenceNode:
		for _, c := range n.Content {
			parts = append(parts, c.Value)
		}
	case yaml.ScalarNode:
		if n.Tag == "!!null" || n.Value == "" {
			*p = nil
			return nil
		}
		parts = strings.FieldsFunc(n.Value, func(r rune) bool { return r == ',' || r == 'x' || r == ' ' })
	}
	if len(parts) != 2 {
		return fmt.Errorf(i18n.S("line %d: expected two numbers like [600, 400]", "第 %d 行：应为两个数字，如 [600, 400]"), n.Line)
	}
	out := Pair{}
	for _, v := range parts {
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || i < 0 {
			return fmt.Errorf(i18n.S("line %d: expected two numbers like [600, 400] (got %q)", "第 %d 行：应为两个数字，如 [600, 400]（实际为 %q）"), n.Line, v)
		}
		out = append(out, i)
	}
	*p = out
	return nil
}

// Scalar accepts strings and numbers (build_number: 42).
type Scalar string

// UnmarshalYAML implements yaml.Unmarshaler.
func (s *Scalar) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf(i18n.S("line %d: expected a single value", "第 %d 行：应为单个值"), n.Line)
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
				return fmt.Errorf(i18n.S("line %d: dart_define entry %q must be KEY=VALUE", "第 %d 行：dart_define 条目 %q 必须是 KEY=VALUE 形式"), n.Line, v)
			}
		}
		*d = Defines(l)
		return nil
	}
	return fmt.Errorf(i18n.S("line %d: dart_define must be a map or a list of KEY=VALUE", "第 %d 行：dart_define 必须是映射或 KEY=VALUE 列表"), n.Line)
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
		return fmt.Errorf(i18n.S("line %d: %v", "第 %d 行：%v"), n.Line, err)
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
	return "", fmt.Errorf(i18n.S("split_per_abi must be false, true or both (got %q)", "split_per_abi 必须是 false、true 或 both（实际为 %q）"), v)
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
			return "", fmt.Errorf(i18n.S("config file %s not found", "找不到配置文件 %s"), explicit)
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
		// Values such as `obfuscate: ${OBF:-false}` or `split_per_abi:
		// ${SPLIT}` only get their type once expanded: expand them in the
		// YAML tree and parse again. Errors keep the original line numbers.
		if !bytes.Contains(data, []byte("${")) {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		exp, unset, xerr := expandYAML(data, getenv)
		c2 := &Config{}
		if xerr != nil || Parse(exp, c2) != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		c2.File, c2.UnsetEnv = path, unset
		return c2, nil
	}
	c.File = path
	c.UnsetEnv = interpolate(reflect.ValueOf(c).Elem(), getenv)
	return c, nil
}

// expandYAML expands ${VAR} references in every scalar value of the YAML
// document and re-types them, so `${OBF:-false}` (quoted or not) becomes a
// boolean for boolean keys.
func expandYAML(data []byte, getenv func(string) string) ([]byte, []string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, nil, err
	}
	unset := map[string]bool{}
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		switch n.Kind {
		case yaml.DocumentNode, yaml.SequenceNode:
			for _, c := range n.Content {
				walk(c)
			}
		case yaml.MappingNode:
			for i := 1; i < len(n.Content); i += 2 {
				walk(n.Content[i])
			}
		case yaml.ScalarNode:
			if strings.Contains(n.Value, "${") {
				// Re-type the expanded value ("false" → bool for bool
				// keys); string keys still receive the text unchanged.
				n.Value = expandString(n.Value, getenv, unset)
				n.Tag, n.Style = "", 0
			}
		}
	}
	walk(&root)
	out, err := yaml.Marshal(&root)
	var names []string
	for k := range unset {
		names = append(names, k)
	}
	sort.Strings(names)
	return out, names, err
}

// expandString expands ${VAR} and ${VAR:-default}; substituted values are
// never expanded again (only defaults nest). Unset names go to unset.
func expandString(s string, getenv func(string) string, unset map[string]bool) string {
	protect := strings.NewReplacer("$", "\x00", "{", "\x01", "}", "\x02")
	restore := strings.NewReplacer("\x00", "$", "\x01", "{", "\x02", "}")
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
	return restore.Replace(s)
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
		return friendlyYAMLError(err, keyLines(data))
	}
	return nil
}

// keyLines maps line numbers to the dotted key whose value starts there.
func keyLines(data []byte) map[int]string {
	out := map[int]string{}
	var root yaml.Node
	if yaml.Unmarshal(data, &root) != nil || len(root.Content) == 0 {
		return out
	}
	var walk func(prefix string, n *yaml.Node)
	walk = func(prefix string, n *yaml.Node) {
		if n.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			p := key.Value
			if prefix != "" {
				p = prefix + "." + key.Value
			}
			out[val.Line] = p
			if _, isKey := KeyByPath(p); !isKey {
				walk(p, val)
			}
		}
	}
	walk("", root.Content[0])
	return out
}

var typeError = regexp.MustCompile("^line (\\d+): cannot unmarshal !!(\\w+) (?:`([^`]*)` )?into (.+)$")

func expectedFor(goType string) string {
	switch {
	case goType == "bool" || goType == "*bool":
		return i18n.S("true or false", "true 或 false")
	case strings.Contains(goType, "int"):
		return i18n.S("a number", "数字")
	case strings.HasPrefix(goType, "map["):
		return i18n.S("a map (key: value)", "映射（key: value）")
	case strings.HasPrefix(goType, "[]"):
		return i18n.S("a list", "列表")
	case goType == "string":
		return i18n.S("a text value", "文本值")
	case strings.HasPrefix(goType, "config."):
		return i18n.S("a section of keys (key: value on indented lines)", "键的分组（缩进的 key: value 行）")
	}
	return goType
}

func friendlyYAMLError(err error, keyLines map[int]string) error {
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
			s := fmt.Sprintf(i18n.S("line %s: unknown key %q", "第 %s 行：未知键 %q"), m[1], m[2])
			if section != "" {
				s += fmt.Sprintf(i18n.S(" in %q", "（位于 %q）"), section)
			}
			if sug := suggest(m[2], keysOf(m[3])); sug != "" {
				s += fmt.Sprintf(i18n.S(" (did you mean %q?)", "（你是不是想写 %q？）"), sug)
			}
			l = s
		} else if m := typeError.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[1])
			where := ""
			if k := keyLines[n]; k != "" {
				where = k + ": "
			}
			got := m[3]
			if m[2] == "map" || m[2] == "seq" {
				got = map[string]string{"map": i18n.S("a map", "映射"), "seq": i18n.S("a list", "列表")}[m[2]]
			} else {
				got = strconv.Quote(got)
			}
			l = fmt.Sprintf(i18n.S("line %s: %sexpected %s, got %s", "第 %s 行：%s应为 %s，实际为 %s"), m[1], where, expectedFor(m[4]), got)
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

// levenshtein is the optimal-string-alignment distance: an adjacent
// transposition ("wbe" → "web") counts as one edit.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
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
				v.SetString(expandString(v.String(), getenv, unset))
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
		case reflect.Map:
			// Map values are not addressable: walk a copy and store it back.
			for _, key := range v.MapKeys() {
				cp := reflect.New(v.Type().Elem()).Elem()
				cp.Set(v.MapIndex(key))
				walk(cp)
				v.SetMapIndex(key, cp)
			}
		case reflect.Interface:
			if !v.IsNil() && v.CanSet() {
				cp := reflect.New(v.Elem().Type()).Elem()
				cp.Set(v.Elem())
				walk(cp)
				v.Set(cp)
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

// extraEnv are variables that don't map to a single fpack.yaml key.
var extraEnv = [][2]string{
	{"FPACK_CONFIG", "config file path"},
	{"FPACK_ANDROID_KEYSTORE_BASE64", "Android keystore content, base64 (CI)"},
	{"FPACK_LANG", "zh | en"},
	{"NO_COLOR", "disable colors"},
}

var extraEnvZH = map[string]string{
	"FPACK_CONFIG":                  "配置文件路径",
	"FPACK_ANDROID_KEYSTORE_BASE64": "Android keystore 内容，base64（CI）",
	"FPACK_LANG":                    "zh | en",
	"NO_COLOR":                      "关闭颜色",
}

// EnvVars documents every environment variable fpack reads (for --help and
// the docs): one per registry key with Env, plus a few extras.
func EnvVars() [][2]string {
	var out [][2]string
	for _, k := range Keys {
		if k.Env != "" {
			out = append(out, [2]string{k.Env, k.Path})
		}
	}
	for _, e := range extraEnv {
		out = append(out, [2]string{e[0], i18n.S(e[1], extraEnvZH[e[0]])})
	}
	return out
}

// ApplyEnv overlays FPACK_* environment variables onto c (see Keys).
func ApplyEnv(c *Config, getenv func(string) string) error {
	var errs []string
	for _, k := range Keys {
		if k.Env == "" {
			continue
		}
		v := getenv(k.Env)
		if v == "" {
			continue
		}
		if err := c.SetString(k.Path, v); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", k.Env, err))
		}
	}
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
	return false, fmt.Errorf(i18n.S("expected true or false, got %q", "应为 true 或 false，实际为 %q"), v)
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
	// Allowed values from the registry.
	for _, k := range Keys {
		if len(k.Enum) == 0 {
			continue
		}
		val, _ := c.Get(k.Path)
		var vals []string
		switch x := val.(type) {
		case string:
			vals = []string{x}
		case Scalar:
			vals = []string{string(x)}
		case List:
			vals = x
		}
		for _, v := range vals {
			if v != "" && !contains(k.Enum, v) {
				msg := fmt.Sprintf(i18n.S("%s must be one of %s (got %q)", "%s 必须是 %s 之一（实际为 %q）"), k.Path, strings.Join(k.Enum, ", "), v)
				if s := suggest(v, k.Enum); s != "" {
					msg += fmt.Sprintf(i18n.S(" — did you mean %q?", "——你是不是想写 %q？"), s)
				}
				p = append(p, msg)
			}
		}
	}
	for _, a := range c.Android.ABIs {
		if !contains(KnownABIs, a) {
			p = append(p, fmt.Sprintf(i18n.S("android.abis: unknown ABI %q (use %s)", "android.abis：未知 ABI %q（可用 %s）"), a, strings.Join(KnownABIs, ", ")))
		}
	}
	for _, l := range c.Windows.InnoSetup.Languages {
		if _, ok := InnoLanguages[strings.ToLower(l)]; !ok && !strings.HasSuffix(strings.ToLower(l), ".isl") {
			p = append(p, fmt.Sprintf(i18n.S("windows.inno_setup.languages: unknown language %q (use %s, or a path to an .isl file)", "windows.inno_setup.languages：未知语言 %q（可用 %s，或 .isl 文件路径）"), l, strings.Join(InnoLanguageNames(), ", ")))
		}
	}
	if l := c.MacOS.Pkg.InstallLocation; l != "" && !strings.HasPrefix(l, "/") {
		p = append(p, fmt.Sprintf(i18n.S("macos.pkg.install_location must be an absolute path (got %q)", "macos.pkg.install_location 必须是绝对路径（实际为 %q）"), l))
	}
	if l := c.Linux.Prefix; l != "" && (!strings.HasPrefix(l, "/") || l == "/") {
		p = append(p, fmt.Sprintf(i18n.S("linux.prefix must be an absolute directory such as /opt (got %q)", "linux.prefix 必须是绝对目录，如 /opt（实际为 %q）"), l))
	}
	if o := c.Web.OptimizationLevel; o != nil && (*o < 0 || *o > 4) {
		p = append(p, fmt.Sprintf(i18n.S("web.optimization_level must be 0-4 (got %d)", "web.optimization_level 必须在 0-4 之间（实际为 %d）"), *o))
	}
	if b := c.Web.BaseHref; b != "" && (!strings.HasPrefix(b, "/") || !strings.HasSuffix(b, "/")) {
		p = append(p, fmt.Sprintf(i18n.S("web.base_href must start and end with \"/\" (got %q, e.g. \"/app/\")", "web.base_href 必须以 \"/\" 开头和结尾（实际为 %q，例如 \"/app/\"）"), b))
	}
	if n := c.MacOS.DMG.IconSize; n != 0 && (n < 16 || n > 512) {
		p = append(p, fmt.Sprintf(i18n.S("macos.dmg.icon_size must be 16-512 (got %d)", "macos.dmg.icon_size 必须在 16-512 之间（实际为 %d）"), n))
	}
	for _, sz := range c.Linux.IconSizes {
		if sz < 16 || sz > 1024 {
			p = append(p, fmt.Sprintf(i18n.S("linux.icon_sizes: %d is out of range 16-1024", "linux.icon_sizes：%d 超出范围 16-1024"), sz))
		}
	}
	if g := c.Windows.InnoSetup.AppID; g != "" && !guidRe.MatchString(strings.Trim(g, "{}")) {
		p = append(p, fmt.Sprintf(i18n.S("windows.inno_setup.app_id must be a GUID like 8F0E7C2A-1B3D-4E5F-9A6B-7C8D9E0F1A2B (got %q)", "windows.inno_setup.app_id 必须是 GUID，如 8F0E7C2A-1B3D-4E5F-9A6B-7C8D9E0F1A2B（实际为 %q）"), g))
	}
	for name, tmpl := range c.Output.Names {
		if !contains(TargetNames, name) {
			p = append(p, fmt.Sprintf(i18n.S("output.names: unknown target %q%s", "output.names：未知目标 %q%s"), name, didYouMean(name, TargetNames)))
		}
		if bad := unknownPlaceholders(tmpl); len(bad) > 0 {
			p = append(p, fmt.Sprintf(i18n.S("output.names.%s: unknown placeholder %s (available: %s)", "output.names.%s：未知占位符 %s（可用：%s）"), name, strings.Join(bad, ", "), strings.Join(NamePlaceholders, ", ")))
		}
	}
	if bad := unknownPlaceholders(c.Output.Name); len(bad) > 0 {
		p = append(p, fmt.Sprintf(i18n.S("output.name: unknown placeholder %s (available: %s)", "output.name：未知占位符 %s（可用：%s）"), strings.Join(bad, ", "), strings.Join(NamePlaceholders, ", ")))
	}
	for _, m := range []map[string]List{c.Hooks.PrePackage, c.Hooks.PostPackage} {
		for name := range m {
			if !contains(TargetNames, name) {
				p = append(p, fmt.Sprintf(i18n.S("hooks: unknown target %q%s", "hooks：未知目标 %q%s"), name, didYouMean(name, TargetNames)))
			}
		}
	}
	if c.IOS.SigningStyle == "manual" && len(c.IOS.ProvisioningProfiles) == 0 {
		p = append(p, i18n.S("ios.signing_style is manual but ios.provisioning_profiles is empty (map bundle id -> profile name)", "ios.signing_style 为 manual，但 ios.provisioning_profiles 为空（bundle id -> 描述文件名称 的映射）"))
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
		p = append(p, i18n.S("android.signing: store_file is set but missing ", "android.signing：已设置 store_file，但缺少 ")+strings.Join(missing, i18n.S(", ", "、")))
	}
	ms := c.MacOS.Sign
	if (ms.NotaryAppleID != "" || ms.NotaryPassword != "") && (ms.NotaryAppleID == "" || ms.NotaryTeamID == "" || ms.NotaryPassword == "") {
		p = append(p, i18n.S("macos.sign: Apple ID notarization needs notary_apple_id, notary_team_id and notary_password together", "macos.sign：Apple ID 公证需要同时设置 notary_apple_id、notary_team_id 和 notary_password"))
	}
	if (ms.NotaryAPIKey != "" || ms.NotaryAPIKeyID != "") && (ms.NotaryAPIKey == "" || ms.NotaryAPIKeyID == "") {
		p = append(p, i18n.S("macos.sign: API key notarization needs notary_api_key and notary_api_key_id (plus notary_api_issuer for team keys)", "macos.sign：API 密钥公证需要 notary_api_key 和 notary_api_key_id（团队密钥还需 notary_api_issuer）"))
	}
	ws := c.Windows.Sign
	if ws.Certificate != "" && ws.Thumbprint != "" {
		p = append(p, i18n.S("windows.sign: set either certificate (.pfx) or thumbprint (certificate store), not both", "windows.sign：certificate（.pfx）与 thumbprint（证书存储）只能设置其一"))
	}
	return p
}

// NamePlaceholders are the keys of file name templates (same as
// pack.Placeholders; a test keeps them in sync).
var NamePlaceholders = []string{"app", "version", "build", "platform", "arch", "variant", "mode", "flavor", "target", "date"}

var namePlaceholder = regexp.MustCompile(`\{([-_.+]?)([a-z]+)\}`)

// unknownPlaceholders returns the placeholders of tmpl that are not known.
func unknownPlaceholders(tmpl string) []string {
	var bad []string
	for _, sm := range namePlaceholder.FindAllStringSubmatch(tmpl, -1) {
		if !contains(NamePlaceholders, sm[2]) {
			bad = append(bad, sm[0])
		}
	}
	return bad
}

var guidRe = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)

// TargetNames are the target keys accepted in per-target maps (kept in sync
// with the targets registry by a test there).
var TargetNames = []string{"apk", "aab", "ipa", "macos", "dmg", "pkg", "windows", "exe", "msix", "linux", "deb", "rpm", "appimage", "web"}

// TargetAliases are the alternative names accepted for targets on the
// command line and in build.targets (kept in sync with targets.aliases by a
// test in the targets package).
var TargetAliases = []string{"AppImage", "android", "app", "appbundle", "bundle", "debian", "fedora", "image", "inno", "installer", "ios", "mac", "osx", "portable", "setup", "tar", "tar.gz", "tgz", "win", "zip"}

func didYouMean(v string, c []string) string {
	if s := suggest(v, c); s != "" {
		return fmt.Sprintf(i18n.S(" (did you mean %q?)", "（你是不是想写 %q？）"), s)
	}
	return ""
}

// InnoLanguages maps fpack language codes to Inno Setup message files.
var InnoLanguages = map[string]string{
	"en": "Default.isl", "zh": "ChineseSimplified.isl", "zh-cn": "ChineseSimplified.isl", "zh-hans": "ChineseSimplified.isl",
	"zh-tw": "ChineseTraditional.isl", "zh-hant": "ChineseTraditional.isl",
	"ja": "Japanese.isl", "ko": "Korean.isl", "de": "German.isl", "fr": "French.isl", "es": "Spanish.isl",
	"it": "Italian.isl", "pt": "Portuguese.isl", "pt-br": "BrazilianPortuguese.isl", "ru": "Russian.isl",
	"uk": "Ukrainian.isl", "pl": "Polish.isl", "nl": "Dutch.isl", "tr": "Turkish.isl", "cs": "Czech.isl",
	"he": "Hebrew.isl", "ar": "Arabic.isl", "da": "Danish.isl", "fi": "Finnish.isl", "no": "Norwegian.isl",
	"sv": "Swedish.isl", "hu": "Hungarian.isl", "sk": "Slovak.isl", "sl": "Slovenian.isl", "ca": "Catalan.isl",
	"hy": "Armenian.isl", "bg": "Bulgarian.isl", "is": "Icelandic.isl", "ta": "Tamil.isl",
}

// InnoLanguageNames lists the accepted language codes, sorted.
func InnoLanguageNames() []string {
	var out []string
	for k := range InnoLanguages {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
