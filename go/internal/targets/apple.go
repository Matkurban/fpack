package targets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/runner"
)

// ------------------------------------------------------------ mac signing --

// MacSigning is the resolved Developer ID signing / notarization setup for
// the macOS .app (zip) and .dmg targets.
type MacSigning struct {
	Enabled      bool
	Identity     string // "" = pick the first "Developer ID Application" identity
	Entitlements string // explicit entitlements file (absolute); "" = project default
	Notarize     bool
	Profile      string // notarytool keychain profile
	Source       string // where the settings came from (for messages)
	// NotarizeZip: notarization was requested through fpack (config, env or
	// flags), not only by the pubspec `dmg:` section, so the macOS zip is
	// notarized too. The `dmg:` section is about the DMG only.
	NotarizeZip bool
}

// DefaultNotaryProfile matches the `dmg` pub package default.
const DefaultNotaryProfile = "NotaryProfile"

// ResolveMacSigning merges, from lowest to highest priority: the `dmg:`
// section of pubspec.yaml (+ `dmg_<flavor>:`), used by the `dmg` pub
// package, read-only; then fpack.yaml macos.sign (which already carries env
// and command-line overrides).
func ResolveMacSigning(p *project.Project, cfg *config.Config) (MacSigning, error) {
	var m MacSigning
	var sources []string
	if sec := mergedDMGSection(p, cfg.Build.Flavor); sec != nil {
		sources = append(sources, "pubspec.yaml dmg:")
		m.Enabled = boolOr(sec["sign"], true)
		m.Notarize = boolOr(sec["notarization"], true)
		m.Identity, _ = sec["sign-certificate"].(string)
		m.Profile, _ = sec["notary-profile"].(string)
		if m.Profile == "" {
			m.Profile = DefaultNotaryProfile
		}
	}
	s := cfg.MacOS.Sign
	touched := false
	if s.Identity != "" {
		m.Identity = s.Identity
		if s.Enabled == nil {
			m.Enabled = true
		}
		touched = true
	}
	if s.NotaryProfile != "" {
		m.Profile = s.NotaryProfile
		if s.Notarize == nil {
			m.Notarize = true
		}
		touched = true
	}
	if s.Enabled != nil {
		m.Enabled = *s.Enabled
		touched = true
	}
	if s.Notarize != nil {
		m.Notarize = *s.Notarize
		touched = true
	}
	if s.Entitlements != "" {
		m.Entitlements = p.Abs(s.Entitlements)
		touched = true
	}
	if touched {
		sources = append(sources, "fpack.yaml/env/flags")
	}
	m.Source = strings.Join(sources, " + ")
	m.NotarizeZip = (s.Notarize != nil && *s.Notarize) || (s.Notarize == nil && s.NotaryProfile != "")
	if !m.Enabled {
		m.Notarize = false // nothing to notarize without a Developer ID signature
		m.NotarizeZip = false
	}
	if m.Notarize && m.Profile == "" {
		m.Profile = DefaultNotaryProfile
	}
	if s.Notarize != nil && *s.Notarize && s.Enabled != nil && !*s.Enabled {
		return m, fmt.Errorf("%s", i18n.S("macos.sign: notarization requires signing (enabled: false with notarize: true)", "macos.sign：公证需要先签名（enabled: false 与 notarize: true 冲突）"))
	}
	return m, nil
}

func mergedDMGSection(p *project.Project, flavor string) map[string]any {
	base := p.Section("dmg")
	var fl map[string]any
	if flavor != "" {
		fl = p.Section("dmg_" + flavor)
	}
	if base == nil && fl == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range fl {
		out[k] = v
	}
	return out
}

func boolOr(v any, d bool) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		if x, err := config.ParseBool(b); err == nil {
			return x
		}
	}
	return d
}

// IdentityLabel is the identity as shown/passed to codesign.
func (m MacSigning) IdentityLabel() string {
	if m.Identity != "" {
		return m.Identity
	}
	return "Developer ID Application"
}

var identityLine = regexp.MustCompile(`^\s*\d+\)\s+([0-9A-F]{40})\s+"([^"]+)"\s*(\(([A-Z_]+)\))?`)

// Identity is a code signing identity from the keychain.
type Identity struct {
	Name    string
	Problem string // e.g. CSSMERR_TP_CERT_REVOKED; "" = usable
}

