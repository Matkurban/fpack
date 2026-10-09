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
)

// ------------------------------------------------------------ mac signing --

// MacSigning is the resolved Developer ID signing / notarization setup for
// the macOS .app (zip) and .dmg targets. It comes only from fpack's own
// configuration: fpack.yaml `macos.sign`, FPACK_MACOS_* and the command line.
type MacSigning struct {
	Enabled      bool
	Identity     string // "" = pick the first "Developer ID Application" identity
	Entitlements string // explicit entitlements file (absolute); "" = project default
	Notarize     bool
	Profile      string // notarytool keychain profile
	// Configured: an identity or notary profile is set somewhere.
	Configured bool
	// TurnedOff: signing was switched off explicitly (--no-sign,
	// FPACK_MACOS_SIGN=false, macos.sign.enabled: false).
	TurnedOff bool
	// Installer signs the .pkg ("Developer ID Installer: …", a separate
	// certificate from the app's); "" = unsigned pkg. Cleared by --no-sign.
	Installer string
	// HardenedRuntime adds --options runtime (required for notarization).
	HardenedRuntime bool
	// Notary credentials other than a keychain profile.
	AppleID, TeamID, Password   string
	APIKey, APIKeyID, APIIssuer string
}

// NotaryAuth returns the notarytool credential arguments and the secrets
// to redact: keychain profile > API key > Apple ID.
func (m MacSigning) NotaryAuth() (args, secrets []string, env []string) {
	switch {
	case m.Profile != "":
		return []string{"--keychain-profile", m.Profile}, nil, nil
	case m.APIKey != "":
		args = []string{"--key", m.APIKey, "--key-id", m.APIKeyID}
		if m.APIIssuer != "" {
			args = append(args, "--issuer", m.APIIssuer)
		}
		return args, nil, nil
	case m.AppleID != "":
		return []string{"--apple-id", m.AppleID, "--team-id", m.TeamID, "--password", m.Password}, []string{m.Password}, nil
	}
	return []string{"--keychain-profile", DefaultNotaryProfile}, nil, nil
}

// NotaryLabel describes the credential kind for messages.
func (m MacSigning) NotaryLabel() string {
	switch {
	case m.Profile != "":
		return "keychain profile " + m.Profile
	case m.APIKey != "":
		return "App Store Connect API key " + m.APIKeyID
	case m.AppleID != "":
		return "Apple ID " + m.AppleID
	}
	return "keychain profile " + DefaultNotaryProfile
}

// Source names where macOS signing settings come from (for messages).
const macSignSource = "fpack.yaml macos.sign / FPACK_MACOS_* / flags"

// DefaultNotaryProfile is used when notarization is on without a profile name.
const DefaultNotaryProfile = "NotaryProfile"

// ResolveMacSigning reads fpack.yaml macos.sign, which already carries the
// FPACK_MACOS_* and command-line overrides.
func ResolveMacSigning(p *project.Project, cfg *config.Config) (MacSigning, error) {
	s := cfg.MacOS.Sign
	m := MacSigning{Identity: s.Identity, Profile: s.NotaryProfile, HardenedRuntime: s.HardenedRuntime == nil || *s.HardenedRuntime,
		AppleID: s.NotaryAppleID, TeamID: s.NotaryTeamID, Password: s.NotaryPassword,
		APIKey: p.Abs(s.NotaryAPIKey), APIKeyID: s.NotaryAPIKeyID, APIIssuer: s.NotaryAPIIssuer}
	hasCred := s.NotaryProfile != "" || s.NotaryAppleID != "" || s.NotaryAPIKey != ""
	m.Configured = s.Identity != "" || hasCred
	switch {
	case s.Enabled != nil:
		m.Enabled = *s.Enabled
		m.TurnedOff = !*s.Enabled
	default:
		m.Enabled = s.Identity != ""
	}
	switch {
	case s.Notarize != nil:
		m.Notarize = *s.Notarize
	default:
		m.Notarize = hasCred
	}
	if s.Entitlements != "" {
		m.Entitlements = p.Abs(s.Entitlements)
	}
	if s.Notarize != nil && *s.Notarize && s.Enabled != nil && !*s.Enabled {
		return m, fmt.Errorf("%s", i18n.S("macos.sign: notarization requires signing (enabled: false with notarize: true)", "macos.sign：公证需要先签名（enabled: false 与 notarize: true 冲突）"))
	}
	if !m.TurnedOff {
		m.Installer = s.InstallerIdentity
	}
	if !m.Enabled {
		m.Notarize = false // nothing to notarize without a Developer ID signature
	}
	if m.Notarize && !hasCred {
		m.Profile = DefaultNotaryProfile
	}
	if m.Notarize && !m.HardenedRuntime {
		return m, fmt.Errorf("%s", i18n.S("macos.sign: notarization requires the hardened runtime (hardened_runtime: false with notarize)", "macos.sign：公证要求启用 hardened runtime（hardened_runtime: false 与公证冲突）"))
	}
	return m, nil
}

