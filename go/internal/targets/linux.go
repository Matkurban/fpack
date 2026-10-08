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

// desktopEntry renders the .desktop file. extra lines are appended (e.g.
// X-AppImage-Version).
func (c *Context) desktopEntry(exec, icon string, extra ...string) string {
	l := c.Config.Linux
	cats := []string(l.Categories)
	if len(cats) == 0 {
		cats = []string{"Utility"}
	}
	join := func(v []string) string {
		var out []string
		for _, x := range v {
			if x = strings.Trim(strings.TrimSpace(x), ";"); x != "" {
				out = append(out, x)
			}
		}
		return strings.Join(out, ";") + ";"
	}
	s := "[Desktop Entry]\nType=Application\nVersion=1.5\nName=" + c.DisplayName() + "\n"
	if l.GenericName != "" {
		s += "GenericName=" + l.GenericName + "\n"
	}
	s += "Comment=" + c.Description() + "\n"
	s += "Exec=" + exec + " %U\n"
	if icon != "" {
		s += "Icon=" + icon + "\n"
	}
	s += "Terminal=false\nCategories=" + join(cats) + "\n"
	if len(l.Keywords) > 0 {
		s += "Keywords=" + join(l.Keywords) + "\n"
	}
	if len(l.MimeTypes) > 0 {
		s += "MimeType=" + join(l.MimeTypes) + "\n"
	}
	s += "StartupWMClass=" + orDefault(l.StartupWMClass, c.linuxBinary()) + "\n"
	for _, e := range extra {
		s += e + "\n"
	}
	return s
}

// linuxPrefix is the install directory of the bundle (linux.prefix, else
// /opt/<package>).
func (c *Context) linuxPrefix() string {
	if p := strings.TrimRight(c.Config.Linux.Prefix, "/"); p != "" {
		return p
	}
	return "/opt/" + c.PackageName()
}

// iconSizes returns the hicolor sizes to generate (never upscaled).
func (c *Context) iconSizes(src string) []int {
	sizes := c.Config.Linux.IconSizes
	if len(sizes) == 0 {
		sizes = []int{16, 32, 48, 64, 128, 256, 512}
	}
	w, h, err := pack.PNGSize(src)
	if err != nil {
		return nil
	}
	max := w
	if h > max {
		max = h
	}
	var out []int
	for _, s := range sizes {
		if s <= max {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		out = []int{max}
	}
	return out
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

// stageLinuxRoot lays out <prefix>, /usr/bin/<pkg>, the desktop file,
// hicolor icons, metainfo and the copyright file. It returns the installed
// paths (relative to root, with leading /) for package file lists.
func stageLinuxRoot(c *Context, bundle, root string) ([]string, error) {
	pkg := c.PackageName()
	prefix := c.linuxPrefix()
	opt := filepath.Join(root, filepath.FromSlash(prefix))
	if err := os.MkdirAll(filepath.Dir(opt), 0o755); err != nil {
		return nil, err
	}
	if err := pack.CopyDir(bundle, opt); err != nil {
		return nil, err
	}
	files := []string{prefix}
	bin := filepath.Join(root, "usr", "bin")
	os.MkdirAll(bin, 0o755)
	if err := os.Symlink(prefix+"/"+c.linuxBinary(), filepath.Join(bin, pkg)); err != nil {
		return nil, err
	}
	files = append(files, "/usr/bin/"+pkg)
	icon := ""
	if src := c.linuxIcon(); src != "" {
		for _, sz := range c.iconSizes(src) {
			rel := fmt.Sprintf("/usr/share/icons/hicolor/%dx%d/apps/%s.png", sz, sz, pkg)
			if err := pack.ResizePNG(src, filepath.Join(root, filepath.FromSlash(rel)), sz); err != nil {
				return nil, err
			}
			os.Chmod(filepath.Join(root, filepath.FromSlash(rel)), 0o644)
			files = append(files, rel)
		}
		pix := "/usr/share/pixmaps/" + pkg + ".png"
		if err := pack.CopyFile(src, filepath.Join(root, filepath.FromSlash(pix))); err != nil {
			return nil, err
		}
		os.Chmod(filepath.Join(root, filepath.FromSlash(pix)), 0o644)
		files = append(files, pix)
		icon = pkg
	}
	desktop := "/usr/share/applications/" + pkg + ".desktop"
	os.MkdirAll(filepath.Join(root, "usr", "share", "applications"), 0o755)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(desktop)), []byte(c.desktopEntry(pkg, icon)), 0o644); err != nil {
		return nil, err
	}
	files = append(files, desktop)
	if m := c.Config.Linux.Metainfo; m != "" {
		rel := "/usr/share/metainfo/" + filepath.Base(m)
		if err := pack.CopyFile(c.Project.Abs(m), filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return nil, err
		}
		os.Chmod(filepath.Join(root, filepath.FromSlash(rel)), 0o644)
		files = append(files, rel)
	}
	doc := "/usr/share/doc/" + pkg + "/copyright"
	os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(doc))), 0o755)
	cp := fmt.Sprintf("Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\nUpstream-Name: %s\n", c.DisplayName())
	if h := c.Config.App.Homepage; h != "" {
		cp += "Source: " + h + "\n"
	}
	cp += fmt.Sprintf("\nFiles: *\nCopyright: %s\nLicense: %s\n", strings.TrimPrefix(strings.TrimPrefix(c.Copyright(), "©"), "(c)"), c.License())
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(doc)), []byte(cp), 0o644); err != nil {
		return nil, err
	}
	files = append(files, doc)
	return files, nil
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
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.F("stage package tree (%s, /usr/bin, .desktop, icons)", "准备安装目录（%s、/usr/bin、.desktop、图标）", c.linuxPrefix()), Fn: func() error {
			if _, err := stageLinuxRoot(c, bundle, root); err != nil {
				return err
			}
			os.MkdirAll(filepath.Join(root, "DEBIAN"), 0o755)
			if err := os.WriteFile(filepath.Join(root, "DEBIAN", "control"), []byte(DebControl(c, dirSizeKB(root))), 0o644); err != nil {
				return err
			}
			d := c.Config.Linux.Deb
			for name, src := range map[string]string{"preinst": d.Preinst, "postinst": d.Postinst, "prerm": d.Prerm, "postrm": d.Postrm} {
				if src == "" {
					continue
				}
				if err := installScript(c.Project.Abs(src), filepath.Join(root, "DEBIAN", name)); err != nil {
					return err
				}
			}
			return nil
		}},
		{Desc: i18n.S("build .deb", "构建 .deb"), Cmd: &runner.Cmd{Name: "dpkg-deb", Args: []string{"--build", "--root-owner-group", root, tmp}}},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "Debian package", Arch: linuxArch(c)}}}, nil
}

