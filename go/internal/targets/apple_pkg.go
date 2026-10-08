package targets

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
)

// --------------------------------------------------------------------- pkg --

// Pkg produces a macOS installer package: pkgbuild makes a component package
// that installs the .app into /Applications, productbuild wraps it in a
// distribution package (title, welcome/readme/license/conclusion, background)
// and signs it with a "Developer ID Installer" identity.
type Pkg struct{}

func (*Pkg) Name() string            { return "pkg" }
func (*Pkg) Platform() host.Platform { return host.MacOS }
func (*Pkg) Formats() []string       { return []string{".pkg"} }
func (*Pkg) Optional() bool          { return false }
func (*Pkg) Description() string {
	return i18n.S("macOS installer package (pkgbuild + productbuild, Developer ID Installer signing + notarization)", "macOS 安装包（pkgbuild + productbuild，可用 Developer ID Installer 签名 + 公证）")
}

// pkgResources are the optional installer pages (fpack.yaml macos.pkg).
func pkgResources(c *Context) [][2]string {
	p := c.Config.MacOS.Pkg
	var out [][2]string
	for _, r := range [][2]string{{"welcome", p.Welcome}, {"readme", p.Readme}, {"license", p.License}, {"conclusion", p.Conclusion}, {"background", p.Background}} {
		if r[1] != "" {
			out = append(out, [2]string{r[0], c.Project.Abs(r[1])})
		}
	}
	return out
}

var pkgPageExts = []string{".html", ".htm", ".rtf", ".rtfd", ".txt"}
var pkgImageExts = []string{".png", ".jpg", ".jpeg", ".tif", ".tiff", ".gif", ".pdf"}

func (*Pkg) Preflight(c *Context) []Issue {
	out := append(xcodePreflight(c, "macos"), macSigningPreflight(c)...)
	if c.Tools.Find("pkgbuild") == "" || c.Tools.Find("productbuild") == "" {
		out = append(out, fatal(i18n.S("pkgbuild/productbuild not found (they ship with macOS / the Xcode command line tools)", "找不到 pkgbuild/productbuild（随 macOS / Xcode 命令行工具提供）"), "xcode-select --install"))
	}
	for _, r := range pkgResources(c) {
		key, path := r[0], r[1]
		exts := pkgPageExts
		if key == "background" {
			exts = pkgImageExts
		}
		switch {
		case !exists(path):
			// reported by ConfigIssues
		case !contains(exts, strings.ToLower(filepath.Ext(path))):
			out = append(out, fatal(i18n.F("macos.pkg.%s: unsupported file type %q (use %s)", "macos.pkg.%s：不支持的文件类型 %q（可用 %s）", key, filepath.Ext(path), strings.Join(exts, ", ")), ""))
		}
	}
	out = append(out, installerPreflight(c)...)
	if c.Mac.Notarize && c.Mac.Installer != "" && !c.DryRun {
		if _, ok := c.Tools.Probe("xcrun", "--find", "notarytool"); !ok {
			out = append(out, fatal(i18n.S("notarization is enabled but `xcrun notarytool` is unavailable (needs Xcode 13+)", "已启用公证，但 `xcrun notarytool` 不可用（需要 Xcode 13+）"), "sudo xcode-select -s /Applications/Xcode.app/Contents/Developer"))
		}
	}
	return out
}

// InstallerIdentities lists usable installer signing identities
// ("Developer ID Installer: …", "3rd Party Mac Developer Installer: …").
// They are not code signing identities, so they only show up with the
// basic policy.
func InstallerIdentities(t Tools) ([]string, bool) {
	out, ok := t.Probe("security", "find-identity", "-v", "-p", "basic")
	if !ok && out == "" {
		return nil, false
	}
	var ids []string
	for _, id := range ParseIdentities(out) {
		if id.Problem == "" && strings.Contains(id.Name, "Installer") {
			ids = append(ids, id.Name)
		}
	}
	return ids, true
}

