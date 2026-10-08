package targets

import (
	"crypto/sha1"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
)

func winStep(c *Context) []FlutterStep {
	args, w := CommonArgs(c, host.Windows, "windows")
	args = append(args, tailArgs(c, c.Config.Windows.ExtraArgs)...)
	st := FlutterStep{Key: "windows", Platform: host.Windows, Args: args, Warnings: w}
	if winSigning(c) {
		in, _ := locateWinRelease(c, true)
		exe := filepath.Join(in["dir"], winExeName(c)+".exe")
		st.After = []Op{signtoolOp(c, exe, i18n.F("sign %s (signtool)", "签名 %s（signtool）", winExeName(c)+".exe"))}
	}
	return []FlutterStep{st}
}

// winExeName is the runner executable name without .exe.
func winExeName(c *Context) string {
	if c.Project.WindowsBinary != "" {
		return c.Project.WindowsBinary
	}
	return c.Project.Name
}

// winSigning reports whether windows.sign is configured.
func winSigning(c *Context) bool {
	s := c.Config.Windows.Sign
	return s.Certificate != "" || s.Thumbprint != ""
}

func signtool(c *Context) string {
	if p := c.Config.Windows.Sign.Signtool; p != "" {
		return c.Project.Abs(p)
	}
	return c.Tools.Find("signtool")
}

// signtoolArgs are the arguments after "signtool sign" (file excluded).
// quote wraps paths/descriptions (Inno's $q for its SignTool command line).
func signtoolArgs(c *Context, quote func(string) string) (args, secrets []string) {
	s := c.Config.Windows.Sign
	args = []string{"/fd", "sha256", "/tr", orDefault(s.TimestampURL, "http://timestamp.digicert.com"), "/td", "sha256"}
	if s.Thumbprint != "" {
		args = append(args, "/sha1", strings.ReplaceAll(s.Thumbprint, " ", ""))
	} else {
		args = append(args, "/f", quote(c.Project.Abs(s.Certificate)))
		if s.Password != "" {
			args = append(args, "/p", s.Password)
			secrets = append(secrets, s.Password)
		}
	}
	args = append(args, "/d", quote(orDefault(s.Description, c.DisplayName())))
	return args, secrets
}

func signtoolOp(c *Context, file, desc string) Op {
	args, secrets := signtoolArgs(c, func(s string) string { return s })
	return Op{Desc: desc, Cmd: &runner.Cmd{Name: orName(signtool(c), "signtool.exe"), Args: append(append([]string{"sign"}, args...), file), Secret: secrets},
		Hint: i18n.S("check windows.sign (certificate path/password or thumbprint) and the timestamp server", "请检查 windows.sign（证书路径/密码或指纹）以及时间戳服务器")}
}

func signPreflight(c *Context) []Issue {
	if !winSigning(c) || c.Host.OS != "windows" {
		return nil
	}
	if signtool(c) == "" {
		return []Issue{fatal(i18n.S("windows.sign is set but signtool.exe was not found", "设置了 windows.sign，但找不到 signtool.exe"),
			i18n.S("install the Windows SDK (Signing Tools), or set windows.sign.signtool", "安装 Windows SDK（Signing Tools），或设置 windows.sign.signtool"))}
	}
	return nil
}

func locateWinRelease(c *Context, predicted bool) (Inputs, error) {
	root := c.Project.Root
	arch := c.Host.FlutterArch()
	want := filepath.Join(root, "build", "windows", arch, "runner", ModeCap(c.Mode()))
	if predicted || exists(want) {
		return Inputs{"dir": want}, nil
	}
	legacy := filepath.Join(root, "build", "windows", "runner", ModeCap(c.Mode()))
	if exists(legacy) {
		return Inputs{"dir": legacy}, nil
	}
	return nil, notFound(ModeCap(c.Mode())+"/", c.Rel(filepath.Dir(want)))
}

func winArch(c *Context) string { return c.Host.FlutterArch() }

// WinZip is the portable Windows folder, zipped.
type WinZip struct{}