// unsignedHowTo explains how to turn on Developer ID signing in fpack.yaml.
func unsignedHowTo() string {
	return i18n.S("to sign + notarize, set macos.sign.identity (\"Developer ID Application: …\") and macos.sign.notary_profile in fpack.yaml (see doc/configuration.md)",
		"如需签名 + 公证，请在 fpack.yaml 中设置 macos.sign.identity（“Developer ID Application: …”）和 macos.sign.notary_profile（见 doc/configuration.md）")
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
	src := i18n.F(" (from %s)", "（来自 %s）", macSignSource)
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
	rt := []string{"--options", "runtime"}
	if !c.Mac.HardenedRuntime {
		rt = nil
	}
	sign2 := append(append([]string{"--force"}, rt...), "--timestamp")
	if e := c.entitlements(); e != "" {
		sign2 = append(sign2, "--entitlements", e)
	}
	sign2 = append(sign2, "--sign", id, app)
	return []Op{
		{Desc: i18n.S("sign nested code", "签名内嵌代码"), Cmd: cmd("codesign", append(append([]string{"--force", "--deep"}, rt...), "--timestamp", "--sign", id, app)...), Hint: hintIdentity()},
		{Desc: i18n.S("sign app bundle", "签名 App"), Cmd: cmd("codesign", sign2...), Hint: hintIdentity()},
		{Desc: i18n.S("verify signature", "校验签名"), Cmd: cmd("codesign", "--verify", "--deep", "--strict", "--verbose=2", app)},
	}
}

func hintIdentity() string {
	return i18n.S("check the identity with: security find-identity -v -p codesigning (keychain must be unlocked)", "用 security find-identity -v -p codesigning 检查证书（钥匙串需处于解锁状态）")
}