// ParseIdentities parses `security find-identity -v -p codesigning`.
// Duplicates (the same certificate in several keychains) are merged; an
// identity is usable if any copy is.
func ParseIdentities(out string) []Identity {
	var list []Identity
	idx := map[string]int{}
	for _, l := range strings.Split(out, "\n") {
		m := identityLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		id := Identity{Name: m[2], Problem: m[4]}
		if i, ok := idx[id.Name]; ok {
			if id.Problem == "" {
				list[i].Problem = ""
			}
			continue
		}
		idx[id.Name] = len(list)
		list = append(list, id)
	}
	return list
}

// AllIdentities lists code signing identities including revoked ones.
func AllIdentities(t Tools) ([]Identity, bool) {
	out, ok := t.Probe("security", "find-identity", "-v", "-p", "codesigning")
	if !ok && out == "" {
		return nil, false
	}
	return ParseIdentities(out), true
}

// CodesignIdentities lists usable (not revoked/expired) identity names.
func CodesignIdentities(t Tools) ([]string, bool) {
	all, ok := AllIdentities(t)
	var ids []string
	for _, id := range all {
		if id.Problem == "" {
			ids = append(ids, id.Name)
		}
	}
	return ids, ok
}

// identitySummary describes identities by kind, e.g.
// "Apple Development ×2, Apple Distribution ×1".
func identitySummary(ids []string) string {
	var order []string
	n := map[string]int{}
	for _, id := range ids {
		kind := id
		if i := strings.Index(id, ":"); i > 0 {
			kind = id[:i]
		}
		if n[kind] == 0 {
			order = append(order, kind)
		}
		n[kind]++
	}
	var parts []string
	for _, k := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", k, n[k]))
	}
	if len(parts) == 0 {
		return i18n.S("none", "无")
	}
	return strings.Join(parts, ", ")
}

// macSigningPreflight validates (and auto-selects) the signing identity.
func macSigningPreflight(c *Context) []Issue {
	if !c.Mac.Enabled {
		return nil
	}
	all, ok := AllIdentities(c.Tools)
	if !ok {
		return []Issue{warn(i18n.S("could not list signing identities (security find-identity)", "无法列出签名证书（security find-identity）"), "")}
	}
	var ids, devIDs []string
	for _, id := range all {
		if id.Problem != "" {
			continue
		}
		ids = append(ids, id.Name)
		if strings.HasPrefix(id.Name, "Developer ID Application") {
			devIDs = append(devIDs, id.Name)
		}
	}
	src := ""
	if c.Mac.Source != "" {
		src = i18n.F(" (from %s)", "（来自 %s）", c.Mac.Source)
	}
	if len(devIDs) == 0 {
		// Revoked Developer ID certificates are worth calling out.
		for _, id := range all {
			if id.Problem != "" && (id.Name == c.Mac.Identity || (c.Mac.Identity == "" && strings.HasPrefix(id.Name, "Developer ID Application"))) {
				return []Issue{fatal(i18n.F("signing identity %q is not usable: %s", "签名证书 %q 不可用：%s", id.Name, id.Problem),
					i18n.S("create a new Developer ID Application certificate (Xcode → Settings → Accounts → Manage Certificates), or build unsigned: --no-sign",
						"新建 Developer ID Application 证书（Xcode → 设置 → Accounts → Manage Certificates），或构建未签名包：--no-sign"))}
			}
		}
		want := c.Mac.Identity
		if want == "" {
			want = "Developer ID Application"
		}
		return []Issue{fatal(
			i18n.F("macOS signing is enabled%s but this Mac has no \"Developer ID Application\" certificate (wanted %q; keychain has: %s). Developer ID is required to distribute outside the App Store; Apple Development/Distribution certificates can't be used for that.",
				"已启用 macOS 签名%s，但这台 Mac 上没有 “Developer ID Application” 证书（需要 %q；钥匙串中有：%s）。在 App Store 之外分发必须使用 Developer ID，Apple Development/Distribution 证书不能用于此用途。", src, want, identitySummary(ids)),
			i18n.S("install the certificate WITH its private key: export it as .p12 from the Mac where it was created (Keychain Access → My Certificates → Export) and double-click it here; or create one in Xcode → Settings → Accounts → Manage Certificates → + → Developer ID Application (Account Holder only).\nTo build now without signing: --no-sign",
				"安装证书及其私钥：在创建该证书的 Mac 上从「钥匙串访问 → 我的证书」导出 .p12，再在本机双击导入；或在 Xcode → 设置 → Accounts → Manage Certificates → + → Developer ID Application 新建（仅账户持有人）。\n先不签名构建：--no-sign"))}
	}
	if c.Mac.Identity == "" {
		c.Mac.Identity = devIDs[0]
		return nil
	}
	for _, id := range ids {
		if id == c.Mac.Identity || strings.Contains(id, c.Mac.Identity) {
			return nil
		}
	}
	return []Issue{fatal(i18n.F("signing identity %q%s not found in the keychain. Developer ID certificates available:\n  %s", "钥匙串中找不到签名证书 %q%s。可用的 Developer ID 证书：\n  %s", c.Mac.Identity, src, strings.Join(devIDs, "\n  ")),
		i18n.S("use one of them with --sign-identity \"…\" (or macos.sign.identity), or build unsigned: --no-sign", "用 --sign-identity \"…\"（或 macos.sign.identity）指定其中一个，或构建未签名包：--no-sign"))}
}

