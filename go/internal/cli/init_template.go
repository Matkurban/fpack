package cli

import (
	"fmt"
	"strings"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
)

type initValues struct {
	Targets      []string
	Display      string
	Split        string
	Keystore     string
	Alias        string
	ExportMethod string
	OutDir       string
	HasDMG       bool
	Proj         *project.Project
}

func yq(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#{}[],&*?|<>=!%@`'\"") || strings.TrimSpace(s) != s {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

func renderInitYAML(v initValues) string {
	S := i18n.S
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("# fpack configuration – %s", S("https://github.com/Matkurban/fpack#configuration", "https://github.com/Matkurban/fpack#配置参考"))
	w("# %s", S("Precedence: command-line flags > FPACK_* environment variables > this file > defaults.", "优先级：命令行参数 > FPACK_* 环境变量 > 本文件 > 默认值。"))
	w("# %s", S("Every key is optional. ${VAR} and ${VAR:-default} read environment variables.", "所有键都是可选的。${VAR} 与 ${VAR:-默认值} 会读取环境变量。"))
	w("# %s", S("fpack never edits your project files; it only reads them.", "fpack 只读取项目文件，绝不修改。"))
	w("")
	w("app:")
	w("  # %s", S("Base name of artifact files (default: pubspec name).", "产物文件名前缀（默认：pubspec 中的 name）。"))
	w("  # name: %s", v.Proj.Name)
	w("  display_name: %s", yq(v.Display))
	w("  # publisher: \"Your Company\"          # %s", S("Windows installer / Linux packages", "Windows 安装程序 / Linux 安装包"))
	w("  # identifier: %s", v.Proj.Identifier())
	w("")
	w("build:")
	w("  # %s", S("Targets built by a plain `fpack build`.", "直接运行 `fpack build` 时构建的目标。"))
	if len(v.Targets) > 0 {
		w("  targets: [%s]", strings.Join(v.Targets, ", "))
	} else {
		w("  # targets: [apk, aab]")
	}
	w("  mode: release                  # release | profile | debug")
	flv := "prod"
	if fl := v.Proj.AndroidFlavors; len(fl) > 0 {
		flv = fl[0]
		w("  # %s %s", S("Android flavors found:", "检测到的 Android flavor："), strings.Join(fl, ", "))
	}
	w("  # flavor: %s", flv)
	w("  # target: lib/main.dart        # %s", S("entry point (flutter -t)", "入口文件（flutter -t）"))
	w("  # dart_define:")
	w("  #   API_URL: https://api.example.com")
	w("  # dart_define_from_file: [env/prod.json]")
	w("  # obfuscate: true               # %s", S("symbols are kept in <output>/debug-info/", "符号文件保存在 <输出目录>/debug-info/"))
	w("  # extra_args: []                # %s", S("appended to every flutter build", "附加到每次 flutter build"))
	w("")
	w("output:")
	w("  dir: %s", yq(v.OutDir))
	w("  # %s {app} {version} {build} {platform} {arch} {variant} {mode} {flavor}; %s", S("placeholders:", "占位符："), S("{-x} adds '-' only when x is set", "{-x} 表示 x 非空时才加 '-'"))
	w("  # name: \"{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}\"")
	w("  # overwrite: false              # %s", S("true = replace existing files (same as --force)", "true = 覆盖已有文件（等同 --force）"))
	w("  # checksums: true               # %s", S("write SHA256SUMS", "生成 SHA256SUMS"))
	if v.Proj.Platforms["android"] {
		w("")
		w("android:")
		w("  split_per_abi: %s            # %s", v.Split, S("false = universal APK, true = one APK per ABI, both", "false = 通用 APK，true = 按 ABI 拆分，both = 两者都要"))
		w("  # abis: [arm64-v8a, armeabi-v7a, x86_64]")
		w("  # %s", S("Release signing, injected without touching Gradle files. Keep passwords in env vars!", "Release 签名，无需修改 Gradle 文件即可注入。密码请放在环境变量里！"))
		w("  # %s", S("CI: FPACK_ANDROID_KEYSTORE_BASE64 can hold the keystore itself.", "CI：可用 FPACK_ANDROID_KEYSTORE_BASE64 直接传入 keystore 内容。"))
		if v.Keystore != "" {
			w("  signing:")
			w("    store_file: %s", yq(v.Keystore))
			w("    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}")
			w("    key_alias: %s", yq(v.Alias))
			w("    key_password: ${FPACK_ANDROID_KEY_PASSWORD:-${FPACK_ANDROID_KEYSTORE_PASSWORD}}")
		} else {
			w("  # signing:")
			w("  #   store_file: ~/keys/upload-keystore.jks")
			w("  #   store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}")
			w("  #   key_alias: upload")
			w("  #   key_password: ${FPACK_ANDROID_KEY_PASSWORD}")
		}
	}
	if v.Proj.Platforms["ios"] {
		w("")
		w("ios:")
		w("  # app-store-connect | app-store | release-testing | ad-hoc | development | enterprise")
		if v.ExportMethod != "" {
			w("  export_method: %s", v.ExportMethod)
		} else {
			w("  # export_method: ad-hoc")
		}
		w("  # export_options_plist: ios/ExportOptions.plist   # %s", S("wins over export_method", "优先于 export_method"))
		w("  # codesign: false               # %s", S("unsigned IPA (same as --no-codesign)", "未签名 IPA（等同 --no-codesign）"))
	}
	if v.Proj.Platforms["macos"] {
		w("")
		w("macos:")
		if v.HasDMG {
			w("  # %s", S("Signing defaults come from the `dmg:` section of pubspec.yaml (read-only).", "签名默认沿用 pubspec.yaml 中 `dmg:` 段的设置（只读）。"))
			w("  # %s", S("Uncomment to override, e.g. notarize: false for quick local builds (or --no-notarize).", "取消注释即可覆盖，例如本地快速构建时 notarize: false（或用 --no-notarize）。"))
		} else {
			w("  # %s", S("Developer ID signing + notarization for distribution outside the App Store.", "用于 App Store 以外分发的 Developer ID 签名 + 公证。"))
		}
		w("  # sign:")
		w("  #   enabled: true")
		w("  #   identity: \"Developer ID Application: Your Name (TEAMID)\"")
		w("  #   notarize: true")
		w("  #   notary_profile: NotaryProfile    # xcrun notarytool store-credentials NotaryProfile ...")
		w("  # dmg:")
		w("  #   tool: auto                   # auto | hdiutil | create-dmg")
		w("  #   volume_name: %s", yq(v.Display))
	}
	if v.Proj.Platforms["windows"] {
		w("")
		w("# windows:")
		w("#   inno_setup:")
		w("#     app_id: \"\"                  # %s", S("stable GUID; default is derived from the app identifier", "固定 GUID；默认根据应用标识生成"))
		w("#     script: windows/installer.iss # %s", S("optional custom script", "可选的自定义脚本"))
	}
	if v.Proj.Platforms["linux"] {
		w("")
		w("# linux:")
		w("#   package_name: %s", strings.ToLower(strings.ReplaceAll(v.Proj.Name, "_", "-")))
		w("#   icon: assets/icon.png        # %s", S("PNG used for .deb/.rpm/.AppImage", ".deb/.rpm/.AppImage 使用的 PNG 图标"))
		w("#   categories: \"Network;Chat;\"")
		w("#   deb:")
		w("#     depends: [\"libgtk-3-0 | libgtk-3-0t64\"]")
	}
	if v.Proj.Platforms["web"] {
		w("")
		w("# web:")
		w("#   base_href: /")
		w("#   wasm: false")
	}
	return b.String()
}