func (*WinZip) Name() string            { return "windows" }
func (*WinZip) Platform() host.Platform { return host.Windows }
func (*WinZip) Formats() []string       { return []string{".zip"} }
func (*WinZip) Optional() bool          { return false }
func (*WinZip) Description() string {
	return i18n.S("Windows portable app folder (zip)", "Windows 绿色版文件夹（zip）")
}
func (*WinZip) Preflight(c *Context) []Issue                           { return append(vsPreflight(c), signPreflight(c)...) }
func (*WinZip) Steps(c *Context) ([]FlutterStep, error)                { return winStep(c), nil }
func (*WinZip) Locate(c *Context, p bool, _ time.Time) (Inputs, error) { return locateWinRelease(c, p) }
func (*WinZip) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Windows, winArch(c), "portable", ".zip")
	if err != nil {
		return nil, err
	}
	src := in["dir"]
	tmp := filepath.Join(c.Stage("windows"), filepath.Base(dst))
	return &Plan{Ops: []Op{
		resetDirOp(c, c.Stage("windows")),
		{Desc: i18n.F("zip %s", "压缩 %s", c.Rel(src)), Fn: func() error { return pack.Zip(src, tmp, c.AppName) }},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "Windows portable (zip)", Arch: winArch(c), Variant: "portable"}}}, nil
}

