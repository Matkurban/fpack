package targets

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
)

func linuxStep(c *Context) []FlutterStep {
	args, w := CommonArgs(c, host.Linux, "linux")
	args = append(args, tailArgs(c, c.Config.Linux.ExtraArgs)...)
	return []FlutterStep{{Key: "linux", Platform: host.Linux, Args: args, Warnings: w}}
}

func locateLinuxBundle(c *Context, predicted bool) (Inputs, error) {
	want := filepath.Join(c.Project.Root, "build", "linux", c.Host.FlutterArch(), c.Mode(), "bundle")
	if predicted || exists(want) {
		return Inputs{"bundle": want}, nil
	}
	return nil, notFound("bundle/", c.Rel(filepath.Dir(want)))
}

func linuxPreflight(c *Context) []Issue {
	var out []Issue
	pk := map[string]string{"apt": "clang cmake ninja-build pkg-config libgtk-3-dev", "dnf": "clang cmake ninja-build pkgconf-pkg-config gtk3-devel", "pacman": "clang cmake ninja pkgconf gtk3"}
	var missing []string
	for _, t := range []string{"clang++", "cmake", "ninja", "pkg-config"} {
		if c.Tools.Find(t) == "" {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		out = append(out, fatal(i18n.F("Linux desktop toolchain incomplete, missing: %s", "Linux 桌面构建工具不完整，缺少：%s", strings.Join(missing, ", ")), installHint(c.Host, pk)))
	} else if _, ok := c.Tools.Probe("pkg-config", "--exists", "gtk+-3.0"); !ok {
		out = append(out, fatal(i18n.S("GTK 3 development files missing (pkg-config gtk+-3.0)", "缺少 GTK 3 开发文件（pkg-config gtk+-3.0）"), installHint(c.Host, pk)))
	}
	return out
}

func linuxArch(c *Context) string { return c.Host.FlutterArch() }

// PackageName is the deb/rpm package name (lowercase, dashes).
func (c *Context) PackageName() string {
	if n := c.Config.Linux.PackageName; n != "" {
		return n
	}
	return strings.ToLower(strings.ReplaceAll(c.Project.Name, "_", "-"))
}

func (c *Context) linuxBinary() string {
	if c.Project.LinuxBinary != "" {
		return c.Project.LinuxBinary
	}
	return c.Project.Name
}

// linuxIcon picks a PNG: linux.icon > flutter_launcher_icons image > web icon.
func (c *Context) linuxIcon() string {
	for _, p := range []string{c.Config.Linux.Icon, c.Project.LauncherIcon, "web/icons/Icon-512.png", "web/icons/Icon-192.png"} {
		if p != "" && strings.HasSuffix(strings.ToLower(p), ".png") && exists(c.Project.Abs(p)) {
			return c.Project.Abs(p)
		}
	}
	return ""
}

func (c *Context) desktopEntry(exec, icon string) string {
	cat := c.Config.Linux.Categories
	if cat == "" {
		cat = "Utility;"
	}
	if !strings.HasSuffix(cat, ";") {
		cat += ";"
	}
	comment := c.Config.App.Description
	if comment == "" {
		comment = c.Project.Description
	}
	s := "[Desktop Entry]\nType=Application\nName=" + c.DisplayName() + "\n"
	if comment != "" {
		s += "Comment=" + strings.ReplaceAll(comment, "\n", " ") + "\n"
	}
	s += "Exec=" + exec + " %U\n"
	if icon != "" {
		s += "Icon=" + icon + "\n"
	}
	s += "Terminal=false\nCategories=" + cat + "\nStartupWMClass=" + c.linuxBinary() + "\n"
	return s
}

// LinuxTar is the bundle as .tar.gz.
type LinuxTar struct{}

func (*LinuxTar) Name() string            { return "linux" }
func (*LinuxTar) Platform() host.Platform { return host.Linux }
func (*LinuxTar) Formats() []string       { return []string{".tar.gz"} }
func (*LinuxTar) Optional() bool          { return false }
func (*LinuxTar) Description() string {
	return i18n.S("Linux app bundle (tar.gz)", "Linux 应用目录（tar.gz）")
}
func (*LinuxTar) Preflight(c *Context) []Issue            { return linuxPreflight(c) }
func (*LinuxTar) Steps(c *Context) ([]FlutterStep, error) { return linuxStep(c), nil }
func (*LinuxTar) Locate(c *Context, p bool, _ time.Time) (Inputs, error) {
	return locateLinuxBundle(c, p)
}
func (*LinuxTar) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Linux, linuxArch(c), "", ".tar.gz")
	if err != nil {
		return nil, err
	}
	src := in["bundle"]
	stage := c.Stage("linux")
	tmp := filepath.Join(stage, filepath.Base(dst))
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.F("tar.gz %s", "打包 %s", c.Rel(src)), Fn: func() error { return pack.TarGz(src, tmp, c.AppName) }},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "Linux bundle (tar.gz)", Arch: linuxArch(c)}}}, nil
}

