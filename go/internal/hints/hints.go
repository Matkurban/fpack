// Package hints turns raw tool output into a short excerpt of the real error
// plus a human hint for well-known failures.
package hints

import (
	"regexp"
	"strings"

	"github.com/Matkurban/fpack/go/internal/i18n"
)

// Hint is a known failure with an explanation and a fix.
type Hint struct {
	ID string
	en string
	zh string
}

// Text returns the hint in the active language.
func (h Hint) Text() string { return i18n.S(h.en, h.zh) }

type rule struct {
	re   *regexp.Regexp
	hint Hint
}

func r(id, pattern, en, zh string) rule {
	return rule{re: regexp.MustCompile(pattern), hint: Hint{ID: id, en: en, zh: zh}}
}

// Order matters: the first matching rule wins, so specific rules come first.
var rules = []rule{
	// ---------- Android ----------
	r("android-keystore-missing", `(?i)keystore file .* (not found|does not exist)|KeytoolException.*(FileNotFound|No such file)|Keystore file '.*' not found`,
		"The Android keystore file was not found. Check android.signing.store_file in fpack.yaml or FPACK_ANDROID_KEYSTORE (paths are relative to the project root). Run `fpack doctor apk` to verify.",
		"找不到 Android 签名文件（keystore）。请检查 fpack.yaml 中的 android.signing.store_file 或环境变量 FPACK_ANDROID_KEYSTORE（相对路径以项目根目录为准）。可运行 `fpack doctor apk` 验证。"),
	r("android-keystore-password", `(?i)keystore password was incorrect|Failed to read key .* from store|Cannot recover key|Given final block not properly padded|password was incorrect|Get Key failed`,
		"The keystore or key password is wrong, or the key alias does not exist. Check FPACK_ANDROID_KEYSTORE_PASSWORD / FPACK_ANDROID_KEY_PASSWORD / FPACK_ANDROID_KEY_ALIAS. List aliases with: keytool -list -v -keystore <file>",
		"Keystore 密码、key 密码错误，或 key 别名不存在。请检查 FPACK_ANDROID_KEYSTORE_PASSWORD / FPACK_ANDROID_KEY_PASSWORD / FPACK_ANDROID_KEY_ALIAS。查看别名：keytool -list -v -keystore <文件>"),
	r("android-licenses", `(?i)(not accepted the license agreements|licen[cs]es? (for|have) not been accepted|Android license status unknown|LicenceNotAcceptedException)`,
		"Android SDK licenses are not accepted. Run: flutter doctor --android-licenses",
		"尚未接受 Android SDK 许可协议。请运行：flutter doctor --android-licenses"),
	r("android-sdk-missing", `(?i)(No Android SDK found|Unable to locate Android SDK|SDK location not found|ANDROID_HOME.*(not set|invalid)|Android SDK not found)`,
		"Android SDK not found. Install Android Studio (or the command-line tools) and run: flutter config --android-sdk <path-to-sdk>",
		"找不到 Android SDK。请安装 Android Studio（或命令行工具），然后运行：flutter config --android-sdk <SDK 路径>"),
	r("java-mismatch", `(?i)(Unsupported class file major version|Could not determine java version|incompatible.*Java|requires Java \d+|Android Gradle plugin requires Java|Unsupported Java|JAVA_HOME is set to an invalid directory|Your build is currently configured to use incompatible Java|java\.lang\.UnsupportedClassVersionError)`,
		"Gradle and Java versions don't match. Use the JDK bundled with Android Studio (17/21): flutter config --jdk-dir=<jdk-path>   (e.g. \"/Applications/Android Studio.app/Contents/jbr/Contents/Home\"). Then run: cd android && ./gradlew --stop",
		"Gradle 与 Java 版本不匹配。请使用 Android Studio 自带的 JDK（17/21）：flutter config --jdk-dir=<JDK 路径>（例如 \"/Applications/Android Studio.app/Contents/jbr/Contents/Home\"），然后执行：cd android && ./gradlew --stop"),
	r("android-ndk", `(?i)(NDK (from ndk\.dir|at .*) did not have a source\.properties|NDK not configured|No version of NDK matched|NDK .* is not installed|CXX1101)`,
		"The Android NDK required by the project is missing or broken. Install it with: sdkmanager \"ndk;<version>\" (the version is shown above), or delete the broken folder under <android-sdk>/ndk and rebuild.",
		"项目需要的 Android NDK 缺失或损坏。请执行：sdkmanager \"ndk;<版本>\" 安装（版本见上方报错），或删除 <android-sdk>/ndk 下损坏的目录后重试。"),
	r("android-r8", `(?i)(Missing classes detected while running R8|minify\w*WithR8|R8: )`,
		"R8 (code shrinking) failed. Add the missing -keep/-dontwarn rules from build/app/outputs/mapping/*/missing_rules.txt to android/app/proguard-rules.pro.",
		"R8 代码压缩失败。请把 build/app/outputs/mapping/*/missing_rules.txt 中的规则加入 android/app/proguard-rules.pro。"),
	r("gradle-oom", `(?i)(OutOfMemoryError|Java heap space|Metaspace|Expiring Daemon because JVM heap space is exhausted)`,
		"Gradle ran out of memory. Increase org.gradle.jvmargs (e.g. -Xmx4G) in android/gradle.properties.",
		"Gradle 内存不足。请在 android/gradle.properties 中调大 org.gradle.jvmargs（例如 -Xmx4G）。"),
	r("gradle-network", `(?i)(Could not (GET|HEAD|resolve|download) .*(https?://|maven|gradle)|Connect timed out|Could not resolve all (files|dependencies)|UnknownHostException|PKIX path building failed)`,
		"Gradle could not download dependencies. Check your network/proxy (systemProp.https.proxyHost in ~/.gradle/gradle.properties) or configure a Maven mirror.",
		"Gradle 无法下载依赖。请检查网络或代理（~/.gradle/gradle.properties 中的 systemProp.https.proxyHost），或配置 Maven 镜像。"),
	r("android-minsdk", `(?i)uses-sdk:minSdkVersion \d+ cannot be smaller than version \d+`,
		"A plugin needs a higher minSdk. Raise minSdk in android/app/build.gradle(.kts) to the version shown above.",
		"某个插件需要更高的 minSdk。请把 android/app/build.gradle(.kts) 中的 minSdk 调到上方提示的版本。"),
	r("android-flavor", `(?i)(Task '.*' not found in root project|does not define a flavor|Gradle build failed to produce an \.apk|Gradle build failed to produce an \.aab)`,
		"Gradle did not produce the expected output. If you use --flavor, make sure that productFlavor exists in android/app/build.gradle(.kts) (fpack list shows detected flavors).",
		"Gradle 没有生成预期的产物。如使用了 --flavor，请确认 android/app/build.gradle(.kts) 中存在该 productFlavor（fpack list 会显示检测到的 flavor）。"),

	// ---------- iOS / macOS ----------
	r("cocoapods-missing", `(?i)(CocoaPods not installed|pod: command not found|CocoaPods is not installed|Warning: CocoaPods not installed|cocoapods.*not found)`,
		"CocoaPods is required for iOS/macOS plugins. Install it: brew install cocoapods   (then: pod setup)",
		"iOS/macOS 插件需要 CocoaPods。安装：brew install cocoapods（然后运行 pod setup）"),
	r("cocoapods-outdated", `(?i)(CocoaPods's specs repository is too out-of-date|could not find compatible versions for pod|None of your spec sources contain a spec|pod repo update)`,
		"CocoaPods specs are out of date. Run: cd ios && pod repo update && pod install   (use macos/ for macOS)",
		"CocoaPods 索引过旧。请运行：cd ios && pod repo update && pod install（macOS 用 macos/ 目录）"),
	r("cocoapods-failed", `(?i)(Error running pod install|pod install.* failed|The sandbox is not in sync with the Podfile\.lock)`,
		"`pod install` failed. Run it manually to see details: cd ios && pod install --repo-update   (use macos/ for macOS)",
		"`pod install` 执行失败。手动运行查看详情：cd ios && pod install --repo-update（macOS 用 macos/ 目录）"),
	r("ios-signing-team", `(?i)(requires a development team|Signing for "\w+" requires|No Accounts|No signing certificate|No valid code signing certificates were found|doesn't include signing certificate|Code signing is required)`,
		"Xcode signing is not configured. Open ios/Runner.xcworkspace in Xcode → Runner target → Signing & Capabilities, select your Team. For CI use an ExportOptions.plist (ios.export_options_plist). Or build an unsigned IPA with: fpack build ipa --no-codesign",
		"Xcode 签名未配置。用 Xcode 打开 ios/Runner.xcworkspace → Runner → Signing & Capabilities，选择你的 Team。CI 环境请配置 ExportOptions.plist（ios.export_options_plist）。也可以先构建未签名 IPA：fpack build ipa --no-codesign"),
	r("ios-profile", `(?i)(requires a provisioning profile|No profiles for '.*' were found|Provisioning profile .* doesn't|provisioning profile.*(expired|doesn't match)|exportArchive: No profiles)`,
		"No matching provisioning profile. In Xcode enable \"Automatically manage signing\", or download the profile for the bundle id and export method you use (ios.export_method) and reference it in ExportOptions.plist.",
		"没有匹配的描述文件（provisioning profile）。请在 Xcode 中开启“自动管理签名”，或下载与当前 bundle id、导出方式（ios.export_method）匹配的描述文件，并在 ExportOptions.plist 中引用。"),
	r("ios-export", `(?i)(error: exportArchive|Encountered error while creating the IPA|exportOptionsPlist|IPA export failed)`,
		"The archive was built but exporting the IPA failed (usually export method / profile mismatch). Check ios.export_method or ios.export_options_plist. The .xcarchive is in build/ios/archive/ and can be exported from Xcode Organizer.",
		"归档成功但导出 IPA 失败（通常是导出方式与描述文件不匹配）。请检查 ios.export_method 或 ios.export_options_plist。xcarchive 位于 build/ios/archive/，也可以在 Xcode Organizer 中手动导出。"),
	r("xcode-missing", `(?i)(xcrun: error: unable to find utility|xcode-select: error|Xcode installation is incomplete|requires Xcode|Unable to find a destination matching|xcodebuild: error: SDK .* cannot be located)`,
		"Xcode command line tools are missing or not selected. Run: sudo xcode-select -s /Applications/Xcode.app/Contents/Developer && sudo xcodebuild -runFirstLaunch",
		"未找到或未选中 Xcode 命令行工具。请运行：sudo xcode-select -s /Applications/Xcode.app/Contents/Developer && sudo xcodebuild -runFirstLaunch"),
	r("macos-identity", `(?i)(: no identity found|The specified item could not be found in the keychain|errSecInternalComponent)`,
		"The signing identity is not available in the keychain (or the keychain is locked). List identities: security find-identity -v -p codesigning   — set macos.sign.identity, or unlock the keychain: security unlock-keychain login.keychain",
		"钥匙串中找不到签名证书（或钥匙串已锁定）。查看证书：security find-identity -v -p codesigning —— 设置 macos.sign.identity，或解锁钥匙串：security unlock-keychain login.keychain"),
	r("pkg-identity", `(?i)(productbuild|pkgbuild): error: (Cannot find|Could not find|Signing).*(identity|sign)`,
		"The pkg could not be signed: the \"Developer ID Installer\" identity is missing or the keychain is locked. List them: security find-identity -v -p basic | grep Installer   — set macos.sign.installer_identity (it is NOT the Developer ID Application certificate), or build unsigned: --no-sign",
		"pkg 签名失败：缺少 “Developer ID Installer” 证书或钥匙串已锁定。查看：security find-identity -v -p basic | grep Installer —— 设置 macos.sign.installer_identity（它不是 Developer ID Application 证书），或构建未签名包：--no-sign"),
	r("notary-invalid", `(?i)(status: Invalid|"status"\s*:\s*"Invalid"|The software asset has already been uploaded|Notarization failed)`,
		"Apple rejected the notarization. Show the reason with: xcrun notarytool log <submission-id> --keychain-profile <profile>   (usually: missing hardened runtime, unsigned nested binary, or wrong certificate type – must be \"Developer ID Application\").",
		"Apple 公证被拒绝。查看原因：xcrun notarytool log <提交ID> --keychain-profile <配置名>（常见原因：未启用 hardened runtime、内嵌二进制未签名、证书类型不是 \"Developer ID Application\"）。"),
	r("notary-auth", `(?i)(No Keychain password item found for profile|HTTP status code: 401|Unable to authenticate|invalid credentials)`,
		"notarytool credentials not found or invalid. Create the profile once: xcrun notarytool store-credentials <profile> --apple-id <id> --team-id <team>   (fpack uses macos.sign.notary_profile / --notary-profile / FPACK_MACOS_NOTARY_PROFILE).",
		"notarytool 凭证不存在或无效。请先创建：xcrun notarytool store-credentials <配置名> --apple-id <账号> --team-id <团队ID>（fpack 使用 macos.sign.notary_profile / --notary-profile / FPACK_MACOS_NOTARY_PROFILE）。"),
	r("hdiutil-busy", `(?i)hdiutil: (create|attach|detach) failed - (Resource busy|Device not configured)`,
		"hdiutil could not create the DMG (a previous image may still be mounted). Eject old \"/Volumes/<app>\" volumes and retry.",
		"hdiutil 创建 DMG 失败（可能有旧镜像仍处于挂载状态）。请推出 /Volumes 下旧的卷后重试。"),
	r("concurrent-xcode", `(?i)database is locked|Xcode build failed due to concurrent builds`,
		"Another Xcode build is running. Wait for it (or quit Xcode) and retry.",
		"另一个 Xcode 构建正在进行。请等待其结束（或退出 Xcode）后重试。"),

	// ---------- Desktop ----------
	r("linux-ninja", `(?i)(unable to find a build program corresponding to "Ninja"|ninja: (command )?not found|CMAKE_MAKE_PROGRAM is not set)`,
		"Ninja is missing. Install the Linux desktop toolchain: sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev",
		"缺少 Ninja。请安装 Linux 桌面构建工具：sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev"),
	r("linux-gtk", `(?i)(Package '?gtk\+-3\.0'?.* not found|gtk\+-3\.0.*No package|Could not find GTK)`,
		"GTK 3 development files are missing: sudo apt-get install -y libgtk-3-dev   (Fedora: sudo dnf install gtk3-devel)",
		"缺少 GTK 3 开发库：sudo apt-get install -y libgtk-3-dev（Fedora：sudo dnf install gtk3-devel）"),
	r("linux-compiler", `(?i)(CMAKE_CXX_COMPILER (not set|could not be found)|No CMAKE_CXX_COMPILER could be found|clang\+\+: not found|cmake: (command )?not found|Unable to find suitable C\+\+ compiler)`,
		"C++ toolchain missing: sudo apt-get install -y clang cmake ninja-build pkg-config libstdc++-12-dev",
		"缺少 C++ 编译工具链：sudo apt-get install -y clang cmake ninja-build pkg-config libstdc++-12-dev"),
	r("windows-vs", `(?i)(Unable to find suitable Visual Studio toolchain|Visual Studio not installed|Desktop development with C\+\+)`,
		"Install Visual Studio 2022 with the \"Desktop development with C++\" workload (not VS Code): winget install Microsoft.VisualStudio.2022.Community --override \"--add Microsoft.VisualStudio.Workload.NativeDesktop --includeRecommended\"",
		"请安装带“使用 C++ 的桌面开发”工作负载的 Visual Studio 2022（不是 VS Code）：winget install Microsoft.VisualStudio.2022.Community --override \"--add Microsoft.VisualStudio.Workload.NativeDesktop --includeRecommended\""),
	r("desktop-not-enabled", `(?i)(No (Linux|Windows|macOS) desktop project configured|is not configured to build on|not enabled for this project|"build (linux|windows|macos)" is not currently supported)`,
		"This platform is not enabled for the project. Add it with: flutter create --platforms=<platform> .",
		"项目未启用该平台。请运行：flutter create --platforms=<平台> . 添加"),

	// ---------- Dart / pub (all platforms) ----------
	r("pub-path-dep", `(?i)(Could not find a file named "pubspec\.yaml" in|path dependency.*(does not exist|not found)|Because \w+ depends on \w+ from path which doesn't exist)`,
		"A path dependency in pubspec.yaml points to a missing directory (e.g. ../some_sdk). Clone/copy that package next to the project, then rerun.",
		"pubspec.yaml 中的 path 依赖指向的目录不存在（例如 ../some_sdk）。请把该包放到对应位置后重试。"),
	r("pub-solve", `(?i)(version solving failed|Because .* depends on .* which depends on|pub get failed)`,
		"Dependency resolution failed. Run `flutter pub get` to see the conflict and adjust versions in pubspec.yaml.",
		"依赖解析失败。运行 `flutter pub get` 查看冲突并调整 pubspec.yaml 中的版本。"),
	r("dart-compile", `(?m)(^lib/.*\.dart:\d+:\d+: Error:|Error: Compilation failed|Target kernel_snapshot\w* failed|Error: The getter|Error: Couldn't find constructor)`,
		"Dart compilation failed – see the file:line errors above. Run `flutter analyze` for the full list.",
		"Dart 代码编译失败 —— 请查看上方的 文件:行号 报错。运行 `flutter analyze` 查看完整列表。"),
	r("disk-full", `(?i)(No space left on device|ENOSPC|disk full)`,
		"The disk is full. Free some space (e.g. `flutter clean`, delete ~/Library/Developer/Xcode/DerivedData, ~/.gradle/caches).",
		"磁盘空间不足。请清理空间（例如 `flutter clean`、删除 ~/Library/Developer/Xcode/DerivedData、~/.gradle/caches）。"),
}

