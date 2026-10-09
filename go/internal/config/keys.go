package config

// The key registry describes every fpack.yaml key once: type, default,
// affected targets, environment variable, CLI flag and a zh/en explanation.
// `fpack init`, the JSON schema, doc/configuration.md, FPACK_* handling and
// path validation are all generated from it (keys_test.go checks that it
// covers exactly the fields of Config).

// T is a zh/en text pair.
type T struct{ EN, ZH string }

// Text returns the text in the given language ("zh" or "en").
func (t T) Text(lang string) string {
	if lang == "zh" && t.ZH != "" {
		return t.ZH
	}
	return t.EN
}

// Kind is the value type of a key.
type Kind string

// Key kinds.
const (
	KString  Kind = "string"
	KPath    Kind = "path" // a file relative to the project root
	KURL     Kind = "url"
	KBool    Kind = "bool"
	KInt     Kind = "int"
	KEnum    Kind = "enum"
	KList    Kind = "list"     // list of strings (a single string is accepted)
	KIntList Kind = "int-list" // list of integers
	KMap     Kind = "map"      // string → string
	KAnyMap  Kind = "any-map"  // string → any scalar
	KListMap Kind = "list-map" // target → list of commands
	KPair    Kind = "pair"     // [x, y]
	KScalar  Kind = "scalar"   // string or number
	KDefines Kind = "defines"  // map or list of KEY=VALUE
	KSplit   Kind = "split"    // false | true | both
)

// Key describes one fpack.yaml key.
type Key struct {
	Path    string
	Kind    Kind
	Enum    []string
	Default T      // human readable default ("" = none / empty)
	Targets string // affected targets ("all", "apk, aab", …)
	Example string // YAML value used as example / placeholder
	Doc     T
	Env     string // FPACK_* variable that overrides it
	Flag    string // command line flag that overrides it
	Secret  bool   // never print the value; prefer ${ENV}
}

// Section describes a top-level or nested section for headings.
type Section struct {
	Path string
	Doc  T
}

func k(path string, kind Kind, def T, targets, example string, doc T, opts ...func(*Key)) Key {
	key := Key{Path: path, Kind: kind, Default: def, Targets: targets, Example: example, Doc: doc}
	for _, o := range opts {
		o(&key)
	}
	return key
}

func enum(v ...string) func(*Key) { return func(k *Key) { k.Kind = KEnum; k.Enum = v } }
func env(v string) func(*Key)     { return func(k *Key) { k.Env = v } }
func flag(v string) func(*Key)    { return func(k *Key) { k.Flag = v } }
func secret() func(*Key)          { return func(k *Key) { k.Secret = true } }

var none = T{}

func d(en, zh string) T { return T{en, zh} }

// Sections in display order (headings in docs and fpack init).
var Sections = []Section{
	{"app", d("Application metadata used by installers and packages.", "应用信息：用于安装程序、软件包元数据。")},
	{"flutter", d("Flutter SDK selection.", "Flutter SDK 选择。")},
	{"build", d("Options for every flutter build.", "所有 flutter build 共用的选项。")},
	{"output", d("Where artifacts go and how they are named.", "产物输出目录与命名。")},
	{"hooks", d("Shell commands run around the build (working directory: project root).", "构建前后执行的 shell 命令（工作目录：项目根目录）。")},
	{"android", d("Android: apk, aab.", "Android：apk、aab。")},
	{"android.signing", d("Release signing, injected without editing Gradle files. Keep passwords in environment variables.", "发布签名：通过注入方式生效，不修改 Gradle 文件。密码请放在环境变量中。")},
	{"ios", d("iOS: ipa. Setting any of team_id … export_options makes fpack generate ExportOptions.plist.", "iOS：ipa。设置 team_id … export_options 中任意一项时，fpack 会自动生成 ExportOptions.plist。")},
	{"macos", d("macOS: macos (.app zip), dmg, pkg.", "macOS：macos（.app zip）、dmg、pkg。")},
	{"macos.sign", d("Developer ID signing and notarization (only from this file, FPACK_MACOS_* and flags).", "Developer ID 签名与公证（只来自本文件、FPACK_MACOS_* 和命令行参数）。")},
	{"macos.notarize", d("How fpack waits for Apple's notary service. Every submission is recorded in <output>/NOTARIZATION.md and notarization.json with copy-paste commands.", "如何等待 Apple 公证服务。每次提交都会记录到 <输出目录>/NOTARIZATION.md 和 notarization.json，其中有可直接复制的查询命令。")},
	{"macos.dmg", d("Disk image layout. Window/icon layout needs create-dmg (brew install create-dmg).", "DMG 磁盘镜像。窗口/图标布局需要 create-dmg（brew install create-dmg）。")},
	{"macos.pkg", d("Installer package (pkgbuild + productbuild).", "安装包（pkgbuild + productbuild）。")},
	{"windows", d("Windows: windows (zip), exe (Inno Setup), msix.", "Windows：windows（zip）、exe（Inno Setup）、msix。")},
	{"windows.inno_setup", d("Inno Setup installer (.exe).", "Inno Setup 安装程序（.exe）。")},
	{"windows.sign", d("Authenticode signing with signtool: the app .exe and the installer.", "使用 signtool 进行 Authenticode 签名：应用 .exe 与安装程序。")},
	{"windows.msix", d("MSIX package (needs the msix dev_dependency). These keys override pubspec msix_config.", "MSIX 包（需要 msix 开发依赖）。这些键会覆盖 pubspec 中的 msix_config。")},
	{"linux", d("Linux: linux (tar.gz), deb, rpm, appimage.", "Linux：linux（tar.gz）、deb、rpm、appimage。")},
	{"linux.deb", d("Debian/Ubuntu package.", "Debian/Ubuntu 软件包。")},
	{"linux.rpm", d("Fedora/RHEL/openSUSE package.", "Fedora/RHEL/openSUSE 软件包。")},
	{"linux.appimage", d("AppImage.", "AppImage。")},
	{"web", d("Web: web (zip).", "Web：web（zip）。")},
}

const (
	native  = "apk, aab, ipa, macos, dmg, pkg, windows, exe, msix, linux, deb, rpm, appimage"
	linuxP  = "deb, rpm, appimage"
	macAll  = "macos, dmg, pkg"
	winAll  = "windows, exe, msix"
	signApp = "macos, dmg, pkg"
)