// stageLinuxRoot lays out /opt/<pkg>, /usr/bin/<pkg>, desktop file and icon.
func stageLinuxRoot(c *Context, bundle, root string) error {
	pkg := c.PackageName()
	opt := filepath.Join(root, "opt", pkg)
	if err := os.MkdirAll(filepath.Dir(opt), 0o755); err != nil {
		return err
	}
	if err := pack.CopyDir(bundle, opt); err != nil {
		return err
	}
	bin := filepath.Join(root, "usr", "bin")
	os.MkdirAll(bin, 0o755)
	if err := os.Symlink("/opt/"+pkg+"/"+c.linuxBinary(), filepath.Join(bin, pkg)); err != nil {
		return err
	}
	apps := filepath.Join(root, "usr", "share", "applications")
	os.MkdirAll(apps, 0o755)
	icon := ""
	if src := c.linuxIcon(); src != "" {
		pix := filepath.Join(root, "usr", "share", "pixmaps")
		os.MkdirAll(pix, 0o755)
		if err := pack.CopyFile(src, filepath.Join(pix, pkg+".png")); err != nil {
			return err
		}
		os.Chmod(filepath.Join(pix, pkg+".png"), 0o644)
		icon = pkg
	}
	return os.WriteFile(filepath.Join(apps, pkg+".desktop"), []byte(c.desktopEntry(pkg, icon)), 0o644)
}

func dirSizeKB(dir string) int64 {
	var n int64
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				n += info.Size()
			}
		}
		return nil
	})
	return (n + 1023) / 1024
}

// Deb builds a Debian package with dpkg-deb.
type Deb struct{}

func (*Deb) Name() string            { return "deb" }
func (*Deb) Platform() host.Platform { return host.Linux }
func (*Deb) Formats() []string       { return []string{".deb"} }
func (*Deb) Optional() bool          { return true }
func (*Deb) Description() string {
	return i18n.S("Debian/Ubuntu package (dpkg-deb)", "Debian/Ubuntu 安装包（dpkg-deb）")
}
func (*Deb) Preflight(c *Context) []Issue {
	out := linuxPreflight(c)
	if c.Tools.Find("dpkg-deb") == "" {
		out = append(out, fatal(i18n.S("dpkg-deb not found", "找不到 dpkg-deb"), installHint(c.Host, map[string]string{"apt": "dpkg-dev", "dnf": "dpkg", "pacman": "dpkg"})))
	}
	return out
}
func (*Deb) Steps(c *Context) ([]FlutterStep, error)                { return linuxStep(c), nil }
func (*Deb) Locate(c *Context, p bool, _ time.Time) (Inputs, error) { return locateLinuxBundle(c, p) }

// DebArch maps Flutter arch names to Debian ones.
func DebArch(a string) string {
	if a == "arm64" {
		return "arm64"
	}
	return "amd64"
}