// NotaryResult is notarytool's JSON output.
type NotaryResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
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
			out = append(out, warnIfFails(i18n.S("no iOS signing certificate in the keychain (Xcode may still create one with automatic signing)", "钥匙串中没有 iOS 签名证书（开启自动签名时 Xcode 可能会自动创建）"),
				i18n.S("Xcode → Settings → Accounts → your team → Manage Certificates", "Xcode → 设置 → Accounts → 你的团队 → Manage Certificates")))
		case needsDist && !dist:
			out = append(out, warnIfFails(i18n.F("export method %q needs an \"Apple Distribution\" certificate, only development certificates found", "导出方式 %q 需要 “Apple Distribution” 证书，但只找到开发证书", exportMethodLabel(m)),
				i18n.S("create one in Xcode → Settings → Accounts → Manage Certificates, or use: --export-method development", "在 Xcode → 设置 → Accounts → Manage Certificates 中创建，或使用：--export-method development")))
		case needsDist && c.Project.IOSTeam != "":
			// The (XXXXXXXXXX) suffix of a distribution certificate is its team.
			if teams := distributionTeams(ids); !contains(teams, c.Project.IOSTeam) {
				out = append(out, warnIfFails(i18n.F("export method %q needs an \"Apple Distribution\" certificate for team %s, but the keychain only has distribution certificates for: %s (Xcode may still use cloud-managed signing if your account has Admin/Account Holder access)", "导出方式 %q 需要团队 %s 的 “Apple Distribution” 证书，但钥匙串中只有以下团队的发布证书：%s（如果账号有 Admin/账户持有人权限，Xcode 可能仍会使用云端管理的签名）", exportMethodLabel(m), c.Project.IOSTeam, strings.Join(teams, ", ")),
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
	var prepare []Op
	switch {
	case !c.Config.IOSCodesign():
		args = append(args, "--no-codesign")
	case ios.ExportOptionsPlist != "":
		args = append(args, "--export-options-plist", ios.ExportOptionsPlist)
	case iosExportOptionsSet(c):
		path := filepath.Join(c.WorkDir, "ExportOptions.plist")
		body := PlistXML(IOSExportOptions(c))
		prepare = append(prepare, Op{Desc: i18n.F("write %s", "写入 %s", c.Rel(path)), Fn: func() error {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(path, []byte(body), 0o644)
		}})
		args = append(args, "--export-options-plist", c.Rel(path))
	case ios.ExportMethod != "":
		args = append(args, "--export-method", ios.ExportMethod)
	}
	args = append(args, tailArgs(c, ios.ExtraArgs)...)
	return []FlutterStep{{Key: "ipa", Platform: host.IOS, Args: args, Warnings: w, Prepare: prepare}}, nil
}

// iosExportOptionsSet reports whether fpack must generate ExportOptions.plist.
func iosExportOptionsSet(c *Context) bool {
	i := c.Config.IOS
	return i.TeamID != "" || i.SigningStyle != "" || i.SigningCertificate != "" || len(i.ProvisioningProfiles) > 0 ||
		i.UploadSymbols != nil || i.ManageVersion != nil || i.Destination != "" || i.Thinning != "" ||
		i.StripSwiftSymbols != nil || len(i.ExportOptions) > 0
}

// IOSExportOptions builds the ExportOptions.plist dictionary.
func IOSExportOptions(c *Context) map[string]any {
	i := c.Config.IOS
	m := map[string]any{"method": orDefault(i.ExportMethod, "app-store-connect")}
	if t := orDefault(i.TeamID, c.Project.IOSTeam); t != "" {
		m["teamID"] = t
	}
	if i.SigningStyle != "" {
		m["signingStyle"] = i.SigningStyle
	}
	if i.SigningCertificate != "" {
		m["signingCertificate"] = i.SigningCertificate
	}
	if len(i.ProvisioningProfiles) > 0 {
		m["provisioningProfiles"] = i.ProvisioningProfiles
	}
	if i.UploadSymbols != nil {
		m["uploadSymbols"] = *i.UploadSymbols
	}
	if i.ManageVersion != nil {
		m["manageAppVersionAndBuildNumber"] = *i.ManageVersion
	}
	if i.Destination != "" {
		m["destination"] = i.Destination
	}
	if i.Thinning != "" {
		m["thinning"] = i.Thinning
	}
	if i.StripSwiftSymbols != nil {
		m["stripSwiftSymbols"] = *i.StripSwiftSymbols
	}
	for k, v := range i.ExportOptions {
		m[k] = v
	}
	return m
}

// iosUploads reports whether the export uploads instead of writing an IPA.
func iosUploads(c *Context) bool {
	return c.Config.IOSCodesign() && c.Config.IOS.ExportOptionsPlist == "" && c.Config.IOS.Destination == "upload"
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
	if iosUploads(c) {
		return Inputs{}, nil
	}
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
	if iosUploads(c) {
		pl.Notes = append(pl.Notes, i18n.S("destination: upload – Xcode uploaded the build to App Store Connect, no IPA is kept locally", "destination: upload —— Xcode 已将构建上传到 App Store Connect，本地不保留 IPA"))
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
	} else if iosExportOptionsSet(c) {
		method += ", generated ExportOptions.plist"
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
	dir := MacProductsDir(c.Project.Root, c.Mode(), platformFlavor(c, host.MacOS))
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
	} else if c.Mac.TurnedOff {
		pl.Notes = append(pl.Notes, i18n.S("not re-signed (--no-sign): keeps Xcode's own signature, which Gatekeeper rejects on other Macs", "未重新签名（--no-sign）：保留 Xcode 的签名，在其他 Mac 上会被 Gatekeeper 拒绝"))
	} else {
		pl.Notes = append(pl.Notes, i18n.S("not Developer ID signed (keeps Xcode's own signature, which Gatekeeper rejects on other Macs); ", "未使用 Developer ID 签名（保留 Xcode 的签名，在其他 Mac 上会被 Gatekeeper 拒绝）；")+unsignedHowTo())
	}
	zip := Op{Desc: i18n.S("zip app (ditto keeps symlinks & metadata)", "压缩 App（ditto 保留符号链接与元数据）"), Cmd: cmd("ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", src, tmp)}
	pl.Ops = append(pl.Ops, zip)
	switch {
	case c.Mac.Enabled && c.Mac.Notarize:
		// Apple notarizes the zip, but the ticket is stapled to the .app,
		// so the app is zipped again afterwards.
		pl.Ops = append(pl.Ops, notarizeOp(c, "macos", tmp, dst, filepath.Base(src)))
		if c.Config.NotarizeWait() {
			pl.Ops = append(pl.Ops,
				Op{Desc: i18n.S("staple ticket to the app", "将票据装订到 App"), Cmd: cmd("xcrun", "stapler", "staple", src)},
				Op{Desc: i18n.S("re-zip stapled app", "重新压缩已装订的 App"), Fn: func() error { return os.Remove(tmp) }},
				zip)
			kind = "macOS app (zip, signed, notarized)"
		} else {
			kind = "macOS app (zip, signed, notarization submitted)"
		}
	}
	pl.Ops = append(pl.Ops, moveOp(c, tmp, dst))
	if c.Mac.Enabled && c.Mac.Notarize {
		pl.Ops = append(pl.Ops, notaryDoneOp(c, dst, c.Config.NotarizeWait()))
	}
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
	d := c.Config.MacOS.DMG
	if d.Tool == "create-dmg" && c.Tools.Find("create-dmg") == "" {
		out = append(out, fatal(i18n.S("macos.dmg.tool is create-dmg but it is not installed", "macos.dmg.tool 设置为 create-dmg，但未安装"), "brew install create-dmg"))
	}
	if keys := dmgLayoutKeys(c); len(keys) > 0 && dmgTool(c) == "hdiutil" {
		msg := i18n.F("%s need create-dmg; hdiutil ignores them", "%s 需要 create-dmg；hdiutil 会忽略这些设置", strings.Join(keys, ", "))
		out = append(out, warn(msg, "brew install create-dmg"))
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

// dmgLayoutKeys lists configured keys only create-dmg honours.
func dmgLayoutKeys(c *Context) []string {
	d := c.Config.MacOS.DMG
	var k []string
	for _, x := range []struct {
		name string
		set  bool
	}{{"background", d.Background != ""}, {"volume_icon", d.VolumeIcon != ""}, {"window_position", len(d.WindowPosition) > 0},
		{"window_size", len(d.WindowSize) > 0}, {"icon_size", d.IconSize != 0}, {"app_position", len(d.AppPosition) > 0},
		{"applications_position", len(d.ApplicationsPosition) > 0}, {"license", d.License != ""}} {
		if x.set {
			k = append(k, "macos.dmg."+x.name)
		}
	}
	return k
}

func pairOr(p config.Pair, x, y int) []string {
	if len(p) == 2 {
		x, y = p[0], p[1]
	}
	return []string{fmt.Sprint(x), fmt.Sprint(y)}
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
		// Reset the whole stage: a DMG left by an earlier failed run
		// (e.g. rejected by notarization) would make create-dmg refuse.
		resetDirOp(c, stage),
		{Desc: i18n.S("copy app", "复制 App"), Cmd: cmd("ditto", app, staged)},
	}}
	if c.Mac.Enabled {
		pl.Ops = append(pl.Ops, signAppOps(c, staged)...)
	}
	tool := dmgTool(c)
	d := c.Config.MacOS.DMG
	format := orDefault(d.Format, "UDZO")
	fs := orDefault(d.Filesystem, "HFS+")
	if tool == "create-dmg" {
		icon := d.IconSize
		if icon == 0 {
			icon = 128
		}
		args := []string{"--volname", vol}
		if len(d.WindowPosition) == 2 {
			args = append(args, append([]string{"--window-pos"}, pairOr(d.WindowPosition, 0, 0)...)...)
		}
		args = append(args, append([]string{"--window-size"}, pairOr(d.WindowSize, 660, 400)...)...)
		args = append(args, "--icon-size", fmt.Sprint(icon))
		args = append(args, append([]string{"--icon", appName}, pairOr(d.AppPosition, 180, 190)...)...)
		args = append(args, "--hide-extension", appName)
		args = append(args, append([]string{"--app-drop-link"}, pairOr(d.ApplicationsPosition, 480, 190)...)...)
		if bg := d.Background; bg != "" {
			args = append(args, "--background", c.Project.Abs(bg))
		}
		if v := d.VolumeIcon; v != "" {
			args = append(args, "--volicon", c.Project.Abs(v))
		}
		if l := d.License; l != "" {
			args = append(args, "--eula", c.Project.Abs(l))
		}
		if format != "UDZO" {
			args = append(args, "--format", format)
		}
		if fs != "HFS+" {
			args = append(args, "--filesystem", fs)
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
			Op{Desc: i18n.S("create DMG (hdiutil)", "创建 DMG（hdiutil）"), Cmd: cmd("hdiutil", "create", "-volname", vol, "-srcfolder", root, "-ov", "-fs", fs, "-format", format, tmp)})
	}
	kind := "DMG"
	if c.Mac.Enabled {
		pl.Ops = append(pl.Ops, Op{Desc: i18n.S("sign DMG", "签名 DMG"), Cmd: cmd("codesign", "--force", "--timestamp", "--sign", c.Mac.IdentityLabel(), tmp), Hint: hintIdentity()})
		kind = "DMG (signed)"
	}
	if c.Mac.Notarize {
		pl.Ops = append(pl.Ops, notarizeOp(c, "dmg", tmp, dst, ""))
		if c.Config.NotarizeWait() {
			pl.Ops = append(pl.Ops,
				Op{Desc: i18n.S("staple notarization ticket", "装订公证票据"), Cmd: cmd("xcrun", "stapler", "staple", tmp)},
				Op{Desc: i18n.S("Gatekeeper assessment", "Gatekeeper 校验"), Cmd: cmd("spctl", "--assess", "--type", "open", "--context", "context:primary-signature", "--verbose=2", tmp), Optional: true})
			kind = "DMG (signed, notarized)"
		} else {
			kind = "DMG (signed, notarization submitted)"
		}
	}
	pl.Ops = append(pl.Ops, moveOp(c, tmp, dst))
	if c.Mac.Notarize {
		pl.Ops = append(pl.Ops, notaryDoneOp(c, dst, c.Config.NotarizeWait()))
	}
	pl.Artifacts = []Artifact{{Path: dst, Kind: kind, Arch: "universal"}}
	if c.Mac.TurnedOff {
		pl.Notes = append(pl.Notes, i18n.S("unsigned DMG (--no-sign): Gatekeeper will warn users; drop --no-sign once a Developer ID certificate is installed", "未签名 DMG（--no-sign）：用户打开时 Gatekeeper 会警告；安装 Developer ID 证书后去掉 --no-sign 即可"))
	} else if !c.Mac.Enabled {
		pl.Notes = append(pl.Notes, i18n.S("unsigned DMG: Gatekeeper will warn users; ", "未签名 DMG：用户打开时 Gatekeeper 会警告；")+unsignedHowTo())
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