func xcodePreflight(c *Context, dir string) []Issue {
	var out []Issue
	if is, ok := flavorCheck(c, host.Platform(dir)); !ok {
		out = append(out, is)
	}
	if c.Tools.Find("xcodebuild") == "" {
		out = append(out, fatal(i18n.S("Xcode is not installed (xcodebuild not found)", "未安装 Xcode（找不到 xcodebuild）"),
			i18n.S("install Xcode from the App Store, then: sudo xcode-select -s /Applications/Xcode.app/Contents/Developer && sudo xcodebuild -runFirstLaunch", "从 App Store 安装 Xcode，然后执行：sudo xcode-select -s /Applications/Xcode.app/Contents/Developer && sudo xcodebuild -runFirstLaunch")))
	}
	if exists(filepath.Join(c.Project.Root, dir, "Podfile")) && c.Tools.Find("pod") == "" {
		out = append(out, fatal(i18n.F("CocoaPods is required (%s/Podfile exists) but `pod` was not found", "需要 CocoaPods（存在 %s/Podfile），但找不到 `pod`", dir), "brew install cocoapods"))
	}
	return out
}

// entitlements returns the entitlements file used when re-signing the app.
func (c *Context) entitlements() string {
	if c.Mac.Entitlements != "" {
		return c.Mac.Entitlements
	}
	name := "Release.entitlements"
	if c.Mode() != "release" {
		name = "DebugProfile.entitlements"
	}
	p := filepath.Join(c.Project.Root, "macos", "Runner", name)
	if exists(p) {
		return p
	}
	return ""
}

// signAppOps re-signs a staged .app with the Developer ID identity, inside
// out: nested code via --deep, then the bundle itself with the project's
// entitlements and the hardened runtime (required for notarization).
func signAppOps(c *Context, app string) []Op {
	id := c.Mac.IdentityLabel()
	sign2 := []string{"--force", "--options", "runtime", "--timestamp"}
	if e := c.entitlements(); e != "" {
		sign2 = append(sign2, "--entitlements", e)
	}
	sign2 = append(sign2, "--sign", id, app)
	return []Op{
		{Desc: i18n.S("sign nested code", "签名内嵌代码"), Cmd: cmd("codesign", "--force", "--deep", "--options", "runtime", "--timestamp", "--sign", id, app), Hint: hintIdentity()},
		{Desc: i18n.S("sign app bundle", "签名 App"), Cmd: cmd("codesign", sign2...), Hint: hintIdentity()},
		{Desc: i18n.S("verify signature", "校验签名"), Cmd: cmd("codesign", "--verify", "--deep", "--strict", "--verbose=2", app)},
	}
}

func hintIdentity() string {
	return i18n.S("check the identity with: security find-identity -v -p codesigning (keychain must be unlocked)", "用 security find-identity -v -p codesigning 检查证书（钥匙串需处于解锁状态）")
}