func vsPreflight(c *Context) []Issue {
	if c.Host.OS != "windows" {
		return nil
	}
	vswhere := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft Visual Studio", "Installer", "vswhere.exe")
	if !exists(vswhere) {
		return []Issue{warn(i18n.S("Visual Studio installer (vswhere) not found – Visual Studio 2022 with \"Desktop development with C++\" is required", "未找到 Visual Studio（vswhere）—— 需要安装带“使用 C++ 的桌面开发”的 Visual Studio 2022"),
			`winget install Microsoft.VisualStudio.2022.Community --override "--add Microsoft.VisualStudio.Workload.NativeDesktop --includeRecommended"`)}
	}
	out, _ := c.Tools.Probe(vswhere, "-latest", "-products", "*", "-requires", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64", "-property", "installationPath")
	if strings.TrimSpace(out) == "" {
		return []Issue{fatal(i18n.S("Visual Studio C++ desktop workload is missing", "缺少 Visual Studio 的 C++ 桌面开发工作负载"),
			i18n.S("Visual Studio Installer → Modify → \"Desktop development with C++\"", "Visual Studio Installer → 修改 → 勾选“使用 C++ 的桌面开发”"))}
	}
	return nil
}

// WinExe builds an Inno Setup installer.
type WinExe struct{}

func (*WinExe) Name() string            { return "exe" }
func (*WinExe) Platform() host.Platform { return host.Windows }
func (*WinExe) Formats() []string       { return []string{".exe"} }
func (*WinExe) Optional() bool          { return true }
func (*WinExe) Description() string {
	return i18n.S("Windows installer .exe (Inno Setup)", "Windows 安装程序 .exe（Inno Setup）")
}
func iscc(c *Context) string {
	if p := c.Config.Windows.InnoSetup.ISCC; p != "" {
		return p
	}
	return c.Tools.Find("iscc")
}
func (*WinExe) Preflight(c *Context) []Issue {
	out := append(vsPreflight(c), signPreflight(c)...)
	if iscc(c) == "" {
		out = append(out, fatal(i18n.S("Inno Setup (ISCC.exe) not found", "找不到 Inno Setup（ISCC.exe）"), "winget install --id JRSoftware.InnoSetup -e"))
	}
	is := c.Config.Windows.InnoSetup
	for _, f := range []struct{ key, path string }{{"wizard_image", is.WizardImage}, {"wizard_small_image", is.WizardSmallImage}} {
		if f.path != "" && !contains([]string{".bmp", ".png"}, strings.ToLower(filepath.Ext(f.path))) {
			out = append(out, fatal(i18n.F("windows.inno_setup.%s must be a .bmp or .png", "windows.inno_setup.%s 必须是 .bmp 或 .png", f.key), ""))
		}
	}
	if is.SetupIcon != "" && !strings.EqualFold(filepath.Ext(is.SetupIcon), ".ico") {
		out = append(out, fatal(i18n.S("windows.inno_setup.setup_icon must be an .ico file", "windows.inno_setup.setup_icon 必须是 .ico 文件"), ""))
	}
	return out
}
func (*WinExe) Steps(c *Context) ([]FlutterStep, error)                { return winStep(c), nil }
func (*WinExe) Locate(c *Context, p bool, _ time.Time) (Inputs, error) { return locateWinRelease(c, p) }

// StableGUID derives a deterministic GUID from a string so upgrades of the
// same app replace the previous installation.
func StableGUID(seed string) string {
	h := sha1.Sum([]byte("fpack:" + seed))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%X-%X-%X-%X-%X", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

//go:embed isl/*.isl
var islFiles embed.FS

// InnoLanguage is one [Languages] entry.
type InnoLanguage struct {
	Name     string // identifier, e.g. zh_cn
	File     string // MessagesFile value
	Embedded string // embedded isl to write next to the script ("" = none)
}

// innoLanguages resolves windows.inno_setup.languages. Chinese message files
// ship with Inno Setup 6.5+; for older installs fpack writes its own copy.
func innoLanguages(c *Context, stage string) []InnoLanguage {
	langs := c.Config.Windows.InnoSetup.Languages
	if len(langs) == 0 {
		langs = config.List{"en"}
	}
	isccDir := ""
	if p := iscc(c); p != "" {
		isccDir = filepath.Dir(p)
	}
	var out []InnoLanguage
	for _, l := range langs {
		code := strings.ToLower(l)
		name := strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(strings.TrimSuffix(strings.ToLower(filepath.Base(l)), ".isl"))
		if strings.HasSuffix(code, ".isl") {
			out = append(out, InnoLanguage{Name: name, File: c.Project.Abs(l)})
			continue
		}
		file := config.InnoLanguages[code]
		switch {
		case file == "Default.isl":
			out = append(out, InnoLanguage{Name: name, File: "compiler:Default.isl"})
		case strings.HasPrefix(file, "Chinese") && (isccDir == "" || !exists(filepath.Join(isccDir, "Languages", file))):
			out = append(out, InnoLanguage{Name: name, File: filepath.Join(stage, file), Embedded: file})
		default:
			out = append(out, InnoLanguage{Name: name, File: "compiler:Languages\\" + file})
		}
	}
	return out
}

// InnoOptions are the values of the generated script.
type InnoOptions struct {
	AppID, Arch                                     string
	SetupIcon, LicenseFile, InfoBefore, InfoAfter   string
	WizardImage, WizardSmallImage, WizardStyle      string
	DefaultDir, GroupName, Privileges, Compression  string
	MinVersion, DesktopIcon, SupportURL, UpdatesURL string
	Copyright                                       string
	RunAfterInstall, Sign                           bool
	Languages                                       []InnoLanguage
}

func (*WinExe) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Windows, winArch(c), "setup", ".exe")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("exe")
	base := strings.TrimSuffix(filepath.Base(dst), ".exe")
	src := in["dir"]
	is := c.Config.Windows.InnoSetup
	appID := strings.Trim(is.AppID, "{}")
	if appID == "" {
		appID = StableGUID(c.Identifier())
	}
	defines := map[string]string{
		"AppName": c.DisplayName(), "AppVersion": c.BuildName, "AppPublisher": orDefault(is.Publisher, c.Publisher()),
		"AppExeName": winExeName(c) + ".exe", "SourceDir": src, "AppId": appID, "AppURL": orDefault(is.PublisherURL, c.Config.App.Homepage),
	}
	script := c.Project.Abs(is.Script)
	ops := []Op{resetDirOp(c, stage)}
	sign := winSigning(c)
	if script == "" {
		script = filepath.Join(stage, "installer.iss")
		icon := c.Project.Abs(is.SetupIcon)
		if icon == "" {
			if def := filepath.Join(c.Project.Root, "windows", "runner", "resources", "app_icon.ico"); exists(def) {
				icon = def
			}
		}
		o := InnoOptions{AppID: appID, Arch: c.Host.Arch, SetupIcon: icon, LicenseFile: c.Project.Abs(is.LicenseFile),
			InfoBefore: c.Project.Abs(is.InfoBefore), InfoAfter: c.Project.Abs(is.InfoAfter),
			WizardImage: c.Project.Abs(is.WizardImage), WizardSmallImage: c.Project.Abs(is.WizardSmallImage),
			WizardStyle: orDefault(is.WizardStyle, "modern"), DefaultDir: is.DefaultDir, GroupName: is.GroupName,
			Privileges: orDefault(is.Privileges, "ask"), Compression: orDefault(is.Compression, "lzma2/max"),
			MinVersion: orDefault(is.MinVersion, "10.0"), DesktopIcon: orDefault(is.DesktopIcon, "unchecked"),
			SupportURL: orDefault(is.SupportURL, c.SupportURL()), UpdatesURL: is.UpdatesURL, Copyright: c.Copyright(),
			RunAfterInstall: boolOr(is.RunAfterInstall, true), Sign: sign, Languages: innoLanguages(c, stage)}
		iss := InnoScript(o)
		ops = append(ops, Op{Desc: i18n.S("generate Inno Setup script", "生成 Inno Setup 脚本"), Fn: func() error {
			for _, l := range o.Languages {
				if l.Embedded == "" {
					continue
				}
				b, err := islFiles.ReadFile("isl/" + l.Embedded)
				if err != nil {
					return err
				}
				if err := os.WriteFile(l.File, append([]byte("\xef\xbb\xbf"), b...), 0o644); err != nil {
					return err
				}
			}
			return os.WriteFile(script, []byte("\xef\xbb\xbf"+iss), 0o644)
		}})
	}
	args := []string{"/Q", "/O" + stage, "/F" + base}
	var secrets []string
	if sign {
		st, sec := signtoolArgs(c, func(s string) string { return "$q" + s + "$q" })
		args = append(args, "/Sfpack=$q"+orName(signtool(c), "signtool.exe")+"$q sign "+strings.Join(st, " ")+" $f")
		secrets = sec
	}
	for _, k := range sortedKeys(defines) {
		args = append(args, "/D"+k+"="+defines[k])
	}
	args = append(args, script)
	kind := "Windows installer (Inno Setup)"
	if sign {
		kind = "Windows installer (Inno Setup, signed)"
	}
	ops = append(ops, Op{Desc: i18n.S("compile installer (ISCC)", "编译安装程序（ISCC）"), Cmd: &runner.Cmd{Name: orName(iscc(c), "ISCC.exe"), Args: args, Secret: secrets}},
		moveOp(c, filepath.Join(stage, base+".exe"), dst))
	pl := &Plan{Ops: ops, Artifacts: []Artifact{{Path: dst, Kind: kind, Arch: winArch(c), Variant: "setup"}}}
	if sign && is.Script != "" {
		pl.Notes = append(pl.Notes, i18n.S("custom script: add SignTool=fpack (and SignedUninstaller=yes) to [Setup] to use windows.sign", "自定义脚本：在 [Setup] 中加入 SignTool=fpack（以及 SignedUninstaller=yes）即可使用 windows.sign 签名"))
	}
	return pl, nil
}