func (*Deb) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Linux, linuxArch(c), "", ".deb")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("deb")
	root := filepath.Join(stage, "root")
	tmp := filepath.Join(stage, filepath.Base(dst))
	bundle := in["bundle"]
	depends := strings.Join(c.Config.Linux.Deb.Depends, ", ")
	if depends == "" {
		depends = "libgtk-3-0 | libgtk-3-0t64"
	}
	maint := c.Config.App.Maintainer
	if maint == "" {
		maint = c.Config.App.Publisher
	}
	if maint == "" {
		maint = c.PackageName() + " maintainers <noreply@example.com>"
	}
	desc := c.Config.App.Description
	if desc == "" {
		desc = c.Project.Description
	}
	if desc == "" {
		desc = c.DisplayName()
	}
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("stage package tree (/opt, /usr/bin, .desktop, icon)", "准备安装目录（/opt、/usr/bin、.desktop、图标）"), Fn: func() error {
			if err := stageLinuxRoot(c, bundle, root); err != nil {
				return err
			}
			control := fmt.Sprintf("Package: %s\nVersion: %s\nArchitecture: %s\nMaintainer: %s\nInstalled-Size: %d\nDepends: %s\nSection: utils\nPriority: optional\nDescription: %s\n",
				c.PackageName(), debVersion(c), DebArch(linuxArch(c)), maint, dirSizeKB(root), depends, strings.ReplaceAll(desc, "\n", " "))
			if c.Config.App.Homepage != "" {
				control += "Homepage: " + c.Config.App.Homepage + "\n"
			}
			os.MkdirAll(filepath.Join(root, "DEBIAN"), 0o755)
			return os.WriteFile(filepath.Join(root, "DEBIAN", "control"), []byte(control), 0o644)
		}},
		{Desc: i18n.S("build .deb", "构建 .deb"), Cmd: &runner.Cmd{Name: "dpkg-deb", Args: []string{"--build", "--root-owner-group", root, tmp}}},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "Debian package", Arch: linuxArch(c)}}}, nil
}

func debVersion(c *Context) string {
	if c.BuildNumber != "" {
		return c.BuildName + "+" + c.BuildNumber
	}
	return c.BuildName
}

// Rpm builds an RPM with rpmbuild.
type Rpm struct{}

func (*Rpm) Name() string            { return "rpm" }
func (*Rpm) Platform() host.Platform { return host.Linux }
func (*Rpm) Formats() []string       { return []string{".rpm"} }
func (*Rpm) Optional() bool          { return true }
func (*Rpm) Description() string {
	return i18n.S("Fedora/RHEL/openSUSE package (rpmbuild)", "Fedora/RHEL/openSUSE 安装包（rpmbuild）")
}
func (*Rpm) Preflight(c *Context) []Issue {
	out := linuxPreflight(c)
	if c.Tools.Find("rpmbuild") == "" {
		out = append(out, fatal(i18n.S("rpmbuild not found", "找不到 rpmbuild"), installHint(c.Host, map[string]string{"apt": "rpm", "dnf": "rpm-build", "pacman": "rpm-tools"})))
	}
	return out
}
func (*Rpm) Steps(c *Context) ([]FlutterStep, error)                { return linuxStep(c), nil }
func (*Rpm) Locate(c *Context, p bool, _ time.Time) (Inputs, error) { return locateLinuxBundle(c, p) }

// RpmArch maps Flutter arch names to RPM ones.
func RpmArch(a string) string {
	if a == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}

func (*Rpm) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Linux, linuxArch(c), "", ".rpm")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("rpm")
	root := filepath.Join(stage, "root")
	top := filepath.Join(stage, "rpmbuild")
	spec := filepath.Join(stage, c.PackageName()+".spec")
	bundle := in["bundle"]
	pkg := c.PackageName()
	requires := strings.Join(c.Config.Linux.Rpm.Requires, ", ")
	if requires == "" {
		requires = "gtk3"
	}
	summary := c.Config.App.Description
	if summary == "" {
		summary = c.DisplayName()
	}
	version := strings.ReplaceAll(c.BuildName, "-", "_")
	release := c.BuildNumber
	if release == "" {
		release = "1"
	}
	arch := RpmArch(linuxArch(c))
	built := filepath.Join(top, "RPMS", arch, fmt.Sprintf("%s-%s-%s.%s.rpm", pkg, version, release, arch))
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("stage package tree and spec", "准备安装目录与 spec"), Fn: func() error {
			if err := stageLinuxRoot(c, bundle, root); err != nil {
				return err
			}
			s := fmt.Sprintf(`%%global debug_package %%{nil}
%%global __os_install_post %%{nil}
%%define _build_id_links none
Name: %s
Version: %s
Release: %s
Summary: %s
License: Proprietary
BuildArch: %s
AutoReqProv: no
Requires: %s

%%description
%s

%%install
mkdir -p %%{buildroot}
cp -a %s/. %%{buildroot}/

%%files
/opt/%s
/usr/bin/%s
/usr/share/applications/%s.desktop
`, pkg, version, release, strings.ReplaceAll(summary, "\n", " "), arch, requires, summary, root, pkg, pkg, pkg)
			if exists(filepath.Join(root, "usr", "share", "pixmaps", pkg+".png")) {
				s += "/usr/share/pixmaps/" + pkg + ".png\n"
			}
			return os.WriteFile(spec, []byte(s), 0o644)
		}},
		{Desc: i18n.S("build .rpm", "构建 .rpm"), Cmd: &runner.Cmd{Name: "rpmbuild", Args: []string{"-bb", "--quiet", "--define", "_topdir " + top, spec}}},
		moveOp(c, built, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "RPM package", Arch: linuxArch(c)}}}, nil
}