// notarizeOps submits a file to Apple, waits, and staples the ticket.
func notarizeOps(c *Context, file string) []Op {
	profile := c.Mac.Profile
	return []Op{
		{Desc: i18n.S("notarize (uploads to Apple and waits, usually 1–10 min)", "公证（上传到 Apple 并等待，通常 1–10 分钟）"),
			Cmd:   &runner.Cmd{Name: "xcrun", Args: []string{"notarytool", "submit", file, "--keychain-profile", profile, "--wait", "--output-format", "json"}, Capture: true},
			Check: notaryCheck(profile),
			Hint:  i18n.F("create the profile once: xcrun notarytool store-credentials %s --apple-id <apple-id> --team-id <team-id>", "先创建凭证：xcrun notarytool store-credentials %s --apple-id <Apple ID> --team-id <团队ID>", profile)},
		{Desc: i18n.S("staple notarization ticket", "装订公证票据"), Cmd: cmd("xcrun", "stapler", "staple", file)},
		{Desc: i18n.S("Gatekeeper assessment", "Gatekeeper 校验"), Cmd: cmd("spctl", "--assess", "--type", "open", "--context", "context:primary-signature", "--verbose=2", file), Optional: true},
	}
}

// NotaryResult is notarytool's JSON output.
type NotaryResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func notaryCheck(profile string) func(runner.Result) (string, error) {
	return func(r runner.Result) (string, error) {
		res, ok := ParseNotary(r.Output)
		if !ok {
			if r.ExitCode != 0 {
				return "", fmt.Errorf("notarytool failed (exit %d)", r.ExitCode)
			}
			return "", nil
		}
		if res.Status != "Accepted" {
			return "", fmt.Errorf("%s", i18n.F("notarization status %q (%s). Details: xcrun notarytool log %s --keychain-profile %s", "公证状态 %q（%s）。查看详情：xcrun notarytool log %s --keychain-profile %s", res.Status, res.Message, res.ID, profile))
		}
		return i18n.S("notarized ✓", "已公证 ✓"), nil
	}
}

// ParseNotary extracts the last JSON object from notarytool output.
func ParseNotary(out string) (NotaryResult, bool) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if strings.HasPrefix(l, "{") {
			var r NotaryResult
			if json.Unmarshal([]byte(l), &r) == nil && r.Status != "" {
				return r, true
			}
		}
	}
	return NotaryResult{}, false
}

// --------------------------------------------------------------------- ipa --

// IPA builds an iOS .ipa via `flutter build ipa` (or an unsigned IPA).
type IPA struct{}

func (*IPA) Name() string            { return "ipa" }
func (*IPA) Platform() host.Platform { return host.IOS }
func (*IPA) Formats() []string       { return []string{".ipa"} }
func (*IPA) Optional() bool          { return false }
func (*IPA) Description() string {
	return i18n.S("iOS IPA (App Store / Ad Hoc / Enterprise / unsigned)", "iOS IPA（App Store / Ad Hoc / 企业 / 未签名）")
}

