package cli

import (
	"fmt"
	"strings"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/targets"
)

type initValues struct {
	Targets      []string
	Display      string
	Split        string
	Keystore     string
	Alias        string
	ExportMethod string
	Flavor       string
	OutDir       string
	DevIDs       []string // "Developer ID Application" identities found in the keychain (hint only)
	KeychainSeen bool     // the keychain was checked (running on macOS)
	InstallerIDs []string // "Developer ID Installer" identities (hint only)
	Proj         *project.Project
}

func yq(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#{}[],&*?|<>=!%@`'\"\\") || strings.TrimSpace(s) != s || isYAMLKeyword(s) {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
	}
	return s
}

func isYAMLKeyword(s string) bool {
	switch strings.ToLower(s) {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return true
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' && r != '-' && r != '+' {
			return false
		}
	}
	return true // numbers stay strings
}

// sectionPlatform maps top-level sections to the platform they need.
var sectionPlatform = map[string]host.Platform{"android": host.Android, "ios": host.IOS, "macos": host.MacOS, "windows": host.Windows, "linux": host.Linux, "web": host.Web}

// initDefaults returns active values (written uncommented) and detected
// values (used instead of the generic example in commented lines).
func initDefaults(v initValues) (active, detected map[string]string, hints map[string][]string) {
	p := v.Proj
	active, detected, hints = map[string]string{}, map[string]string{}, map[string][]string{}
	if v.Display != "" {
		active["app.display_name"] = yq(v.Display)
	}
	if len(v.Targets) > 0 {
		active["build.targets"] = "[" + strings.Join(v.Targets, ", ") + "]"
	}
	if v.OutDir != "" && v.OutDir != config.DefaultOutputDir {
		active["output.dir"] = yq(v.OutDir)
	}
	if p.Platforms[host.Android] && v.Split != "" {
		active["android.split_per_abi"] = v.Split
	}
	if v.Keystore != "" {
		active["android.signing.store_file"] = yq(v.Keystore)
		active["android.signing.store_password"] = "${FPACK_ANDROID_KEYSTORE_PASSWORD}"
		active["android.signing.key_alias"] = yq(orStr(v.Alias, "upload"))
		active["android.signing.key_password"] = "${FPACK_ANDROID_KEY_PASSWORD:-${FPACK_ANDROID_KEYSTORE_PASSWORD}}"
	}
	if v.ExportMethod != "" {
		active["ios.export_method"] = v.ExportMethod
	}
	if v.Flavor != "" {
		active["build.flavor"] = yq(v.Flavor)
	}
	set := func(k, val string) {
		if val != "" && !strings.Contains(val, "$") {
			detected[k] = yq(val)
		}
	}
	id := p.Identifier()
	set("app.name", p.Name)
	set("app.description", p.Description)
	set("app.identifier", id)
	set("build.build_name", p.Version)
	set("build.build_number", p.BuildNumber)
	if fl := p.AndroidFlavors; len(fl) > 0 {
		set("build.flavor", fl[0])
		hints["build.flavor"] = []string{i18n.S("Android flavors found: ", "检测到的 Android flavor：") + strings.Join(fl, ", ")}
	}
	set("app.publisher", p.WindowsCompany)
	if df := p.DefineFiles; len(df) > 0 {
		pick := df[0]
		for _, f := range df {
			if v.Flavor != "" && strings.Contains(f, v.Flavor) {
				pick = f
				break
			}
		}
		detected["build.dart_define_from_file"] = "[" + yq(pick) + "]"
		hints["build.dart_define_from_file"] = []string{i18n.S("files found: ", "检测到的文件：") + strings.Join(df, ", ")}
	}
	set("ios.team_id", p.IOSTeam)
	set("macos.pkg.identifier", orStr(p.MacBundleID, id))
	set("macos.pkg.min_os", p.MacDeploymentTarget)
	set("macos.pkg.title", v.Display)
	set("macos.dmg.volume_name", v.Display)
	if len(v.DevIDs) > 0 {
		set("macos.sign.identity", v.DevIDs[0])
		hints["macos.sign.identity"] = append([]string{i18n.S("Developer ID identities found in this Mac's keychain:", "本机钥匙串中的 Developer ID 证书：")}, v.DevIDs...)
	} else if v.KeychainSeen {
		hints["macos.sign.identity"] = []string{i18n.S("(no Developer ID Application certificate in this keychain; list them with: security find-identity -v -p codesigning)", "（本机钥匙串中没有 Developer ID Application 证书；查看：security find-identity -v -p codesigning）")}
	}
	if len(v.InstallerIDs) > 0 {
		set("macos.sign.installer_identity", v.InstallerIDs[0])
		hints["macos.sign.installer_identity"] = append([]string{i18n.S("installer identities in this keychain:", "本机钥匙串中的安装包证书：")}, v.InstallerIDs...)
	}
	if id != "" {
		set("windows.inno_setup.app_id", targets.StableGUID(id))
		set("windows.msix.identity_name", id)
	}
	set("windows.inno_setup.group_name", v.Display)
	set("windows.msix.display_name", v.Display)
	pkg := strings.ToLower(strings.ReplaceAll(p.Name, "_", "-"))
	set("linux.package_name", pkg)
	set("linux.startup_wm_class", p.LinuxBinary)
	if p.LauncherIcon != "" {
		set("linux.icon", p.LauncherIcon)
	}
	return active, detected, hints
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// renderInitYAML writes every fpack.yaml key, grouped by section, each with
// a comment (what it does, allowed values, default, example). Sections of
// platforms the project does not have are left out.
func renderInitYAML(v initValues) string {
	lang := "en"
	if i18n.IsZH() {
		lang = "zh"
	}
	S := i18n.S
	active, detected, hints := initDefaults(v)
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("# yaml-language-server: $schema=%s", config.SchemaURL)
	w("# fpack %s – %s", S("configuration", "配置文件"), S("https://matkurban.github.io/fpack/configuration", "https://matkurban.github.io/fpack/zh/configuration"))
	w("# %s", S("Precedence: command-line flags > FPACK_* environment variables > this file > defaults.", "优先级：命令行参数 > FPACK_* 环境变量 > 本文件 > 默认值。"))
	w("# %s", S("Every key is optional: uncomment what you need. ${VAR} / ${VAR:-default} read environment variables when the file is loaded.", "所有键都是可选的：需要哪个就取消注释。${VAR} / ${VAR:-默认值} 会在加载时读取环境变量。"))
	w("# %s", S("Relative paths are relative to the project root. fpack never edits your project files; it only reads them.", "相对路径均相对于项目根目录。fpack 只读取项目文件，绝不修改。"))

	var skipped []string
	shown := map[string]bool{}
	for _, sec := range config.Sections {
		top := strings.Split(sec.Path, ".")[0]
		if pl, ok := sectionPlatform[top]; ok && !v.Proj.Platforms[pl] {
			if !strings.Contains(sec.Path, ".") {
				skipped = append(skipped, top)
			}
			continue
		}
		depth := strings.Count(sec.Path, ".")
		ind := strings.Repeat("  ", depth)
		w("")
		if depth == 0 {
			w("# %s", strings.Repeat("-", 76))
		}
		w("%s# %s", ind, sec.Doc.Text(lang))
		w("%s%s:", ind, sec.Path[strings.LastIndex(sec.Path, ".")+1:])
		for _, k := range config.Keys {
			parent := ""
			if i := strings.LastIndex(k.Path, "."); i > 0 {
				parent = k.Path[:i]
			}
			if parent != sec.Path || shown[k.Path] {
				continue
			}
			shown[k.Path] = true
			writeKey(&b, k, lang, depth+1, active, detected, hints)
		}
	}
	if len(skipped) > 0 {
		w("")
		w("# %s %s", S("Not shown (the project has no folder for them):", "未列出（项目中没有对应平台目录）："), strings.Join(skipped, ", "))
		w("# %s", S("see https://matkurban.github.io/fpack/configuration or `fpack schema` for their keys.", "这些键见 https://matkurban.github.io/fpack/zh/configuration 或 `fpack schema`。"))
	}
	return b.String()
}

func writeKey(b *strings.Builder, k config.Key, lang string, depth int, active, detected map[string]string, hints map[string][]string) {
	S := i18n.S
	ind := strings.Repeat("  ", depth)
	name := k.Path[strings.LastIndex(k.Path, ".")+1:]
	fmt.Fprintf(b, "%s# %s\n", ind, k.Doc.Text(lang))
	var meta []string
	if len(k.Enum) > 0 {
		meta = append(meta, S("values: ", "可选值：")+strings.Join(k.Enum, " | "))
	}
	if k.Kind == config.KSplit {
		meta = append(meta, S("values: ", "可选值：")+"false | true | both")
	}
	if def := k.Default.Text(lang); def != "" {
		meta = append(meta, S("default: ", "默认：")+def)
	}
	val, isActive := active[k.Path]
	shownVal := val
	if !isActive {
		shownVal = k.Example
		if d, ok := detected[k.Path]; ok {
			shownVal = d
			meta = append(meta, S("detected", "检测到"))
		}
	}
	if shownVal != k.Example {
		meta = append(meta, S("example: ", "示例：")+k.Example)
	}
	if k.Env != "" {
		meta = append(meta, "env "+k.Env)
	}
	if k.Flag != "" {
		meta = append(meta, S("flag ", "参数 ")+k.Flag)
	}
	if k.Secret {
		meta = append(meta, S("keep it in an environment variable", "请放在环境变量中"))
	}
	if len(meta) > 0 {
		fmt.Fprintf(b, "%s# %s\n", ind, strings.Join(meta, S("; ", "；")))
	}
	for _, h := range hints[k.Path] {
		fmt.Fprintf(b, "%s#   %s\n", ind, h)
	}
	if isActive {
		fmt.Fprintf(b, "%s%s: %s\n", ind, name, val)
	} else {
		fmt.Fprintf(b, "%s# %s: %s\n", ind, name, shownVal)
	}
}
