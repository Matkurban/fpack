package targets

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
)

func winStep(c *Context) []FlutterStep {
	args, w := CommonArgs(c, host.Windows, "windows")
	args = append(args, tailArgs(c, c.Config.Windows.ExtraArgs)...)
	return []FlutterStep{{Key: "windows", Platform: host.Windows, Args: args, Warnings: w}}
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
func (*WinZip) Preflight(c *Context) []Issue                           { return vsPreflight(c) }
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
	out := vsPreflight(c)
	if iscc(c) == "" {
		out = append(out, fatal(i18n.S("Inno Setup (ISCC.exe) not found", "找不到 Inno Setup（ISCC.exe）"), "winget install --id JRSoftware.InnoSetup -e"))
	}
	if s := c.Config.Windows.InnoSetup.Script; s != "" && !exists(c.Project.Abs(s)) {
		out = append(out, fatal(i18n.F("Inno Setup script not found: %s", "找不到 Inno Setup 脚本：%s", s), ""))
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

func (*WinExe) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Windows, winArch(c), "setup", ".exe")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("exe")
	base := strings.TrimSuffix(filepath.Base(dst), ".exe")
	src := in["dir"]
	exeName := c.Project.WindowsBinary
	if exeName == "" {
		exeName = c.Project.Name
	}
	appID := c.Config.Windows.InnoSetup.AppID
	if appID == "" {
		appID = StableGUID(c.Project.Identifier())
	}
	defines := map[string]string{
		"AppName": c.DisplayName(), "AppVersion": c.BuildName, "AppPublisher": c.Config.App.Publisher,
		"AppExeName": exeName + ".exe", "SourceDir": src, "AppId": appID, "AppURL": c.Config.App.Homepage,
	}
	script := c.Project.Abs(c.Config.Windows.InnoSetup.Script)
	ops := []Op{resetDirOp(c, stage)}
	if script == "" {
		script = filepath.Join(stage, "installer.iss")
		icon := filepath.Join(c.Project.Root, "windows", "runner", "resources", "app_icon.ico")
		iss := InnoScript(defines, exists(icon), icon, c.Host.Arch)
		ops = append(ops, Op{Desc: i18n.S("generate Inno Setup script", "生成 Inno Setup 脚本"), Fn: func() error { return os.WriteFile(script, []byte(iss), 0o644) }})
	}
	args := []string{"/Q", "/O" + stage, "/F" + base}
	for _, k := range sortedKeys(defines) {
		args = append(args, "/D"+k+"="+defines[k])
	}
	args = append(args, script)
	ops = append(ops, Op{Desc: i18n.S("compile installer (ISCC)", "编译安装程序（ISCC）"), Cmd: &runner.Cmd{Name: orName(iscc(c), "ISCC.exe"), Args: args}},
		moveOp(c, filepath.Join(stage, base+".exe"), dst))
	return &Plan{Ops: ops, Artifacts: []Artifact{{Path: dst, Kind: "Windows installer (Inno Setup)", Arch: winArch(c), Variant: "setup"}}}, nil
}

// InnoScript renders a default per-user/all-users installer script. Values
// come from /D defines so a custom script can use the same names.
func InnoScript(d map[string]string, hasIcon bool, icon, arch string) string {
	archLine := "ArchitecturesAllowed=x64compatible\nArchitecturesInstallIn64BitMode=x64compatible"
	if arch == "arm64" {
		archLine = "ArchitecturesAllowed=arm64\nArchitecturesInstallIn64BitMode=arm64"
	}
	iconLine := ""
	if hasIcon {
		iconLine = "SetupIconFile=" + icon + "\n"
	}
	return `; Generated by fpack – values are passed with /D defines.
[Setup]
AppId={{` + d["AppId"] + `}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
UninstallDisplayIcon={app}\{#AppExeName}
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
` + archLine + "\n" + iconLine + `
[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "{#SourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"
Name: "{group}\{cm:UninstallProgram,{#AppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExeName}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent
`
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
	out := vsPreflight(c)
	if !c.Project.HasDep("msix") {
		out = append(out, fatal(i18n.S("the `msix` package is not in dev_dependencies (fpack does not edit pubspec.yaml)", "dev_dependencies 中没有 `msix` 包（fpack 不会修改 pubspec.yaml）"), "flutter pub add --dev msix"))
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

func (*Msix) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Windows, winArch(c), "", ".msix")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("msix")
	base := strings.TrimSuffix(filepath.Base(dst), ".msix")
	args := []string{"run", "msix:create", "--build-windows", "false", "--output-path", stage, "--output-name", base, "--version", MsixVersion(c.BuildName)}
	cfg := c.Project.Section("msix_config")
	extra := strings.Join(c.Config.Windows.Msix.ExtraArgs, " ")
	// msix:create asks "Do you want to install the certificate?" when it signs
	// with its test certificate; with no terminal (CI, fpack's own runner) that
	// prompt crashes. Installing certificates is not packaging's job.
	if _, set := cfg["install_certificate"]; !set && !strings.Contains(extra, "--install-certificate") {
		args = append(args, "--install-certificate", "false")
	}
	args = append(args, c.Config.Windows.Msix.ExtraArgs...)
	var notes []string
	_, hasCert := cfg["certificate_path"]
	_, hasOpts := cfg["signtool_options"]
	if !hasCert && !hasOpts && cfg["store"] != true && cfg["sign_msix"] != false &&
		!strings.Contains(extra, "--certificate-path") && !strings.Contains(extra, "--signtool-options") && !strings.Contains(extra, "--store") {
		notes = append(notes, i18n.S("signed with the msix package's self-signed test certificate: fine for testing; set msix_config certificate_path (or store: true) for distribution", "使用 msix 包自带的自签名测试证书签名：仅适合测试；正式分发请在 msix_config 中设置 certificate_path（或 store: true）"))
	}
	return &Plan{Notes: notes, Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("create MSIX (msix package)", "创建 MSIX（msix 包）"), Cmd: &runner.Cmd{Name: c.SDK.Dart, Args: args, Dir: c.Project.Root}},
		moveOp(c, filepath.Join(stage, base+".msix"), dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "MSIX", Arch: winArch(c)}}}, nil
}

func orName(p, name string) string {
	if p == "" {
		return name
	}
	return p
}