func (*IPA) Preflight(c *Context) []Issue {
	out := xcodePreflight(c, "ios")
	ios := c.Config.IOS
	if ios.ExportOptionsPlist != "" && !exists(c.Project.Abs(ios.ExportOptionsPlist)) {
		out = append(out, fatal(i18n.F("export options plist not found: %s", "找不到导出配置文件：%s", ios.ExportOptionsPlist),
			i18n.S("export one from Xcode Organizer (Distribute App → Export) or fix ios.export_options_plist", "可在 Xcode Organizer 中导出（Distribute App → Export），或修正 ios.export_options_plist")))
	}
	if !c.Config.IOSCodesign() {
		return out
	}
	if ios.ExportOptionsPlist != "" && ios.ExportMethod != "" {
		out = append(out, warn(i18n.S("both ios.export_options_plist and ios.export_method are set; the plist wins", "同时设置了 ios.export_options_plist 和 ios.export_method，以 plist 为准"), ""))
	}
	if c.Project.IOSTeam == "" && ios.ExportOptionsPlist == "" {
		out = append(out, warn(i18n.S("no DEVELOPMENT_TEAM set in ios/Runner.xcodeproj – signing will probably fail", "ios/Runner.xcodeproj 中没有设置 DEVELOPMENT_TEAM —— 签名很可能失败"),
			i18n.S("open ios/Runner.xcworkspace → Runner → Signing & Capabilities → Team, or build unsigned: fpack build ipa --no-codesign", "打开 ios/Runner.xcworkspace → Runner → Signing & Capabilities → 选择 Team；或构建未签名包：fpack build ipa --no-codesign")))
	}
	if ids, ok := CodesignIdentities(c.Tools); ok {
		dist, dev := false, false
		for _, id := range ids {
			if strings.HasPrefix(id, "Apple Distribution") || strings.HasPrefix(id, "iPhone Distribution") {
				dist = true
			}
			if strings.HasPrefix(id, "Apple Development") || strings.HasPrefix(id, "iPhone Developer") {
				dev = true
			}
		}
		m := ios.ExportMethod
		needsDist := ios.ExportOptionsPlist == "" && m != "development" && m != "debugging"
		switch {
		case !dist && !dev:
			out = append(out, warn(i18n.S("no iOS signing certificate in the keychain (Xcode may still create one with automatic signing)", "钥匙串中没有 iOS 签名证书（开启自动签名时 Xcode 可能会自动创建）"),
				i18n.S("Xcode → Settings → Accounts → your team → Manage Certificates", "Xcode → 设置 → Accounts → 你的团队 → Manage Certificates")))
		case needsDist && !dist:
			out = append(out, warn(i18n.F("export method %q needs an \"Apple Distribution\" certificate, only development certificates found", "导出方式 %q 需要 “Apple Distribution” 证书，但只找到开发证书", exportMethodLabel(m)),
				i18n.S("create one in Xcode → Settings → Accounts → Manage Certificates, or use: --export-method development", "在 Xcode → 设置 → Accounts → Manage Certificates 中创建，或使用：--export-method development")))
		case needsDist && c.Project.IOSTeam != "":
			// The (XXXXXXXXXX) suffix of a distribution certificate is its team.
			if teams := distributionTeams(ids); !contains(teams, c.Project.IOSTeam) {
				out = append(out, warn(i18n.F("export method %q needs an \"Apple Distribution\" certificate for team %s, but the keychain only has distribution certificates for: %s (Xcode may still use cloud-managed signing if your account has Admin/Account Holder access)", "导出方式 %q 需要团队 %s 的 “Apple Distribution” 证书，但钥匙串中只有以下团队的发布证书：%s（如果账号有 Admin/账户持有人权限，Xcode 可能仍会使用云端管理的签名）", exportMethodLabel(m), c.Project.IOSTeam, strings.Join(teams, ", ")),
					i18n.F("install the team's distribution certificate (.p12) or create one in Xcode → Settings → Accounts → %s → Manage Certificates; or use --export-method development / --no-codesign", "安装该团队的发布证书（.p12），或在 Xcode → 设置 → Accounts → %s → Manage Certificates 中创建；也可以使用 --export-method development / --no-codesign", c.Project.IOSTeam)))
			}
		}
	}
	return out
}

func exportMethodLabel(m string) string {
	if m == "" {
		return "app-store"
	}
	return m
}

func (*IPA) Steps(c *Context) ([]FlutterStep, error) {
	args, w := CommonArgs(c, host.IOS, "ipa")
	ios := c.Config.IOS
	switch {
	case !c.Config.IOSCodesign():
		args = append(args, "--no-codesign")
	case ios.ExportOptionsPlist != "":
		args = append(args, "--export-options-plist", ios.ExportOptionsPlist)
	case ios.ExportMethod != "":
		args = append(args, "--export-method", ios.ExportMethod)
	}
	args = append(args, tailArgs(c, ios.ExtraArgs)...)
	return []FlutterStep{{Key: "ipa", Platform: host.IOS, Args: args, Warnings: w}}, nil
}

func (*IPA) Locate(c *Context, predicted bool, since time.Time) (Inputs, error) {
	root := c.Project.Root
	if !c.Config.IOSCodesign() {
		dir := filepath.Join(root, "build", "ios", "archive", "Runner.xcarchive", "Products", "Applications")
		if predicted {
			return Inputs{"app": filepath.Join(dir, "Runner.app")}, nil
		}
		if p := findNewest(filepath.Join(dir, "*.app"), time.Time{}, nil); p != "" {
			return Inputs{"app": p}, nil
		}
		return nil, notFound("*.app", c.Rel(dir))
	}
	dir := filepath.Join(root, "build", "ios", "ipa")
	if predicted {
		return Inputs{"ipa": filepath.Join(dir, "*.ipa")}, nil
	}
	if p := findNewest(filepath.Join(dir, "*.ipa"), since, nil); p != "" {
		return Inputs{"ipa": p}, nil
	}
	return nil, notFound("*.ipa", c.Rel(dir)+i18n.S(" (the archive may have been built without export – check signing)", "（可能只完成了归档而未导出 —— 请检查签名）"))
}