// AppImage builds a portable AppImage with appimagetool.
type AppImage struct{}

func (*AppImage) Name() string            { return "appimage" }
func (*AppImage) Platform() host.Platform { return host.Linux }
func (*AppImage) Formats() []string       { return []string{".AppImage"} }
func (*AppImage) Optional() bool          { return true }
func (*AppImage) Description() string {
	return i18n.S("Linux AppImage (appimagetool)", "Linux AppImage（appimagetool）")
}
func appimagetool(c *Context) string {
	if p := c.Config.Linux.AppImageTool; p != "" {
		return c.Project.Abs(p)
	}
	return c.Tools.Find("appimagetool")
}

func appimagetoolOrName(c *Context) string {
	if p := appimagetool(c); p != "" {
		return p
	}
	return "appimagetool"
}
func (*AppImage) Preflight(c *Context) []Issue {
	out := linuxPreflight(c)
	if appimagetool(c) == "" {
		out = append(out, fatal(i18n.S("appimagetool not found", "找不到 appimagetool"),
			"curl -Lo ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-$(uname -m).AppImage && chmod +x ~/.local/bin/appimagetool"))
	}
	if c.linuxIcon() == "" {
		out = append(out, fatal(i18n.S("AppImage needs a PNG icon", "AppImage 需要一个 PNG 图标"), i18n.S("set linux.icon in fpack.yaml", "在 fpack.yaml 中设置 linux.icon")))
	}
	return out
}
func (*AppImage) Steps(c *Context) ([]FlutterStep, error) { return linuxStep(c), nil }
func (*AppImage) Locate(c *Context, p bool, _ time.Time) (Inputs, error) {
	return locateLinuxBundle(c, p)
}
func (*AppImage) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Linux, linuxArch(c), "", ".AppImage")
	if err != nil {
		return nil, err
	}
	stage := c.Stage("appimage")
	appdir := filepath.Join(stage, c.AppName+".AppDir")
	tmp := filepath.Join(stage, filepath.Base(dst))
	bundle := in["bundle"]
	pkg := c.PackageName()
	bin := c.linuxBinary()
	arch := RpmArch(linuxArch(c))
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("stage AppDir", "准备 AppDir"), Fn: func() error {
			if err := pack.CopyDir(bundle, appdir); err != nil {
				return err
			}
			run := "#!/bin/sh\nHERE=\"$(dirname \"$(readlink -f \"$0\")\")\"\nexec \"$HERE/" + bin + "\" \"$@\"\n"
			if err := os.WriteFile(filepath.Join(appdir, "AppRun"), []byte(run), 0o755); err != nil {
				return err
			}
			if err := pack.CopyFile(c.linuxIcon(), filepath.Join(appdir, pkg+".png")); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(appdir, pkg+".desktop"), []byte(c.desktopEntry(bin, pkg)), 0o644)
		}},
		{Desc: i18n.S("build AppImage", "构建 AppImage"), Cmd: &runner.Cmd{Name: appimagetoolOrName(c), Args: []string{"--no-appstream", appdir, tmp}, Env: []string{"ARCH=" + arch, "APPIMAGE_EXTRACT_AND_RUN=1"}}},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "AppImage", Arch: linuxArch(c)}}}, nil
}