// installScript copies a maintainer script with LF endings and mode 0755.
func installScript(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	b = []byte(strings.ReplaceAll(string(b), "\r\n", "\n"))
	if !strings.HasPrefix(string(b), "#!") {
		b = append([]byte("#!/bin/sh\nset -e\n"), b...)
	}
	return os.WriteFile(dst, b, 0o755)
}

// DebControl renders DEBIAN/control.
func DebControl(c *Context, sizeKB int64) string {
	d := c.Config.Linux.Deb
	depends := []string(d.Depends)
	if len(depends) == 0 {
		depends = []string{"libgtk-3-0 | libgtk-3-0t64"}
	}
	s := fmt.Sprintf("Package: %s\nVersion: %s\nArchitecture: %s\nMaintainer: %s\nInstalled-Size: %d\nDepends: %s\n",
		c.PackageName(), debVersion(c), DebArch(linuxArch(c)), c.Maintainer(), sizeKB, strings.Join(depends, ", "))
	for _, f := range []struct {
		k string
		v []string
	}{{"Recommends", d.Recommends}, {"Suggests", d.Suggests}, {"Conflicts", d.Conflicts}} {
		if len(f.v) > 0 {
			s += f.k + ": " + strings.Join(f.v, ", ") + "\n"
		}
	}
	s += "Section: " + orDefault(d.Section, "utils") + "\nPriority: " + orDefault(d.Priority, "optional") + "\n"
	if h := c.Config.App.Homepage; h != "" {
		s += "Homepage: " + h + "\n"
	}
	s += "Description: " + c.Description() + "\n"
	if c.DisplayName() != c.Description() {
		s += " " + c.DisplayName() + " " + c.BuildName + "\n"
	}
	return s
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
	version, release := rpmVersion(c)
	arch := RpmArch(linuxArch(c))
	built := filepath.Join(top, "RPMS", arch, fmt.Sprintf("%s-%s-%s.%s.rpm", pkg, version, release, arch))
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.S("stage package tree and spec", "准备安装目录与 spec"), Fn: func() error {
			files, err := stageLinuxRoot(c, bundle, root)
			if err != nil {
				return err
			}
			s, err := RpmSpec(c, root, files)
			if err != nil {
				return err
			}
			return os.WriteFile(spec, []byte(s), 0o644)
		}},
		{Desc: i18n.S("build .rpm", "构建 .rpm"), Cmd: &runner.Cmd{Name: "rpmbuild", Args: []string{"-bb", "--quiet", "--define", "_topdir " + top, spec}}},
		moveOp(c, built, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "RPM package", Arch: linuxArch(c)}}}, nil
}