// installerPreflight checks the configured "Developer ID Installer" identity.
func installerPreflight(c *Context) []Issue {
	want := c.Mac.Installer
	if want == "" {
		return nil
	}
	ids, ok := InstallerIdentities(c.Tools)
	if !ok {
		return []Issue{warn(i18n.S("could not list installer signing identities (security find-identity -p basic)", "无法列出安装包签名证书（security find-identity -p basic）"), "")}
	}
	var match string
	for _, id := range ids {
		if id == want || strings.Contains(id, want) {
			match = id
			break
		}
	}
	if match == "" {
		have := i18n.S("none", "无")
		if len(ids) > 0 {
			have = "\n  " + strings.Join(ids, "\n  ")
		}
		return []Issue{fatal(i18n.F("pkg signing identity %q (macos.sign.installer_identity) not found in the keychain. Installer identities available: %s", "钥匙串中找不到 pkg 签名证书 %q（macos.sign.installer_identity）。可用的安装包证书：%s", want, have),
			i18n.S("a \"Developer ID Installer\" certificate is separate from \"Developer ID Application\": create it in Xcode → Settings → Accounts → Manage Certificates → + → Developer ID Installer (Account Holder only), or import its .p12; or build an unsigned pkg: --no-sign",
				"“Developer ID Installer” 与 “Developer ID Application” 是两个不同的证书：在 Xcode → 设置 → Accounts → Manage Certificates → + → Developer ID Installer 中创建（仅账户持有人），或导入其 .p12；也可以构建未签名 pkg：--no-sign"))}
	}
	if !strings.HasPrefix(match, "Developer ID Installer") {
		return []Issue{warn(i18n.F("%q is not a \"Developer ID Installer\" identity: such packages are only accepted by the Mac App Store, Gatekeeper rejects them elsewhere", "%q 不是 “Developer ID Installer” 证书：这类安装包只适用于 Mac App Store，在其他渠道会被 Gatekeeper 拒绝", match), "")}
	}
	return nil
}

func (*Pkg) Steps(c *Context) ([]FlutterStep, error) { return macStep(c), nil }

func (*Pkg) Locate(c *Context, predicted bool, _ time.Time) (Inputs, error) {
	return locateMacApp(c, predicted)
}

// pkgIdentifier is the package id: macos.pkg.identifier, else the macOS
// bundle id, else the best app id of the project.
func pkgIdentifier(c *Context) string {
	if id := c.Config.MacOS.Pkg.Identifier; id != "" {
		return id
	}
	if id := c.Project.MacBundleID; id != "" && !strings.Contains(id, "$") {
		return id
	}
	return c.Project.Identifier()
}