func (*IPA) Package(c *Context, in Inputs) (*Plan, error) {
	pl := &Plan{}
	if !c.Config.IOSCodesign() {
		dst, err := c.ArtifactPath(host.IOS, "arm64", "unsigned", ".ipa")
		if err != nil {
			return nil, err
		}
		stage := c.Stage("ipa")
		payload := filepath.Join(stage, "Payload")
		app := in["app"]
		tmp := filepath.Join(stage, filepath.Base(dst))
		pl.Ops = []Op{
			resetDirOp(c, stage),
			{Desc: i18n.S("copy app into Payload/", "复制 App 到 Payload/"), Cmd: cmd("ditto", app, filepath.Join(payload, filepath.Base(app)))},
			{Desc: i18n.S("zip Payload into .ipa", "将 Payload 压缩为 .ipa"), Cmd: cmd("ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", payload, tmp)},
			moveOp(c, tmp, dst),
		}
		pl.Artifacts = []Artifact{{Path: dst, Kind: "IPA (unsigned)", Arch: "arm64", Variant: "unsigned"}}
		pl.Notes = append(pl.Notes, i18n.S("unsigned IPA: re-sign it (e.g. with Xcode, fastlane resign or a signing service) before installing on devices", "未签名 IPA：安装到设备前需要重新签名（例如 Xcode、fastlane resign 或签名服务）"))
		return pl, nil
	}
	dst, err := c.ArtifactPath(host.IOS, "arm64", "", ".ipa")
	if err != nil {
		return nil, err
	}
	pl.Ops = []Op{copyOp(c, in["ipa"], dst)}
	method := exportMethodLabel(c.Config.IOS.ExportMethod)
	if c.Config.IOS.ExportOptionsPlist != "" {
		method = c.Config.IOS.ExportOptionsPlist
	}
	pl.Artifacts = []Artifact{{Path: dst, Kind: "IPA (" + method + ")", Arch: "arm64"}}
	return pl, nil
}

// ------------------------------------------------------------------- macOS --

// MacApp produces a zipped, optionally Developer-ID-signed .app.
type MacApp struct{}

func (*MacApp) Name() string            { return "macos" }
func (*MacApp) Platform() host.Platform { return host.MacOS }
func (*MacApp) Formats() []string       { return []string{".zip"} }
func (*MacApp) Optional() bool          { return false }
func (*MacApp) Description() string {
	return i18n.S("macOS .app, zipped with ditto (optionally Developer ID signed)", "macOS .app，用 ditto 压缩（可选 Developer ID 签名）")
}
func (*MacApp) Preflight(c *Context) []Issue {
	return append(xcodePreflight(c, "macos"), macSigningPreflight(c)...)
}

func macStep(c *Context) []FlutterStep {
	args, w := CommonArgs(c, host.MacOS, "macos")
	args = append(args, tailArgs(c, c.Config.MacOS.ExtraArgs)...)
	return []FlutterStep{{Key: "macos", Platform: host.MacOS, Args: args, Warnings: w}}
}

func (*MacApp) Steps(c *Context) ([]FlutterStep, error) { return macStep(c), nil }

// MacProductsDir is build/macos/Build/Products/<Mode>[-<flavor>].
func MacProductsDir(root, mode, flavor string) string {
	d := ModeCap(mode)
	if flavor != "" {
		d += "-" + flavor
	}
	return filepath.Join(root, "build", "macos", "Build", "Products", d)
}

func locateMacApp(c *Context, predicted bool) (Inputs, error) {
	dir := MacProductsDir(c.Project.Root, c.Mode(), c.Flavor())
	name := c.Project.MacProductName
	if name == "" || strings.Contains(name, "$") {
		name = c.Project.Name
	}
	want := filepath.Join(dir, name+".app")
	if predicted {
		return Inputs{"app": want}, nil
	}
	if exists(want) {
		return Inputs{"app": want}, nil
	}
	if p := findNewest(filepath.Join(dir, "*.app"), time.Time{}, nil); p != "" {
		return Inputs{"app": p}, nil
	}
	return nil, notFound(name+".app", c.Rel(dir))
}

func (*MacApp) Locate(c *Context, predicted bool, _ time.Time) (Inputs, error) {
	return locateMacApp(c, predicted)
}