// Match returns the first known hint found in the output lines.
func Match(lines []string) (Hint, bool) {
	text := strings.Join(lines, "\n")
	for _, rl := range rules {
		if rl.re.MatchString(text) {
			return rl.hint, true
		}
	}
	return Hint{}, false
}

var errorLine = regexp.MustCompile(`(?i)(\berror\b|\bfailed\b|failure|exception|what went wrong|not found|cannot|could not|unable to|✗|^\s*e:\s)`)

// Excerpt picks the most relevant lines from the tail of a failed command:
// the Gradle "What went wrong" block when present, otherwise a window that
// starts around the first error-looking line among the last lines.
func Excerpt(lines []string, max int) []string {
	if max <= 0 {
		max = 20
	}
	// Drop trailing noise.
	trimmed := make([]string, 0, len(lines))
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "> Run with --") || strings.HasPrefix(t, "> Get more help") || strings.HasPrefix(t, "> Ask for help") {
			continue
		}
		trimmed = append(trimmed, l)
	}
	lines = trimmed
	for i, l := range lines {
		if strings.Contains(l, "What went wrong:") {
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "* Try:") || strings.HasPrefix(strings.TrimSpace(lines[j]), "* Exception is:") {
					end = j
					break
				}
			}
			if end-i > max {
				end = i + max
			}
			return lines[i:end]
		}
	}
	window := lines
	if len(window) > 120 {
		window = window[len(window)-120:]
	}
	first := -1
	for i, l := range window {
		if errorLine.MatchString(l) {
			first = i
			break
		}
	}
	if first < 0 {
		if len(window) > max {
			return window[len(window)-max:]
		}
		return window
	}
	start := first - 2
	if start < 0 {
		start = 0
	}
	end := start + max
	if end > len(window) {
		end = len(window)
		if end-max > 0 && end-max < start {
			start = end - max
		}
	}
	return window[start:end]
}