// componentPlist keeps the app at the install location: by default pkgbuild
// marks bundles relocatable, so an upgrade would land wherever an older copy
// of the app was found instead of /Applications.
func componentPlist(appName string, relocatable bool) string {
	reloc := "<false/>"
	if relocatable {
		reloc = "<true/>"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array>
	<dict>
		<key>BundleHasStrictIdentifier</key>
		<true/>
		<key>BundleIsRelocatable</key>
		` + reloc + `
		<key>BundleIsVersionChecked</key>
		<true/>
		<key>BundleOverwriteAction</key>
		<string>upgrade</string>
		<key>RootRelativeBundlePath</key>
		<string>` + xmlText(appName) + `</string>
	</dict>
</array>
</plist>
`
}

func xmlText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// distributionXML is productbuild's distribution file. hostArchitectures
// keeps Installer from asking for Rosetta on Apple silicon.
func distributionXML(title, id, version, compPkg string, res [][2]string, o pkgDistOptions) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<installer-gui-script minSpecVersion=\"2\">\n")
	fmt.Fprintf(&b, "    <title>%s</title>\n", xmlText(title))
	for _, r := range res {
		name := xmlText(filepath.Base(r[1]))
		if r[0] == "background" {
			fmt.Fprintf(&b, "    <background file=\"%s\" alignment=\"bottomleft\" scaling=\"proportional\"/>\n", name)
			fmt.Fprintf(&b, "    <background-darkAqua file=\"%s\" alignment=\"bottomleft\" scaling=\"proportional\"/>\n", name)
			continue
		}
		fmt.Fprintf(&b, "    <%s file=\"%s\"/>\n", r[0], name)
	}
	if o.MinOS != "" {
		fmt.Fprintf(&b, "    <volume-check>\n        <allowed-os-versions>\n            <os-version min=\"%s\"/>\n        </allowed-os-versions>\n    </volume-check>\n", xmlText(o.MinOS))
	}
	conclusion := "none"
	if o.Restart {
		conclusion = "RequireRestart"
	}
	fmt.Fprintf(&b, `    <options customize="never" require-scripts="%[5]t" hostArchitectures="arm64,x86_64"/>
    <domains enable_anywhere="false" enable_currentUserHome="false" enable_localSystem="true"/>
    <choices-outline>
        <line choice="default">
            <line choice="%[1]s"/>
        </line>
    </choices-outline>
    <choice id="default"/>
    <choice id="%[1]s" visible="false">
        <pkg-ref id="%[1]s"/>
    </choice>
    <pkg-ref id="%[1]s" version="%[2]s" onConclusion="%[4]s">%[3]s</pkg-ref>
</installer-gui-script>
`, xmlText(id), xmlText(version), xmlText(compPkg), conclusion, o.Scripts)
	return b.String()
}

// pkgDistOptions are distribution.xml settings beyond pages and title.
type pkgDistOptions struct {
	MinOS   string
	Restart bool
	Scripts bool
}

func (*Pkg) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.MacOS, "", "", ".pkg")
	if err != nil {
		return nil, err
	}
	app := in["app"]
	appName := filepath.Base(app)
	stage := c.Stage("pkg")
	root := filepath.Join(stage, "root")
	staged := filepath.Join(root, appName)
	plist := filepath.Join(stage, "component.plist")
	pkgs := filepath.Join(stage, "packages")
	compPkg := "app.pkg"
	resDir := filepath.Join(stage, "resources")
	dist := filepath.Join(stage, "distribution.xml")
	tmp := filepath.Join(stage, filepath.Base(dst))

	pc := c.Config.MacOS.Pkg
	id := pkgIdentifier(c)
	loc := pc.InstallLocation
	if loc == "" {
		loc = "/Applications"
	}
	title := pc.Title
	if title == "" {
		title = strings.TrimSuffix(appName, ".app")
	}
	res := pkgResources(c)
	version := orDefault(pc.Version, c.BuildName)
	minOS := orDefault(pc.MinOS, c.Project.MacDeploymentTarget)
	scripts := filepath.Join(stage, "scripts")
	hasScripts := pc.Preinstall != "" || pc.Postinstall != ""
	opts := pkgDistOptions{MinOS: minOS, Restart: boolOr(pc.RequireRestart, false), Scripts: hasScripts}
	relocatable := boolOr(pc.Relocatable, false)

	pl := &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("copy app", "复制 App"), Cmd: cmd("ditto", app, staged)},
	}}
	if c.Mac.Enabled {
		pl.Ops = append(pl.Ops, signAppOps(c, staged)...)
	}
	pl.Ops = append(pl.Ops,
		Op{Desc: i18n.S("write component plist (not relocatable) and distribution.xml", "写入组件 plist（不可重定位）与 distribution.xml"), Fn: func() error {
			if err := os.MkdirAll(pkgs, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(plist, []byte(componentPlist(appName, relocatable)), 0o644); err != nil {
				return err
			}
			for name, src := range map[string]string{"preinstall": pc.Preinstall, "postinstall": pc.Postinstall} {
				if src == "" {
					continue
				}
				if err := os.MkdirAll(scripts, 0o755); err != nil {
					return err
				}
				if err := installScript(c.Project.Abs(src), filepath.Join(scripts, name)); err != nil {
					return err
				}
			}
			for _, r := range res {
				if err := pack.CopyFile(r[1], filepath.Join(resDir, filepath.Base(r[1]))); err != nil {
					return err
				}
			}
			return os.WriteFile(dist, []byte(distributionXML(title, id, version, compPkg, res, opts)), 0o644)
		}},
		Op{Desc: i18n.F("component package → %s (pkgbuild)", "组件包 → %s（pkgbuild）", loc), Cmd: cmd("pkgbuild", pkgbuildArgs(root, plist, id, version, loc, scripts, hasScripts, filepath.Join(pkgs, compPkg))...)},
	)
	pb := []string{"--distribution", dist, "--package-path", pkgs}
	if len(res) > 0 {
		pb = append(pb, "--resources", resDir)
	}
	signed := c.Mac.Installer != ""
	if signed {
		pb = append(pb, "--sign", c.Mac.Installer, "--timestamp")
	}
	pb = append(pb, tmp)
	desc := i18n.S("installer package (productbuild)", "安装包（productbuild）")
	if signed {
		desc = i18n.S("installer package, signed (productbuild)", "安装包并签名（productbuild）")
	}
	pl.Ops = append(pl.Ops, Op{Desc: desc, Cmd: cmd("productbuild", pb...), Hint: hintInstallerIdentity()})
	kind := "macOS installer (pkg)"
	if signed {
		pl.Ops = append(pl.Ops, Op{Desc: i18n.S("verify package signature", "校验安装包签名"), Cmd: cmd("pkgutil", "--check-signature", tmp)})
		kind = "macOS installer (pkg, signed)"
	}
	notarize := signed && c.Mac.Notarize
	if notarize {
		pl.Ops = append(pl.Ops, notarizeOp(c, "pkg", tmp, dst, ""))
		if c.Config.NotarizeWait() {
			pl.Ops = append(pl.Ops,
				Op{Desc: i18n.S("staple notarization ticket", "装订公证票据"), Cmd: cmd("xcrun", "stapler", "staple", tmp)},
				Op{Desc: i18n.S("Gatekeeper assessment", "Gatekeeper 校验"), Cmd: cmd("spctl", "--assess", "--type", "install", "--verbose=2", tmp), Optional: true})
			kind = "macOS installer (pkg, signed, notarized)"
		} else {
			kind = "macOS installer (pkg, signed, notarization submitted)"
		}
	}
	pl.Ops = append(pl.Ops, moveOp(c, tmp, dst))
	if notarize {
		pl.Ops = append(pl.Ops, notaryDoneOp(c, dst, c.Config.NotarizeWait()))
	}
	pl.Artifacts = []Artifact{{Path: dst, Kind: kind}}

	switch {
	case c.Mac.TurnedOff:
		pl.Notes = append(pl.Notes, i18n.S("unsigned pkg (--no-sign): Gatekeeper blocks it on other Macs (right-click → Open to install anyway)", "未签名 pkg（--no-sign）：在其他 Mac 上会被 Gatekeeper 拦截（可右键 → 打开 强制安装）"))
	case !signed:
		pl.Notes = append(pl.Notes, i18n.S("unsigned pkg: Gatekeeper blocks it on other Macs; to sign it set macos.sign.installer_identity (\"Developer ID Installer: …\", a separate certificate from the app's Developer ID Application) in fpack.yaml",
			"未签名 pkg：在其他 Mac 上会被 Gatekeeper 拦截；如需签名，请在 fpack.yaml 中设置 macos.sign.installer_identity（“Developer ID Installer: …”，与 App 使用的 Developer ID Application 是不同的证书）"))
		if c.Mac.Notarize {
			pl.Notes = append(pl.Notes, i18n.S("pkg not notarized: Apple only notarizes signed packages", "pkg 未公证：Apple 只公证已签名的安装包"))
		}
	case !c.Mac.Enabled:
		pl.Notes = append(pl.Notes, i18n.S("the app inside the pkg is not Developer ID signed (notarization needs it); ", "pkg 内的 App 未使用 Developer ID 签名（公证需要）；")+unsignedHowTo())
	}
	return pl, nil
}

func pkgbuildArgs(root, plist, id, version, loc, scripts string, hasScripts bool, out string) []string {
	a := []string{"--root", root, "--component-plist", plist, "--identifier", id, "--version", version, "--install-location", loc}
	if hasScripts {
		a = append(a, "--scripts", scripts)
	}
	return append(a, out)
}

func hintInstallerIdentity() string {
	return i18n.S("check installer identities with: security find-identity -v -p basic | grep Installer (keychain must be unlocked)", "用 security find-identity -v -p basic | grep Installer 检查安装包证书（钥匙串需处于解锁状态）")
}