func (*MacApp) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.MacOS, "universal", "", ".zip")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("macos")
	app := in["app"]
	tmp := filepath.Join(stage, filepath.Base(dst))
	pl := &Plan{Ops: []Op{resetDirOp(c, stage)}}
	src := app
	kind := "macOS app (zip)"
	if c.Mac.Enabled {
		src = filepath.Join(stage, filepath.Base(app))
		pl.Ops = append(pl.Ops, Op{Desc: i18n.S("copy app for signing", "复制 App 以便签名"), Cmd: cmd("ditto", app, src)})
		pl.Ops = append(pl.Ops, signAppOps(c, src)...)
		kind = "macOS app (zip, Developer ID signed)"
	} else {
		pl.Notes = append(pl.Notes, i18n.S("not re-signed (Xcode project signing is used); enable macos.sign for distribution outside the App Store", "未重新签名（使用 Xcode 工程中的签名）；如需在 App Store 外分发请启用 macos.sign"))
	}
	zip := Op{Desc: i18n.S("zip app (ditto keeps symlinks & metadata)", "压缩 App（ditto 保留符号链接与元数据）"), Cmd: cmd("ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", src, tmp)}
	pl.Ops = append(pl.Ops, zip)
	switch {
	case c.Mac.Enabled && c.Mac.NotarizeZip:
		// Apple notarizes the zip, but the ticket is stapled to the .app,
		// so the app is zipped again afterwards.
		pl.Ops = append(pl.Ops, notarizeOps(c, tmp)[0],
			Op{Desc: i18n.S("staple ticket to the app", "将票据装订到 App"), Cmd: cmd("xcrun", "stapler", "staple", src)},
			Op{Desc: i18n.S("re-zip stapled app", "重新压缩已装订的 App"), Fn: func() error { return os.Remove(tmp) }},
			zip)
		kind = "macOS app (zip, signed, notarized)"
	case c.Mac.Enabled && c.Mac.Notarize:
		pl.Notes = append(pl.Notes, i18n.S("the zip is signed but not notarized (pubspec dmg: notarization applies to the DMG); add --notarize to notarize it too", "zip 已签名但未公证（pubspec 的 dmg: 公证设置只作用于 DMG）；加 --notarize 可同时公证 zip"))
	}
	pl.Ops = append(pl.Ops, moveOp(c, tmp, dst))
	pl.Artifacts = []Artifact{{Path: dst, Kind: kind, Arch: "universal"}}
	return pl, nil
}

// --------------------------------------------------------------------- dmg --

// DMG produces a disk image with an Applications link, optionally signed,
// notarized and stapled.
type DMG struct{}

func (*DMG) Name() string            { return "dmg" }
func (*DMG) Platform() host.Platform { return host.MacOS }
func (*DMG) Formats() []string       { return []string{".dmg"} }
func (*DMG) Optional() bool          { return false }
func (*DMG) Description() string {
	return i18n.S("macOS disk image (hdiutil or create-dmg, sign + notarize)", "macOS 磁盘镜像（hdiutil 或 create-dmg，可签名 + 公证）")
}

func (*DMG) Preflight(c *Context) []Issue {
	out := append(xcodePreflight(c, "macos"), macSigningPreflight(c)...)
	if c.Config.MacOS.DMG.Tool == "create-dmg" && c.Tools.Find("create-dmg") == "" {
		out = append(out, fatal(i18n.S("macos.dmg.tool is create-dmg but it is not installed", "macos.dmg.tool 设置为 create-dmg，但未安装"), "brew install create-dmg"))
	}
	if c.Mac.Notarize && !c.DryRun {
		if _, ok := c.Tools.Probe("xcrun", "--find", "notarytool"); !ok {
			out = append(out, fatal(i18n.S("notarization is enabled but `xcrun notarytool` is unavailable (needs Xcode 13+)", "已启用公证，但 `xcrun notarytool` 不可用（需要 Xcode 13+）"), "sudo xcode-select -s /Applications/Xcode.app/Contents/Developer"))
		}
	}
	return out
}

func (*DMG) Steps(c *Context) ([]FlutterStep, error) { return macStep(c), nil }

func (*DMG) Locate(c *Context, predicted bool, _ time.Time) (Inputs, error) {
	return locateMacApp(c, predicted)
}