func innoPath(p string) string { return strings.ReplaceAll(p, `"`, `""`) }

// InnoScript renders the installer script. Values that a custom script also
// needs come from /D defines (AppName, AppVersion, AppPublisher, AppExeName,
// SourceDir, AppId, AppURL).
func InnoScript(o InnoOptions) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("; Generated by fpack – values are passed with /D defines.")
	w("[Setup]")
	w("AppId={{%s}", o.AppID)
	w("AppName={#AppName}")
	w("AppVersion={#AppVersion}")
	w("AppVerName={#AppName} {#AppVersion}")
	w("AppPublisher={#AppPublisher}")
	w("AppPublisherURL={#AppURL}")
	if o.SupportURL != "" {
		w("AppSupportURL=%s", o.SupportURL)
	}
	if o.UpdatesURL != "" {
		w("AppUpdatesURL=%s", o.UpdatesURL)
	}
	if o.Copyright != "" {
		w("AppCopyright=%s", o.Copyright)
	}
	w("VersionInfoVersion={#AppVersion}")
	w("DefaultDirName=%s", orDefault(o.DefaultDir, `{autopf}\{#AppName}`))
	w("DefaultGroupName=%s", orDefault(o.GroupName, "{#AppName}"))
	w("DisableProgramGroupPage=yes")
	w(`UninstallDisplayIcon={app}\{#AppExeName}`)
	w("UninstallDisplayName={#AppName}")
	switch o.Privileges {
	case "admin":
		w("PrivilegesRequired=admin")
	case "user":
		w("PrivilegesRequired=lowest")
	default:
		w("PrivilegesRequired=lowest")
		w("PrivilegesRequiredOverridesAllowed=dialog commandline")
	}
	w("Compression=%s", o.Compression)
	w("SolidCompression=yes")
	w("WizardStyle=%s", o.WizardStyle)
	w("MinVersion=%s", o.MinVersion)
	if o.Arch == "arm64" {
		w("ArchitecturesAllowed=arm64")
		w("ArchitecturesInstallIn64BitMode=arm64")
	} else {
		w("ArchitecturesAllowed=x64compatible")
		w("ArchitecturesInstallIn64BitMode=x64compatible")
	}
	for _, kv := range [][2]string{{"SetupIconFile", o.SetupIcon}, {"LicenseFile", o.LicenseFile}, {"InfoBeforeFile", o.InfoBefore},
		{"InfoAfterFile", o.InfoAfter}, {"WizardImageFile", o.WizardImage}, {"WizardSmallImageFile", o.WizardSmallImage}} {
		if kv[1] != "" {
			w(`%s="%s"`, kv[0], innoPath(kv[1]))
		}
	}
	if len(o.Languages) > 1 {
		w("ShowLanguageDialog=yes")
	}
	if o.Sign {
		w("SignTool=fpack")
		w("SignedUninstaller=yes")
	}
	w("")
	w("[Languages]")
	for _, l := range o.Languages {
		w(`Name: "%s"; MessagesFile: "%s"`, l.Name, innoPath(l.File))
	}
	if o.DesktopIcon != "none" {
		w("")
		w("[Tasks]")
		flags := ""
		if o.DesktopIcon == "unchecked" {
			flags = "; Flags: unchecked"
		}
		w(`Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"%s`, flags)
	}
	w("")
	w("[Files]")
	w(`Source: "{#SourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs`)
	w("")
	w("[Icons]")
	w(`Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"`)
	w(`Name: "{group}\{cm:UninstallProgram,{#AppName}}"; Filename: "{uninstallexe}"`)
	if o.DesktopIcon != "none" {
		w(`Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon`)
	}
	if o.RunAfterInstall {
		w("")
		w("[Run]")
		w(`Filename: "{app}\{#AppExeName}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent`)
	}
	return b.String()
}