func rpmVersion(c *Context) (string, string) {
	version := strings.ReplaceAll(c.BuildName, "-", "_")
	release := c.BuildNumber
	if release == "" {
		release = "1"
	}
	return version, release
}

// rpmEscape protects % in free text from macro expansion.
func rpmEscape(s string) string { return strings.ReplaceAll(s, "%", "%%") }

// RpmSpec renders the spec file for an already staged root.
func RpmSpec(c *Context, root string, files []string) (string, error) {
	r := c.Config.Linux.Rpm
	requires := []string(r.Requires)
	if len(requires) == 0 {
		requires = []string{"gtk3"}
	}
	version, release := rpmVersion(c)
	var b strings.Builder
	fmt.Fprintf(&b, "%%global debug_package %%{nil}\n%%global __os_install_post %%{nil}\n%%define _build_id_links none\n")
	fmt.Fprintf(&b, "Name: %s\nVersion: %s\nRelease: %s\nSummary: %s\nLicense: %s\nGroup: %s\n",
		c.PackageName(), version, release, rpmEscape(c.Description()), rpmEscape(orDefault(r.License, c.License())), orDefault(r.Group, "Applications/Internet"))
	if h := c.Config.App.Homepage; h != "" {
		fmt.Fprintf(&b, "URL: %s\n", h)
	}
	if p := c.Publisher(); p != "" {
		fmt.Fprintf(&b, "Vendor: %s\n", rpmEscape(p))
	}
	fmt.Fprintf(&b, "Packager: %s\nBuildArch: %s\nAutoReqProv: no\nRequires: %s\n\n", rpmEscape(c.Maintainer()), RpmArch(linuxArch(c)), strings.Join(requires, ", "))
	fmt.Fprintf(&b, "%%description\n%s\n\n%%install\nmkdir -p %%{buildroot}\ncp -a %s/. %%{buildroot}/\n\n", rpmEscape(c.Description()), root)
	for _, sc := range []struct{ name, path string }{{"pre", r.Pre}, {"post", r.Post}, {"preun", r.Preun}, {"postun", r.Postun}} {
		if sc.path == "" {
			continue
		}
		body, err := os.ReadFile(c.Project.Abs(sc.path))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%%%s\n%s\n\n", sc.name, strings.TrimSpace(strings.ReplaceAll(string(body), "\r\n", "\n")))
	}
	b.WriteString("%files\n")
	for _, f := range files {
		if strings.HasPrefix(f, "/usr/share/doc/") {
			fmt.Fprintf(&b, "%%doc %s\n", f)
			continue
		}
		b.WriteString(f + "\n")
	}
	return b.String(), nil
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
	ai := c.Config.Linux.AppImage
	args := []string{"--no-appstream"}
	if u := ai.UpdateInformation; u != "" {
		args = append(args, "--updateinformation", u)
	}
	args = append(append(args, ai.ExtraArgs...), appdir, tmp)
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
			icon := c.linuxIcon()
			if err := pack.CopyFile(icon, filepath.Join(appdir, pkg+".png")); err != nil {
				return err
			}
			if err := pack.CopyFile(icon, filepath.Join(appdir, ".DirIcon")); err != nil {
				return err
			}
			sizes := c.iconSizes(icon)
			for _, sz := range sizes {
				if err := pack.ResizePNG(icon, filepath.Join(appdir, "usr", "share", "icons", "hicolor", fmt.Sprintf("%dx%d", sz, sz), "apps", pkg+".png"), sz); err != nil {
					return err
				}
			}
			if m := c.Config.Linux.Metainfo; m != "" {
				if err := pack.CopyFile(c.Project.Abs(m), filepath.Join(appdir, "usr", "share", "metainfo", filepath.Base(m))); err != nil {
					return err
				}
			}
			entry := c.desktopEntry(bin, pkg, "X-AppImage-Version="+debVersion(c))
			return os.WriteFile(filepath.Join(appdir, pkg+".desktop"), []byte(entry), 0o644)
		}},
		{Desc: i18n.S("build AppImage", "构建 AppImage"), Cmd: &runner.Cmd{Name: appimagetoolOrName(c), Args: args, Env: []string{"ARCH=" + arch, "APPIMAGE_EXTRACT_AND_RUN=1"}}},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "AppImage", Arch: linuxArch(c)}}}, nil
}