// dmgTool decides between create-dmg (andreyvit's script) and hdiutil.
func dmgTool(c *Context) string {
	switch c.Config.MacOS.DMG.Tool {
	case "hdiutil":
		return "hdiutil"
	case "create-dmg":
		return "create-dmg"
	}
	if p := c.Tools.Find("create-dmg"); p != "" {
		if help, _ := c.Tools.Probe(p, "--help"); strings.Contains(help, "--app-drop-link") {
			return "create-dmg"
		}
	}
	return "hdiutil"
}

func (*DMG) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.MacOS, "universal", "", ".dmg")
	if err != nil {
		return nil, err
	}
	app := in["app"]
	appName := filepath.Base(app)
	stage := c.Stage("dmg")
	root := filepath.Join(stage, "root")
	staged := filepath.Join(root, appName)
	tmp := filepath.Join(stage, filepath.Base(dst))
	vol := c.Config.MacOS.DMG.VolumeName
	if vol == "" {
		vol = strings.TrimSuffix(appName, ".app")
	}

	pl := &Plan{Ops: []Op{
		resetDirOp(c, root),
		{Desc: i18n.S("copy app", "复制 App"), Cmd: cmd("ditto", app, staged)},
	}}
	if c.Mac.Enabled {
		pl.Ops = append(pl.Ops, signAppOps(c, staged)...)
	}
	tool := dmgTool(c)
	if tool == "create-dmg" {
		args := []string{"--volname", vol, "--window-size", "660", "400", "--icon-size", "128",
			"--icon", appName, "180", "190", "--hide-extension", appName, "--app-drop-link", "480", "190"}
		if bg := c.Config.MacOS.DMG.Background; bg != "" {
			args = append(args, "--background", c.Project.Abs(bg))
		}
		if !c.Interactive {
			args = append(args, "--skip-jenkins") // no Finder/AppleScript styling in CI
		}
		args = append(args, tmp, root)
		pl.Ops = append(pl.Ops, Op{Desc: i18n.S("create DMG (create-dmg)", "创建 DMG（create-dmg）"), Cmd: cmd("create-dmg", args...)})
	} else {
		link := filepath.Join(root, "Applications")
		pl.Ops = append(pl.Ops,
			Op{Desc: i18n.S("add /Applications shortcut", "添加 /Applications 快捷方式"), Fn: func() error { return os.Symlink("/Applications", link) }},
			Op{Desc: i18n.S("create DMG (hdiutil)", "创建 DMG（hdiutil）"), Cmd: cmd("hdiutil", "create", "-volname", vol, "-srcfolder", root, "-ov", "-fs", "HFS+", "-format", "UDZO", tmp)})
	}
	kind := "DMG"
	if c.Mac.Enabled {
		pl.Ops = append(pl.Ops, Op{Desc: i18n.S("sign DMG", "签名 DMG"), Cmd: cmd("codesign", "--force", "--timestamp", "--sign", c.Mac.IdentityLabel(), tmp), Hint: hintIdentity()})
		kind = "DMG (signed)"
	}
	if c.Mac.Notarize {
		pl.Ops = append(pl.Ops, notarizeOps(c, tmp)...)
		kind = "DMG (signed, notarized)"
	}
	pl.Ops = append(pl.Ops, moveOp(c, tmp, dst))
	pl.Artifacts = []Artifact{{Path: dst, Kind: kind, Arch: "universal"}}
	if !c.Mac.Enabled {
		pl.Notes = append(pl.Notes, i18n.S("unsigned DMG: Gatekeeper will warn users. Configure macos.sign (or a pubspec dmg: section) to sign and notarize.", "未签名 DMG：用户打开时 Gatekeeper 会警告。配置 macos.sign（或 pubspec 的 dmg: 段）即可签名并公证。"))
	} else if c.Mac.Source != "" {
		pl.Notes = append(pl.Notes, i18n.F("signing settings from %s", "签名配置来源：%s", c.Mac.Source))
	}
	return pl, nil
}

var teamSuffix = regexp.MustCompile(`\(([A-Z0-9]{10})\)$`)

// distributionTeams returns the team IDs of distribution certificates.
func distributionTeams(ids []string) []string {
	var teams []string
	for _, id := range ids {
		if !strings.HasPrefix(id, "Apple Distribution") && !strings.HasPrefix(id, "iPhone Distribution") {
			continue
		}
		if m := teamSuffix.FindStringSubmatch(id); m != nil && !contains(teams, m[1]) {
			teams = append(teams, m[1])
		}
	}
	return teams
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