// Msix builds an MSIX package via the `msix` pub package.
type Msix struct{}

func (*Msix) Name() string            { return "msix" }
func (*Msix) Platform() host.Platform { return host.Windows }
func (*Msix) Formats() []string       { return []string{".msix"} }
func (*Msix) Optional() bool          { return true }
func (*Msix) Description() string {
	return i18n.S("Windows MSIX package (needs the msix dev_dependency)", "Windows MSIX 包（需要 msix 开发依赖）")
}
func (*Msix) Preflight(c *Context) []Issue {
	out := append(vsPreflight(c), signPreflight(c)...)
	if !c.Project.HasDep("msix") {
		out = append(out, fatal(i18n.S("the `msix` package is not in dev_dependencies (fpack does not edit pubspec.yaml)", "dev_dependencies 中没有 `msix` 包（fpack 不会修改 pubspec.yaml）"), "flutter pub add --dev msix"))
	}
	m := c.Config.Windows.Msix
	if m.Sign != nil && !*m.Sign && m.Publisher == "" && !boolOr(m.Store, false) {
		out = append(out, warn(i18n.S("windows.msix.sign is false: set windows.msix.publisher (CN=…) so the package identity is valid", "windows.msix.sign 为 false：请设置 windows.msix.publisher（CN=…），包标识才有效"), ""))
	}
	return out
}
func (*Msix) Steps(c *Context) ([]FlutterStep, error)                { return winStep(c), nil }
func (*Msix) Locate(c *Context, p bool, _ time.Time) (Inputs, error) { return locateWinRelease(c, p) }

// MsixVersion converts 1.2.3 to the 4-part 1.2.3.0 MSIX requires.
func MsixVersion(v string) string {
	parts := strings.Split(v, ".")
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:4], ".")
}