// Keys lists every fpack.yaml key in display order.
var Keys = []Key{
	// ---- app ----
	k("app.name", KString, d("pubspec name", "pubspec 的 name"), "all", "xue_hua_im",
		d("Base name of artifact files ({app} in output.name).", "产物文件名前缀（output.name 中的 {app}）。")),
	k("app.display_name", KString, d("macOS PRODUCT_NAME, else pubspec name", "macOS 的 PRODUCT_NAME，否则为 pubspec 的 name"), "exe, msix, pkg, deb, rpm, appimage, linux", "雪花IM",
		d("Human-readable app name: installer title, Start menu, .desktop Name=.", "给人看的应用名：安装程序标题、开始菜单、.desktop 的 Name=。")),
	k("app.description", KString, d("pubspec description", "pubspec 的 description"), "deb, rpm, appimage, msix", "A fast and secure messenger",
		d("Short description: deb Description, rpm Summary, .desktop Comment=, msix description.", "简短描述：deb 的 Description、rpm 的 Summary、.desktop 的 Comment=、msix 描述。")),
	k("app.publisher", KString, d("CompanyName in windows/runner/Runner.rc", "windows/runner/Runner.rc 中的 CompanyName"), "exe, msix, deb, rpm", "XueHua Tech",
		d("Company / author: Windows installer publisher, msix publisher display name, deb Maintainer fallback, rpm Vendor.", "公司 / 作者：Windows 安装程序发布者、msix 发布者显示名、deb Maintainer 的后备值、rpm Vendor。")),
	k("app.identifier", KString, d("Linux APPLICATION_ID, Android applicationId or iOS bundle id", "Linux APPLICATION_ID、Android applicationId 或 iOS bundle id"), "exe, msix, pkg, appimage", "com.xuehua.im",
		d("Reverse-DNS app id: Inno Setup AppId seed, msix identity name, pkg identifier fallback.", "反向域名格式的应用 ID：Inno Setup AppId 的种子、msix identity name、pkg identifier 的后备值。")),
	k("app.homepage", KURL, none, "exe, deb, rpm", "https://xuehua.example.com",
		d("Website: Inno Setup publisher URL, deb Homepage, rpm URL.", "官网：Inno Setup 发布者网址、deb 的 Homepage、rpm 的 URL。")),
	k("app.support_url", KURL, d("app.homepage", "app.homepage"), "exe", "https://xuehua.example.com/support",
		d("Support link (Windows \"Apps & features\").", "技术支持链接（Windows“应用和功能”中显示）。")),
	k("app.maintainer", KString, d("app.publisher", "app.publisher"), "deb, rpm", "XueHua Team <dev@xuehua.example.com>",
		d("\"Name <email>\" for deb Maintainer and rpm Packager.", "deb Maintainer 与 rpm Packager 字段，格式 “名字 <邮箱>”。")),
	k("app.copyright", KString, d("© <year> <publisher>", "© <年份> <发布者>"), "exe, deb, rpm", "© 2026 XueHua Tech",
		d("Copyright line: Inno Setup AppCopyright and version info, deb copyright file.", "版权信息：Inno Setup 的 AppCopyright 与版本信息、deb 的 copyright 文件。")),
	k("app.license", KString, d("Proprietary", "Proprietary"), "rpm, deb", "MIT",
		d("License (SPDX id): rpm License, deb copyright file.", "许可证（SPDX 标识）：rpm 的 License、deb 的 copyright 文件。")),

	// ---- flutter ----
	k("flutter.sdk", KPath, d("FLUTTER_ROOT, fvm, PATH", "FLUTTER_ROOT、fvm、PATH 中的 flutter"), "all", "~/fvm/versions/stable",
		d("Flutter SDK root to use.", "使用的 Flutter SDK 根目录。"), env("FPACK_FLUTTER"), flag("--flutter")),

	// ---- build ----
	k("build.targets", KList, none, "—", "[apk, aab, ipa, dmg]",
		d("Targets built by `fpack build` without arguments. Aliases accepted by the command line (e.g. `bundle`, `ios`, `setup`) work here too.", "执行 `fpack build` 且不带目标时构建的目标。命令行接受的别名（如 `bundle`、`ios`、`setup`）这里同样可用。")),
	k("build.mode", KEnum, d("release", "release"), "all", "release",
		d("Build mode.", "构建模式。"), enum("release", "profile", "debug"), env("FPACK_MODE"), flag("--mode")),
	k("build.flavor", KString, none, "apk, aab, ipa, macos, dmg, pkg", "prod",
		d("Flavor / Xcode scheme (--flavor); appears in file names as {flavor}.", "flavor / Xcode scheme（--flavor）；文件名中的 {flavor}。"), env("FPACK_FLAVOR"), flag("--flavor")),
	k("build.target", KPath, d("lib/main.dart", "lib/main.dart"), "all", "lib/main_prod.dart",
		d("Entry point (flutter -t).", "入口文件（flutter -t）。"), env("FPACK_ENTRY"), flag("-t, --target")),
	k("build.dart_define", KDefines, none, "all", "{API_URL: https://api.example.com}",
		d("--dart-define values (map or list of KEY=VALUE).", "--dart-define 值（map 或 KEY=VALUE 列表）。"), flag("--dart-define")),
	k("build.dart_define_from_file", KList, none, "all", "[config/prod.json]",
		d("--dart-define-from-file files (.json or .env).", "--dart-define-from-file 文件（.json 或 .env）。"), flag("--dart-define-from-file")),
	k("build.build_name", KScalar, d("pubspec version before +", "pubspec 版本号 + 之前的部分"), "all", "1.2.0",
		d("Version name ({version}).", "版本名（{version}）。"), env("FPACK_BUILD_NAME"), flag("--build-name")),
	k("build.build_number", KScalar, d("pubspec version after +", "pubspec 版本号 + 之后的部分"), "all", "42",
		d("Build number ({build}).", "构建号（{build}）。"), env("FPACK_BUILD_NUMBER"), flag("--build-number")),
	k("build.obfuscate", KBool, d("false", "false"), native, "true",
		d("Obfuscate Dart code; symbols go to split_debug_info (default <output>/debug-info/<platform>).", "混淆 Dart 代码；符号文件保存到 split_debug_info（默认 <输出目录>/debug-info/<平台>）。"), env("FPACK_OBFUSCATE"), flag("--obfuscate")),
	k("build.split_debug_info", KString, d("<output>/debug-info/<platform> when obfuscating", "混淆时为 <输出目录>/debug-info/<平台>"), native, "build/symbols",
		d("Directory for Dart debug symbols (--split-debug-info).", "Dart 调试符号目录（--split-debug-info）。"), flag("--split-debug-info")),
	k("build.tree_shake_icons", KBool, d("true", "true"), "all", "false",
		d("false = --no-tree-shake-icons (keep all icon font glyphs).", "false 时传 --no-tree-shake-icons（保留全部图标字体字形）。")),
	k("build.extra_args", KList, none, "all", "[--no-pub]",
		d("Extra arguments for every flutter build (also: everything after -- on the command line).", "追加到每个 flutter build 的参数（命令行 -- 之后的参数同理）。")),

	// ---- output ----
	k("output.dir", KString, d("dist/{version}{+build}", "dist/{version}{+build}"), "all", "\"dist/{version}{+build}\"",
		d("Artifact directory (relative to the project; placeholders allowed).", "产物目录（相对于项目，可用占位符）。"), env("FPACK_OUTPUT_DIR"), flag("-o, --output")),
	k("output.name", KString, d("{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}", "{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}"), "all", "\"{app}-{version}-{platform}{-arch}\"",
		d("File name template without extension. Placeholders: {app} {version} {build} {platform} {arch} {variant} {mode} {flavor} {target} {date}; {-x}/{+x} add the separator only when x is set.", "文件名模板（不含扩展名）。占位符：{app} {version} {build} {platform} {arch} {variant} {mode} {flavor} {target} {date}；{-x}/{+x} 表示 x 非空时才加分隔符。")),
	k("output.names", KMap, none, "all", "{exe: \"{app}-setup-{version}\", web: \"{app}-web\"}",
		d("Per-target file name templates (target → template), overriding output.name.", "按目标单独设置文件名模板（目标 → 模板），优先于 output.name。")),
	k("output.overwrite", KBool, d("false", "false"), "all", "true",
		d("Replace existing artifacts instead of stopping.", "已存在同名产物时覆盖，而不是停止。"), env("FPACK_OVERWRITE"), flag("-f, --force")),
	k("output.checksums", KBool, d("true", "true"), "all", "false",
		d("Write a checksum file next to the artifacts.", "在产物旁写入校验和文件。")),
	k("output.checksum_algorithm", KEnum, d("sha256", "sha256"), "all", "sha512",
		d("Checksum algorithm (file SHA256SUMS or SHA512SUMS).", "校验算法（文件名 SHA256SUMS 或 SHA512SUMS）。"), enum("sha256", "sha512")),

	// ---- hooks ----
	k("hooks.pre_build", KList, none, "all", "[dart run build_runner build --delete-conflicting-outputs]",
		d("Commands run once before the first flutter build; a failure stops the build.", "第一次 flutter build 之前执行一次；失败则停止构建。")),
	k("hooks.post_build", KList, none, "all", "[./scripts/upload.sh]",
		d("Commands run once after all targets (also after build failures, but not when every target already failed its checks); FPACK_ARTIFACTS lists the produced files (one per line), FPACK_SUCCESS is 1/0.", "全部目标完成后执行一次（构建失败后也会执行，但所有目标都未通过构建前检查时跳过）；FPACK_ARTIFACTS 为产物列表（每行一个），FPACK_SUCCESS 为 1/0。")),
	k("hooks.pre_package", KListMap, none, "all", "{apk: [./scripts/check_size.sh]}",
		d("Per target (target → commands), before its packaging steps; FPACK_TARGET is set.", "按目标（目标 → 命令）在打包步骤之前执行；提供 FPACK_TARGET。")),
	k("hooks.post_package", KListMap, none, "all", "{dmg: [./scripts/upload_dmg.sh \"$FPACK_ARTIFACT\"]}",
		d("Per target, after its artifacts exist; FPACK_ARTIFACT (first file) and FPACK_ARTIFACTS are set.", "按目标在产物生成后执行；提供 FPACK_ARTIFACT（第一个产物）和 FPACK_ARTIFACTS。")),

	// ---- android ----
	k("android.split_per_abi", KSplit, d("false", "false"), "apk", "both",
		d("false = one universal APK, true = one APK per ABI, both = universal + per-ABI.", "false：一个通用 APK；true：每个 ABI 一个 APK；both：两者都要。"), env("FPACK_SPLIT_PER_ABI"), flag("--split-per-abi")),
	k("android.abis", KList, d("armeabi-v7a, arm64-v8a, x86_64", "armeabi-v7a、arm64-v8a、x86_64"), "apk, aab", "[arm64-v8a, armeabi-v7a]",
		d("Target ABIs (--target-platform).", "目标 ABI（--target-platform）。"), flag("--abis")),
	k("android.signing.store_file", KPath, d("none (android/key.properties of the project is used)", "无（使用项目自己的 android/key.properties）"), "apk, aab", "~/keys/upload.jks",
		d("Keystore file (.jks/.keystore). FPACK_ANDROID_KEYSTORE_BASE64 can provide it in CI.", "keystore 文件（.jks/.keystore）。CI 中可用 FPACK_ANDROID_KEYSTORE_BASE64 提供。"), env("FPACK_ANDROID_KEYSTORE")),
	k("android.signing.store_password", KString, none, "apk, aab", "${KEYSTORE_PASSWORD}",
		d("Keystore password.", "keystore 密码。"), env("FPACK_ANDROID_KEYSTORE_PASSWORD"), secret()),
	k("android.signing.key_alias", KString, none, "apk, aab", "upload",
		d("Key alias.", "key 别名。"), env("FPACK_ANDROID_KEY_ALIAS")),
	k("android.signing.key_password", KString, d("store_password", "store_password"), "apk, aab", "${KEY_PASSWORD}",
		d("Key password.", "key 密码。"), env("FPACK_ANDROID_KEY_PASSWORD"), secret()),
	k("android.signing.v1", KBool, d("apksigner default (on when minSdk < 24)", "apksigner 默认（minSdk < 24 时开启）"), "apk", "true",
		d("APK signature scheme v1 (JAR signing, needed below Android 7). Setting any of v1-v4 makes fpack re-sign the APKs with apksigner using android.signing.", "APK v1 签名（JAR 签名，Android 7 以下需要）。设置 v1-v4 任意一项时，fpack 会用 apksigner 和 android.signing 重新签名 APK。")),
	k("android.signing.v2", KBool, d("true", "true"), "apk", "true",
		d("APK signature scheme v2 (Android 7+).", "APK v2 签名（Android 7+）。")),
	k("android.signing.v3", KBool, d("true", "true"), "apk", "true",
		d("APK signature scheme v3 (Android 9+, key rotation).", "APK v3 签名（Android 9+，支持密钥轮换）。")),
	k("android.signing.v4", KBool, d("false", "false"), "apk", "true",
		d("APK signature scheme v4 (incremental install, Android 11+); writes <apk>.idsig next to the APK.", "APK v4 签名（增量安装，Android 11+）；会在 APK 旁生成 <apk>.idsig。")),
	k("android.project_args", KMap, none, "apk, aab", "{minify: \"true\"}",
		d("Gradle project properties (flutter -P key=value), readable in build.gradle with project.findProperty (e.g. to toggle minify/R8).", "Gradle 项目属性（flutter -P key=value），build.gradle 中可用 project.findProperty 读取（例如开关 minify/R8）。")),
	k("android.extra_args", KList, none, "apk, aab", "[--android-skip-build-dependency-validation]",
		d("Extra arguments for flutter build apk/appbundle.", "追加到 flutter build apk/appbundle 的参数。")),

	// ---- ios ----
	k("ios.export_method", KEnum, d("app-store-connect", "app-store-connect"), "ipa", "ad-hoc",
		d("IPA export method.", "IPA 导出方式。"), enum("app-store-connect", "app-store", "release-testing", "ad-hoc", "development", "debugging", "enterprise"), env("FPACK_IOS_EXPORT_METHOD"), flag("--export-method")),
	k("ios.export_options_plist", KPath, none, "ipa", "ios/ExportOptions.plist",
		d("Your own ExportOptions.plist; wins over every generated option below.", "自己的 ExportOptions.plist；优先于下面所有生成选项。"), env("FPACK_IOS_EXPORT_OPTIONS_PLIST"), flag("--export-options-plist")),
	k("ios.codesign", KBool, d("true", "true"), "ipa", "false",
		d("false = unsigned IPA (Payload/ zip) for re-signing later.", "false：构建未签名 IPA（Payload/ 结构），用于之后重签名。"), env("FPACK_IOS_CODESIGN"), flag("--no-codesign")),
	k("ios.team_id", KString, d("DEVELOPMENT_TEAM of the Xcode project", "Xcode 工程中的 DEVELOPMENT_TEAM"), "ipa", "ABCDE12345",
		d("Apple team ID (teamID).", "Apple 团队 ID（teamID）。")),
	k("ios.signing_style", KEnum, d("automatic", "automatic"), "ipa", "manual",
		d("signingStyle: automatic or manual (manual needs provisioning_profiles).", "signingStyle：automatic 或 manual（manual 需要 provisioning_profiles）。"), enum("automatic", "manual")),
	k("ios.signing_certificate", KString, none, "ipa", "Apple Distribution",
		d("signingCertificate (manual signing), e.g. \"Apple Distribution\".", "signingCertificate（手动签名），例如 \"Apple Distribution\"。")),
	k("ios.provisioning_profiles", KMap, none, "ipa", "{com.xuehua.im: XueHua AdHoc}",
		d("provisioningProfiles: bundle id → profile name or UUID (include extensions).", "provisioningProfiles：bundle id → 描述文件名称或 UUID（扩展也要列出）。")),
	k("ios.upload_symbols", KBool, d("true", "true"), "ipa", "false",
		d("uploadSymbols for App Store Connect.", "uploadSymbols：是否上传符号表到 App Store Connect。")),
	k("ios.manage_app_version_and_build_number", KBool, d("true", "true"), "ipa", "false",
		d("manageAppVersionAndBuildNumber: let App Store Connect bump the build number.", "manageAppVersionAndBuildNumber：是否由 App Store Connect 自动管理版本号/构建号。")),
	k("ios.destination", KEnum, d("export", "export"), "ipa", "upload",
		d("export = write the IPA locally, upload = send it to App Store Connect (ExportOptions destination).", "export 为本地导出 IPA，upload 为直接上传到 App Store Connect（ExportOptions 的 destination）。"), enum("export", "upload")),
	k("ios.thinning", KString, d("<none>", "<none>"), "ipa", "<thin-for-all-variants>",
		d("thinning for ad-hoc/development/enterprise exports.", "ad-hoc/development/enterprise 导出时的瘦身（thinning）选项。")),
	k("ios.strip_swift_symbols", KBool, d("true", "true"), "ipa", "false",
		d("stripSwiftSymbols.", "stripSwiftSymbols。")),
	k("ios.export_options", KAnyMap, none, "ipa", "{iCloudContainerEnvironment: Production}",
		d("Any other ExportOptions.plist keys (added as-is).", "其他任意 ExportOptions.plist 键（原样写入）。")),
	k("ios.extra_args", KList, none, "ipa", "[--no-tree-shake-icons]",
		d("Extra arguments for flutter build ipa.", "追加到 flutter build ipa 的参数。")),

	// ---- macos.sign ----
	k("macos.sign.enabled", KBool, d("true when identity is set", "设置了 identity 时为 true"), signApp, "true",
		d("Re-sign the .app with Developer ID (and sign the DMG). true without identity picks the first \"Developer ID Application\" identity. false (--no-sign) also disables pkg signing and notarization.", "用 Developer ID 重新签名 .app（并签名 DMG）。为 true 但未设置 identity 时自动选用第一个 “Developer ID Application” 证书。false（--no-sign）同时关闭 pkg 签名与公证。"), env("FPACK_MACOS_SIGN"), flag("--sign / --no-sign")),
	k("macos.sign.identity", KString, none, signApp, "\"Developer ID Application: Your Name (TEAMID)\"",
		d("codesign identity for the app; setting it turns signing on.", "App 的 codesign 证书；设置后即启用签名。"), env("FPACK_MACOS_SIGN_IDENTITY"), flag("--sign-identity")),
	k("macos.sign.entitlements", KPath, d("macos/Runner/Release.entitlements", "macos/Runner/Release.entitlements"), signApp, "macos/Runner/Release.entitlements",
		d("Entitlements used when re-signing the app.", "重新签名 App 时使用的 entitlements。")),
	k("macos.sign.hardened_runtime", KBool, d("true", "true"), signApp, "true",
		d("Sign with the hardened runtime (required for notarization).", "使用 Hardened Runtime 签名（公证必需）。")),
	k("macos.sign.notarize", KBool, d("true when a notary credential is set", "设置了公证凭证时为 true"), signApp, "true",
		d("Notarize and staple the zip, DMG and signed pkg (uploads to Apple).", "公证并装订 zip、DMG 与已签名的 pkg（会上传到 Apple）。"), env("FPACK_MACOS_NOTARIZE"), flag("--notarize / --no-notarize")),
	k("macos.sign.notary_profile", KString, none, signApp, "NotaryProfile",
		d("Keychain profile from `xcrun notarytool store-credentials <name>` (recommended locally).", "`xcrun notarytool store-credentials <名字>` 创建的钥匙串配置名（本地推荐）。"), env("FPACK_MACOS_NOTARY_PROFILE"), flag("--notary-profile")),
	k("macos.sign.notary_apple_id", KString, none, signApp, "dev@example.com",
		d("Apple ID for notarization (with notary_team_id + notary_password), instead of a profile.", "公证用 Apple ID（配合 notary_team_id + notary_password），替代钥匙串配置。"), env("FPACK_NOTARY_APPLE_ID")),
	k("macos.sign.notary_team_id", KString, none, signApp, "ABCDE12345",
		d("Team ID for Apple ID notarization.", "Apple ID 公证时的团队 ID。"), env("FPACK_NOTARY_TEAM_ID")),
	k("macos.sign.notary_password", KString, none, signApp, "${NOTARY_PASSWORD}",
		d("App-specific password for Apple ID notarization.", "Apple ID 公证用的 App 专用密码。"), env("FPACK_NOTARY_PASSWORD"), secret()),
	k("macos.sign.notary_api_key", KPath, none, signApp, "~/keys/AuthKey_ABC123.p8",
		d("App Store Connect API key (.p8) for notarization (CI friendly).", "公证用 App Store Connect API 密钥（.p8，适合 CI）。"), env("FPACK_NOTARY_API_KEY")),
	k("macos.sign.notary_api_key_id", KString, none, signApp, "ABC123DEF4",
		d("API key ID.", "API 密钥 ID。"), env("FPACK_NOTARY_API_KEY_ID")),
	k("macos.sign.notary_api_issuer", KString, none, signApp, "69a6de7e-…",
		d("API issuer UUID (omit for individual keys).", "API Issuer UUID（个人密钥可省略）。"), env("FPACK_NOTARY_API_ISSUER")),
	k("macos.sign.installer_identity", KString, none, "pkg", "\"Developer ID Installer: Your Name (TEAMID)\"",
		d("Signs the .pkg; a separate certificate from the app's \"Developer ID Application\". Unset = unsigned pkg.", "签名 .pkg 的证书，与 App 的 “Developer ID Application” 不同。不设置则 pkg 不签名。"), env("FPACK_MACOS_INSTALLER_IDENTITY"), flag("--installer-identity")),

	// ---- macos.notarize ----
	k("macos.notarize.wait", KBool, d("true", "true"), signApp, "false",
		d("true: stay attached until Apple answers (shows elapsed time; Ctrl-C stops waiting, the submission continues at Apple). false: submit, write NOTARIZATION.md / notarization.json and finish with status \"submitted\"; later run `fpack notarize finish` to staple.",
			"true：一直等待 Apple 返回结果（显示已等待时间；按 Ctrl-C 只是停止等待，Apple 端会继续处理）。false：提交后写入 NOTARIZATION.md / notarization.json 并结束，状态为 “submitted”；之后运行 `fpack notarize finish` 装订。"),
		env("FPACK_NOTARIZE_WAIT"), flag("--notarize-no-wait")),

	// ---- macos.dmg ----
	k("macos.dmg.tool", KEnum, d("auto", "auto"), "dmg", "create-dmg",
		d("auto = create-dmg when installed (or when layout keys are set), else hdiutil.", "auto：装了 create-dmg（或设置了布局键）时用 create-dmg，否则用 hdiutil。"), enum("auto", "hdiutil", "create-dmg"), env("FPACK_DMG_TOOL"), flag("--dmg-tool")),
	k("macos.dmg.volume_name", KString, d(".app name", ".app 名称"), "dmg", "雪花IM",
		d("Volume name shown when the DMG is mounted.", "挂载 DMG 后显示的卷名。")),
	k("macos.dmg.volume_icon", KPath, none, "dmg", "macos/dmg/volume.icns",
		d("Volume icon (.icns). create-dmg.", "卷图标（.icns）。需要 create-dmg。")),
	k("macos.dmg.background", KPath, none, "dmg", "macos/dmg/background.png",
		d("Window background image. create-dmg.", "窗口背景图。需要 create-dmg。")),
	k("macos.dmg.window_position", KPair, d("[200, 120]", "[200, 120]"), "dmg", "[200, 120]",
		d("Window position [x, y]. create-dmg.", "窗口位置 [x, y]。需要 create-dmg。")),
	k("macos.dmg.window_size", KPair, d("[660, 400]", "[660, 400]"), "dmg", "[660, 400]",
		d("Window size [width, height]. create-dmg.", "窗口大小 [宽, 高]。需要 create-dmg。")),
	k("macos.dmg.icon_size", KInt, d("128", "128"), "dmg", "128",
		d("Icon size in the window. create-dmg.", "窗口中的图标大小。需要 create-dmg。")),
	k("macos.dmg.app_position", KPair, d("[180, 190]", "[180, 190]"), "dmg", "[180, 190]",
		d("Position of the app icon [x, y]. create-dmg.", "App 图标位置 [x, y]。需要 create-dmg。")),
	k("macos.dmg.applications_position", KPair, d("[480, 190]", "[480, 190]"), "dmg", "[480, 190]",
		d("Position of the Applications link [x, y]. create-dmg.", "“应用程序”快捷方式位置 [x, y]。需要 create-dmg。")),
	k("macos.dmg.format", KEnum, d("UDZO", "UDZO"), "dmg", "ULFO",
		d("Image format: UDZO (zlib), UDBZ (bzip2), ULFO (lzfse, macOS 10.11+), ULMO (lzma, 10.15+), UDRO (read-only, uncompressed).", "镜像格式：UDZO（zlib）、UDBZ（bzip2）、ULFO（lzfse，macOS 10.11+）、ULMO（lzma，10.15+）、UDRO（只读不压缩）。"), enum("UDZO", "UDBZ", "ULFO", "ULMO", "UDRO")),
	k("macos.dmg.filesystem", KEnum, d("HFS+", "HFS+"), "dmg", "APFS",
		d("Filesystem of the image.", "镜像文件系统。"), enum("HFS+", "APFS")),
	k("macos.dmg.license", KPath, none, "dmg", "LICENSE.txt",
		d("License agreement shown when the DMG is opened (.txt/.rtf). create-dmg.", "打开 DMG 时显示的许可协议（.txt/.rtf）。需要 create-dmg。")),

	// ---- macos.pkg ----
	k("macos.pkg.identifier", KString, d("macOS bundle id", "macOS 工程的 bundle id"), "pkg", "com.xuehua.im",
		d("Package identifier (pkgutil --pkgs).", "安装包标识（pkgutil --pkgs 中显示）。")),
	k("macos.pkg.version", KString, d("build name", "版本名"), "pkg", "1.2.0",
		d("Package version.", "安装包版本。")),
	k("macos.pkg.install_location", KString, d("/Applications", "/Applications"), "pkg", "/Applications",
		d("Absolute directory the app is installed into.", "App 安装到的绝对路径目录。")),
	k("macos.pkg.title", KString, d(".app name", ".app 名称"), "pkg", "雪花IM",
		d("Installer window title.", "安装器窗口标题。")),
	k("macos.pkg.welcome", KPath, none, "pkg", "macos/installer/welcome.html",
		d("Welcome page (.html/.rtf/.txt).", "欢迎页（.html/.rtf/.txt）。")),
	k("macos.pkg.readme", KPath, none, "pkg", "macos/installer/readme.html",
		d("Read-me page (.html/.rtf/.txt).", "“请先阅读”页（.html/.rtf/.txt）。")),
	k("macos.pkg.license", KPath, none, "pkg", "macos/installer/license.rtf",
		d("License page the user must accept (.html/.rtf/.txt).", "用户必须同意的许可页（.html/.rtf/.txt）。")),
	k("macos.pkg.conclusion", KPath, none, "pkg", "macos/installer/done.html",
		d("Conclusion page (.html/.rtf/.txt).", "完成页（.html/.rtf/.txt）。")),
	k("macos.pkg.background", KPath, none, "pkg", "macos/installer/background.png",
		d("Background image (light and dark mode).", "背景图（浅色/深色模式都使用）。")),
	k("macos.pkg.min_os", KString, d("MACOSX_DEPLOYMENT_TARGET of the project", "工程的 MACOSX_DEPLOYMENT_TARGET"), "pkg", "10.15",
		d("Minimum macOS version; Installer refuses older systems.", "最低 macOS 版本；低于此版本时安装器会拒绝安装。")),
	k("macos.pkg.preinstall", KPath, none, "pkg", "macos/installer/preinstall.sh",
		d("Script run before installing (made executable automatically).", "安装前执行的脚本（自动设为可执行）。")),
	k("macos.pkg.postinstall", KPath, none, "pkg", "macos/installer/postinstall.sh",
		d("Script run after installing.", "安装后执行的脚本。")),
	k("macos.pkg.relocatable", KBool, d("false", "false"), "pkg", "true",
		d("true = Installer updates a moved copy of the app wherever it is; false = always install_location.", "true：App 被移动过时在原位置升级；false：总是安装到 install_location。")),
	k("macos.pkg.require_restart", KBool, d("false", "false"), "pkg", "true",
		d("Ask the user to restart after installing.", "安装完成后要求重启。")),
	k("macos.extra_args", KList, none, macAll, "[--no-tree-shake-icons]",
		d("Extra arguments for flutter build macos.", "追加到 flutter build macos 的参数。")),

	// ---- windows.inno_setup ----
	k("windows.inno_setup.app_id", KString, d("stable GUID derived from app.identifier", "由 app.identifier 推导的固定 GUID"), "exe", "8D3B5E6A-1C2D-4E5F-8A9B-0C1D2E3F4A5B",
		d("AppId: keep it stable forever so upgrades replace the old install.", "AppId：请永远保持不变，升级时才能覆盖旧版本。")),
	k("windows.inno_setup.script", KPath, none, "exe", "windows/installer.iss",
		d("Your own .iss script; fpack passes /DAppName /DAppVersion /DAppPublisher /DAppExeName /DSourceDir /DAppId /DAppURL.", "自定义 .iss 脚本；fpack 会传入 /DAppName /DAppVersion /DAppPublisher /DAppExeName /DSourceDir /DAppId /DAppURL。")),
	k("windows.inno_setup.iscc", KPath, d("ISCC on PATH or in Program Files", "PATH 或 Program Files 中的 ISCC"), "exe", "C:/Program Files (x86)/Inno Setup 6/ISCC.exe",
		d("Path to ISCC.exe.", "ISCC.exe 路径。")),
	k("windows.inno_setup.publisher", KString, d("app.publisher", "app.publisher"), "exe", "XueHua Tech",
		d("AppPublisher.", "AppPublisher（发布者）。")),
	k("windows.inno_setup.publisher_url", KURL, d("app.homepage", "app.homepage"), "exe", "https://xuehua.example.com",
		d("AppPublisherURL.", "AppPublisherURL（发布者网址）。")),
	k("windows.inno_setup.support_url", KURL, d("app.support_url", "app.support_url"), "exe", "https://xuehua.example.com/support",
		d("AppSupportURL.", "AppSupportURL（支持网址）。")),
	k("windows.inno_setup.updates_url", KURL, none, "exe", "https://xuehua.example.com/download",
		d("AppUpdatesURL.", "AppUpdatesURL（更新网址）。")),
	k("windows.inno_setup.default_dir", KString, d("{autopf}\\<display name>", "{autopf}\\<显示名>"), "exe", "'{autopf}\\XueHua'",
		d("Default install directory (Inno constants allowed).", "默认安装目录（可用 Inno 常量）。")),
	k("windows.inno_setup.group_name", KString, d("display name", "显示名"), "exe", "XueHua",
		d("Start menu folder.", "开始菜单文件夹。")),
	k("windows.inno_setup.desktop_icon", KEnum, d("unchecked", "unchecked"), "exe", "checked",
		d("Desktop shortcut task: none, unchecked (offered), checked (on by default).", "桌面快捷方式：none 不提供，unchecked 提供但默认不勾选，checked 默认勾选。"), enum("none", "unchecked", "checked")),
	k("windows.inno_setup.run_after_install", KBool, d("true", "true"), "exe", "false",
		d("Offer \"Launch <app>\" on the last page.", "最后一页提供“运行 <应用>”选项。")),
	k("windows.inno_setup.license_file", KPath, none, "exe", "LICENSE.txt",
		d("License page (.txt/.rtf).", "许可协议页（.txt/.rtf）。")),
	k("windows.inno_setup.info_before", KPath, none, "exe", "docs/before.txt",
		d("Information page before installing (.txt/.rtf).", "安装前信息页（.txt/.rtf）。")),
	k("windows.inno_setup.info_after", KPath, none, "exe", "docs/after.txt",
		d("Information page after installing (.txt/.rtf).", "安装后信息页（.txt/.rtf）。")),
	k("windows.inno_setup.setup_icon", KPath, d("windows/runner/resources/app_icon.ico", "windows/runner/resources/app_icon.ico"), "exe", "windows/installer/setup.ico",
		d("Installer icon (.ico).", "安装程序图标（.ico）。")),
	k("windows.inno_setup.wizard_image", KPath, none, "exe", "windows/installer/wizard.bmp",
		d("Large wizard image (.bmp/.png, 164×314 at 100%).", "向导大图（.bmp/.png，100% 缩放下 164×314）。")),
	k("windows.inno_setup.wizard_small_image", KPath, none, "exe", "windows/installer/wizard-small.bmp",
		d("Small wizard image (.bmp/.png, 55×55).", "向导小图（.bmp/.png，55×55）。")),
	k("windows.inno_setup.wizard_style", KEnum, d("modern", "modern"), "exe", "classic",
		d("Wizard style.", "向导样式。"), enum("modern", "classic")),
	k("windows.inno_setup.languages", KList, d("[en]", "[en]"), "exe", "[zh-CN, en]",
		d("Installer languages; the first is the default, the user picks one if several. zh-CN/zh-TW need Inno Setup 6.5+ (fpack ships the files otherwise). Known: en zh-CN zh-TW ja ko de fr es it pt-BR pt ru uk tr pl nl cs ar he.", "安装程序语言；第一个为默认，多个时让用户选择。zh-CN/zh-TW 需要 Inno Setup 6.5+（旧版本时 fpack 自带语言文件）。支持：en zh-CN zh-TW ja ko de fr es it pt-BR pt ru uk tr pl nl cs ar he。")),
	k("windows.inno_setup.privileges", KEnum, d("ask", "ask"), "exe", "admin",
		d("user = per-user install (no UAC), admin = all users (UAC), ask = let the user choose.", "user：仅当前用户（无需管理员）；admin：所有用户（需要 UAC）；ask：让用户选择。"), enum("user", "admin", "ask")),
	k("windows.inno_setup.compression", KString, d("lzma2/max", "lzma2/max"), "exe", "lzma2/ultra64",
		d("Compression (lzma2/max, lzma2/ultra64, zip, none …).", "压缩方式（lzma2/max、lzma2/ultra64、zip、none 等）。")),
	k("windows.inno_setup.min_version", KString, d("10.0", "10.0"), "exe", "10.0.17763",
		d("Minimum Windows version (MinVersion).", "最低 Windows 版本（MinVersion）。")),

	// ---- windows.sign ----
	k("windows.sign.certificate", KPath, none, winAll, "C:/certs/codesign.pfx",
		d("Code signing certificate (.pfx). Setting it (or thumbprint) signs the app .exe, the installer and the MSIX.", "代码签名证书（.pfx）。设置它（或 thumbprint）后会签名应用 .exe、安装程序和 MSIX。"), env("FPACK_WINDOWS_CERTIFICATE")),
	k("windows.sign.password", KString, none, winAll, "${WINDOWS_CERT_PASSWORD}",
		d("Certificate password.", "证书密码。"), env("FPACK_WINDOWS_CERTIFICATE_PASSWORD"), secret()),
	k("windows.sign.thumbprint", KString, none, "windows, exe", "1A2B3C…",
		d("SHA-1 thumbprint of a certificate in the Windows certificate store (instead of a .pfx).", "Windows 证书存储中证书的 SHA-1 指纹（替代 .pfx）。"), env("FPACK_WINDOWS_CERT_THUMBPRINT")),
	k("windows.sign.timestamp_url", KURL, d("http://timestamp.digicert.com", "http://timestamp.digicert.com"), "windows, exe", "http://timestamp.sectigo.com",
		d("RFC 3161 timestamp server.", "RFC 3161 时间戳服务器。")),
	k("windows.sign.signtool", KPath, d("signtool on PATH or in the Windows SDK", "PATH 或 Windows SDK 中的 signtool"), "windows, exe", "C:/Program Files (x86)/Windows Kits/10/bin/10.0.22621.0/x64/signtool.exe",
		d("Path to signtool.exe.", "signtool.exe 路径。")),
	k("windows.sign.description", KString, d("display name", "显示名"), "windows, exe", "XueHua IM",
		d("Description shown in the UAC prompt (/d).", "UAC 弹窗中显示的描述（/d）。")),

	// ---- windows.msix ----
	k("windows.msix.display_name", KString, d("app.display_name", "app.display_name"), "msix", "雪花IM",
		d("Display name.", "显示名称。")),
	k("windows.msix.publisher_display_name", KString, d("app.publisher", "app.publisher"), "msix", "XueHua Tech",
		d("Publisher display name.", "发布者显示名称。")),
	k("windows.msix.identity_name", KString, d("app.identifier", "app.identifier"), "msix", "com.xuehua.im",
		d("Package identity name.", "包标识名称（Identity Name）。")),
	k("windows.msix.publisher", KString, d("from the certificate", "取自证书"), "msix", "CN=XueHua Tech, O=XueHua Tech, C=CN",
		d("Publisher (certificate subject); required for the Store.", "发布者（证书 Subject）；上架 Store 时必填。")),
	k("windows.msix.version", KString, d("build name as a.b.c.0", "版本名补齐为 a.b.c.0"), "msix", "1.2.0.0",
		d("MSIX version (a.b.c.d).", "MSIX 版本（a.b.c.d）。")),
	k("windows.msix.logo", KPath, d("msix default / app icon", "msix 默认 / 应用图标"), "msix", "windows/msix/logo.png",
		d("Logo image (≥ 400×400 PNG).", "Logo 图片（≥ 400×400 的 PNG）。")),
	k("windows.msix.description", KString, d("app.description", "app.description"), "msix", "A fast and secure messenger",
		d("Package description.", "包描述。")),
	k("windows.msix.capabilities", KList, none, "msix", "[internetClient, microphone, webcam]",
		d("Capabilities.", "能力声明（capabilities）。")),
	k("windows.msix.languages", KList, none, "msix", "[zh-cn, en-us]",
		d("Languages.", "语言。")),
	k("windows.msix.file_extensions", KList, none, "msix", "[.xhim]",
		d("File extensions the app opens.", "应用可打开的文件扩展名。")),
	k("windows.msix.protocol_activation", KList, none, "msix", "[xuehua]",
		d("URL protocols that activate the app.", "可激活应用的 URL 协议。")),
	k("windows.msix.execution_alias", KString, none, "msix", "xuehua",
		d("Command-line alias.", "命令行别名。")),
	k("windows.msix.start_at_login", KBool, d("false", "false"), "msix", "true",
		d("Start the app at login.", "登录时自动启动。")),
	k("windows.msix.os_min_version", KString, d("10.0.17763.0", "10.0.17763.0"), "msix", "10.0.19041.0",
		d("Minimum Windows version.", "最低 Windows 版本。")),
	k("windows.msix.store", KBool, d("false", "false"), "msix", "true",
		d("Build for the Microsoft Store (unsigned, Store signs it).", "为 Microsoft Store 构建（不签名，由 Store 签名）。")),
	k("windows.msix.sign", KBool, d("true", "true"), "msix", "false",
		d("Sign the MSIX (false requires publisher).", "是否签名 MSIX（false 时必须设置 publisher）。")),
	k("windows.msix.certificate", KPath, d("windows.sign.certificate, else msix test certificate", "windows.sign.certificate，否则使用 msix 测试证书"), "msix", "C:/certs/codesign.pfx",
		d("Certificate (.pfx) for the MSIX.", "签名 MSIX 用的证书（.pfx）。")),
	k("windows.msix.certificate_password", KString, d("windows.sign.password", "windows.sign.password"), "msix", "${WINDOWS_CERT_PASSWORD}",
		d("Certificate password.", "证书密码。"), secret()),
	k("windows.msix.extra_args", KList, none, "msix", "[--trim-logo, \"false\"]",
		d("Extra arguments for dart run msix:create.", "追加到 dart run msix:create 的参数。")),
	k("windows.extra_args", KList, none, winAll, "[--no-tree-shake-icons]",
		d("Extra arguments for flutter build windows.", "追加到 flutter build windows 的参数。")),

	// ---- linux ----
	k("linux.package_name", KString, d("app name, lowercase with dashes", "应用名转小写并用 - 连接"), linuxP, "xuehua-im",
		d("deb/rpm package name and command name in /usr/bin.", "deb/rpm 包名，以及 /usr/bin 中的命令名。")),
	k("linux.prefix", KString, d("/opt/<package_name>", "/opt/<package_name>"), "deb, rpm", "/usr/lib/xuehua-im",
		d("Directory the app bundle is installed into.", "应用文件的安装目录。")),
	k("linux.icon", KPath, d("flutter_launcher_icons image, else web/icons/Icon-512.png", "flutter_launcher_icons 的图片，否则 web/icons/Icon-512.png"), linuxP, "assets/icon/icon.png",
		d("PNG icon for menus and the AppImage.", "菜单与 AppImage 使用的 PNG 图标。")),
	k("linux.icon_sizes", KIntList, d("[16, 32, 48, 64, 128, 256, 512]", "[16, 32, 48, 64, 128, 256, 512]"), "deb, rpm", "[48, 128, 256]",
		d("hicolor icon sizes installed (resized from linux.icon; never upscaled).", "安装到 hicolor 主题的图标尺寸（由 linux.icon 缩放，不放大）。")),
	k("linux.categories", KList, d("[Utility]", "[Utility]"), linuxP, "[Network, InstantMessaging]",
		d("freedesktop.org menu categories (.desktop Categories=).", "freedesktop.org 菜单分类（.desktop 的 Categories=）。")),
	k("linux.generic_name", KString, none, linuxP, "Instant Messenger",
		d(".desktop GenericName=.", ".desktop 的 GenericName=。")),
	k("linux.keywords", KList, none, linuxP, "[chat, im, message]",
		d(".desktop Keywords= (search terms).", ".desktop 的 Keywords=（搜索关键词）。")),
	k("linux.mime_types", KList, none, linuxP, "[x-scheme-handler/xuehua]",
		d(".desktop MimeType= (files / URL schemes the app opens).", ".desktop 的 MimeType=（可打开的文件类型 / URL 协议）。")),
	k("linux.startup_wm_class", KString, d("binary name (APPLICATION_ID on GTK)", "可执行文件名"), linuxP, "xue_hua_im",
		d(".desktop StartupWMClass= (groups windows with the launcher).", ".desktop 的 StartupWMClass=（让窗口与启动器图标归为一组）。")),
	k("linux.metainfo", KPath, none, linuxP, "linux/packaging/com.xuehua.im.metainfo.xml",
		d("AppStream metainfo installed to /usr/share/metainfo (software centers).", "AppStream metainfo，安装到 /usr/share/metainfo（软件中心展示）。")),
	k("linux.deb.depends", KList, d("[libgtk-3-0 | libgtk-3-0t64]", "[libgtk-3-0 | libgtk-3-0t64]"), "deb", "[libgtk-3-0, libsecret-1-0]",
		d("Depends.", "Depends（依赖）。")),
	k("linux.deb.recommends", KList, none, "deb", "[gnome-keyring]",
		d("Recommends.", "Recommends（推荐）。")),
	k("linux.deb.suggests", KList, none, "deb", "[libnotify-bin]",
		d("Suggests.", "Suggests（建议）。")),
	k("linux.deb.conflicts", KList, none, "deb", "[xuehua-im-beta]",
		d("Conflicts.", "Conflicts（冲突）。")),
	k("linux.deb.section", KString, d("utils", "utils"), "deb", "net",
		d("Section.", "Section（分区）。")),
	k("linux.deb.priority", KEnum, d("optional", "optional"), "deb", "optional",
		d("Priority.", "Priority（优先级）。"), enum("required", "important", "standard", "optional", "extra")),
	k("linux.deb.preinst", KPath, none, "deb", "linux/packaging/preinst",
		d("Maintainer script run before unpacking.", "解包前执行的维护脚本。")),
	k("linux.deb.postinst", KPath, none, "deb", "linux/packaging/postinst",
		d("Maintainer script run after installing.", "安装后执行的维护脚本。")),
	k("linux.deb.prerm", KPath, none, "deb", "linux/packaging/prerm",
		d("Maintainer script run before removal.", "卸载前执行的维护脚本。")),
	k("linux.deb.postrm", KPath, none, "deb", "linux/packaging/postrm",
		d("Maintainer script run after removal.", "卸载后执行的维护脚本。")),
	k("linux.rpm.requires", KList, d("[gtk3]", "[gtk3]"), "rpm", "[gtk3, libsecret]",
		d("Requires.", "Requires（依赖）。")),
	k("linux.rpm.group", KString, d("Applications/Internet", "Applications/Internet"), "rpm", "Applications/Communications",
		d("Group.", "Group（分组）。")),
	k("linux.rpm.license", KString, d("app.license", "app.license"), "rpm", "MIT",
		d("License.", "License（许可证）。")),
	k("linux.rpm.pre", KPath, none, "rpm", "linux/packaging/pre.sh",
		d("%pre script.", "%pre 脚本。")),
	k("linux.rpm.post", KPath, none, "rpm", "linux/packaging/post.sh",
		d("%post script.", "%post 脚本。")),
	k("linux.rpm.preun", KPath, none, "rpm", "linux/packaging/preun.sh",
		d("%preun script.", "%preun 脚本。")),
	k("linux.rpm.postun", KPath, none, "rpm", "linux/packaging/postun.sh",
		d("%postun script.", "%postun 脚本。")),
	k("linux.appimage.update_information", KString, none, "appimage", "gh-releases-zsync|xuehua|im|latest|*x86_64.AppImage.zsync",
		d("Embedded update information (AppImageUpdate).", "内嵌更新信息（供 AppImageUpdate 使用）。")),
	k("linux.appimage.extra_args", KList, none, "appimage", "[--comp, zstd]",
		d("Extra arguments for appimagetool.", "追加到 appimagetool 的参数。")),
	k("linux.appimagetool", KPath, d("appimagetool on PATH", "PATH 中的 appimagetool"), "appimage", "~/.local/bin/appimagetool",
		d("Path to appimagetool.", "appimagetool 路径。")),
	k("linux.extra_args", KList, none, "linux, deb, rpm, appimage", "[--no-tree-shake-icons]",
		d("Extra arguments for flutter build linux.", "追加到 flutter build linux 的参数。")),

	// ---- web ----
	k("web.base_href", KString, d("index.html as is", "保持 index.html 原样"), "web", "/app/",
		d("<base href>; must start and end with /.", "<base href>；必须以 / 开头和结尾。"), flag("--base-href")),
	k("web.wasm", KBool, d("false", "false"), "web", "true",
		d("Compile to WebAssembly (with JS fallback).", "编译为 WebAssembly（带 JS 回退）。"), flag("--wasm")),
	k("web.source_maps", KBool, d("false", "false"), "web", "true",
		d("Generate source maps.", "生成 source map。")),
	k("web.csp", KBool, d("false", "false"), "web", "true",
		d("No dynamic code generation (Content-Security-Policy).", "不动态生成代码（满足 CSP 限制）。")),
	k("web.optimization_level", KInt, d("4", "4"), "web", "2",
		d("dart2js / dart2wasm optimization level 0-4 (-O).", "dart2js / dart2wasm 优化级别 0-4（-O）。")),
	k("web.static_assets_url", KURL, none, "web", "https://cdn.example.com/app/",
		d("Serve static assets from another domain (must end with /).", "从其他域名加载静态资源（必须以 / 结尾）。")),
	k("web.web_resources_cdn", KBool, d("true", "true"), "web", "false",
		d("false = bundle CanvasKit instead of loading it from the CDN.", "false：打包 CanvasKit，不从 CDN 加载。")),
	k("web.web_define", KMap, none, "web", "{API_URL: https://api.example.com}",
		d("--web-define template variables for web/index.html.", "--web-define：web/index.html 中的模板变量。")),
	k("web.extra_args", KList, none, "web", "[--dump-info]",
		d("Extra arguments for flutter build web.", "追加到 flutter build web 的参数。")),
}

// KeyByPath returns the key with the given dotted path.
func KeyByPath(path string) (Key, bool) {
	for _, k := range Keys {
		if k.Path == path {
			return k, true
		}
	}
	return Key{}, false
}

// AppliesTo reports whether a key affects the given target.
func (k Key) AppliesTo(target string) bool {
	if k.Targets == "all" {
		return true
	}
	for _, t := range splitTargets(k.Targets) {
		if t == target {
			return true
		}
	}
	return false
}

func splitTargets(s string) []string { return stringsFields(s) }

func stringsFields(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' || r == ' ' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// IsKeyName reports whether name is the last segment of some key.
func IsKeyName(name string) bool {
	for _, k := range Keys {
		if k.Path == name || len(k.Path) > len(name) && k.Path[len(k.Path)-len(name)-1:] == "."+name {
			return true
		}
	}
	return false
}