// msixArgs maps fpack.yaml windows.msix (and app.*) to msix:create flags.
// Values already in the project's msix_config win over app.* defaults, but
// explicit windows.msix keys always win.
func msixArgs(c *Context) (args, secrets []string, signedWith string) {
	m := c.Config.Windows.Msix
	cfg := c.Project.Section("msix_config")
	has := func(k string) bool { _, ok := cfg[k]; return ok }
	add := func(flag, v string) {
		if v != "" {
			args = append(args, flag, v)
		}
	}
	pick := func(explicit, key, def string) string {
		if explicit != "" {
			return explicit
		}
		if has(key) {
			return ""
		}
		return def
	}
	add("--display-name", pick(m.DisplayName, "display_name", c.Config.App.DisplayName))
	add("--publisher-display-name", pick(m.PublisherDisplayName, "publisher_display_name", c.Publisher()))
	add("--identity-name", pick(m.IdentityName, "identity_name", c.Config.App.Identifier))
	add("--publisher", m.Publisher)
	ver := m.Version
	if ver == "" && !has("msix_version") {
		ver = MsixVersion(c.BuildName)
	}
	add("--version", ver)
	if m.Logo != "" {
		add("--logo-path", c.Project.Abs(m.Logo))
	}
	add("--description", pick(m.Description, "description", c.Config.App.Description))
	add("--capabilities", strings.Join(m.Capabilities, ","))
	add("--languages", strings.Join(m.Languages, ","))
	add("--file-extension", strings.Join(m.FileExtensions, ","))
	add("--protocol-activation", strings.Join(m.ProtocolActivation, ","))
	add("--execution-alias", m.ExecutionAlias)
	if m.StartAtLogin != nil {
		add("--enable-at-startup", fmt.Sprint(*m.StartAtLogin))
	}
	add("--os-min-version", m.OSMinVersion)
	if m.Store != nil {
		add("--store", fmt.Sprint(*m.Store))
	}
	if m.Sign != nil {
		add("--sign-msix", fmt.Sprint(*m.Sign))
	}
	cert, pw := m.Certificate, m.CertificatePassword
	ws := c.Config.Windows.Sign
	if cert == "" && ws.Certificate != "" {
		cert, pw = ws.Certificate, orDefault(pw, ws.Password)
	}
	switch {
	case cert != "":
		add("--certificate-path", c.Project.Abs(cert))
		if pw != "" {
			add("--certificate-password", pw)
			secrets = append(secrets, pw)
		}
		signedWith = cert
	case ws.Thumbprint != "":
		add("--signtool-options", "/fd SHA256 /sha1 "+ws.Thumbprint+" /tr "+orDefault(ws.TimestampURL, "http://timestamp.digicert.com")+" /td SHA256")
		signedWith = "thumbprint " + ws.Thumbprint
	}
	return args, secrets, signedWith
}

func (*Msix) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Windows, winArch(c), "", ".msix")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("msix")
	base := strings.TrimSuffix(filepath.Base(dst), ".msix")
	args := []string{"run", "msix:create", "--build-windows", "false", "--output-path", stage, "--output-name", base}
	margs, secrets, signedWith := msixArgs(c)
	args = append(args, margs...)
	cfg := c.Project.Section("msix_config")
	m := c.Config.Windows.Msix
	extra := strings.Join(m.ExtraArgs, " ")
	// msix:create asks "Do you want to install the certificate?" when it signs
	// with its test certificate; with no terminal (CI, fpack's own runner) that
	// prompt crashes. Installing certificates is not packaging's job.
	if _, set := cfg["install_certificate"]; !set && !strings.Contains(extra, "--install-certificate") {
		args = append(args, "--install-certificate", "false")
	}
	args = append(args, m.ExtraArgs...)
	var notes []string
	_, hasCert := cfg["certificate_path"]
	_, hasOpts := cfg["signtool_options"]
	store := cfg["store"] == true || boolOr(m.Store, false)
	unsigned := cfg["sign_msix"] == false || (m.Sign != nil && !*m.Sign)
	if signedWith == "" && !hasCert && !hasOpts && !store && !unsigned &&
		!strings.Contains(extra, "--certificate-path") && !strings.Contains(extra, "--signtool-options") && !strings.Contains(extra, "--store") {
		notes = append(notes, i18n.S("signed with the msix package's self-signed test certificate: fine for testing; set windows.sign.certificate (or windows.msix.store: true) for distribution", "使用 msix 包自带的自签名测试证书签名：仅适合测试；正式分发请设置 windows.sign.certificate（或 windows.msix.store: true）"))
	}
	kind := "MSIX"
	if signedWith != "" {
		kind = "MSIX (signed)"
	}
	return &Plan{Notes: notes, Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("create MSIX (msix package)", "创建 MSIX（msix 包）"), Cmd: &runner.Cmd{Name: c.SDK.Dart, Args: args, Dir: c.Project.Root, Secret: secrets}},
		moveOp(c, filepath.Join(stage, base+".msix"), dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: kind, Arch: winArch(c)}}}, nil
}

func orName(p, name string) string {
	if p == "" {
		return name
	}
	return p
}
