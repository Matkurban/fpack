// Package doctor checks, per target platform, everything a build needs and
// prints exactly what is missing together with the command that fixes it.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
	"github.com/Matkurban/fpack/go/internal/version"
)

// Level of a check line.
type Level string

// Levels.
const (
	OK   Level = "ok"
	Warn Level = "warn"
	Fail Level = "fail"
	Info Level = "info"
	Skip Level = "skip"
)

// Line is one check result.
type Line struct {
	Level Level  `json:"level"`
	Text  string `json:"text"`
	Fix   string `json:"fix,omitempty"`
}

// Group is a platform section.
type Group struct {
	Platform  string   `json:"platform"`
	Title     string   `json:"title"`
	Targets   []string `json:"targets"`
	Buildable bool     `json:"buildable"`
	Lines     []Line   `json:"checks"`
}

// Report is the doctor result.
type Report struct {
	FpackVersion string   `json:"fpackVersion"`
	Host         string   `json:"host"`
	General      []Line   `json:"general"`
	Groups       []Group  `json:"platforms"`
	Ready        []string `json:"readyTargets"`
	Problems     int      `json:"problems"`
	Warnings     int      `json:"warnings"`
}

// Input for Run. Ctx may be nil when no project was found.
type Input struct {
	Ctx       *targets.Context
	SDK       *flutter.SDK
	SDKErr    error
	ProjErr   error
	Host      host.Host
	Tools     targets.Tools
	Only      []targets.Target // restrict to these targets
	WrapperOK string           // wrapper/core version info
}

// Run performs all checks.
func Run(in Input) *Report {
	r := &Report{FpackVersion: version.Version, Host: in.Host.OS + "/" + in.Host.Arch}
	// ---- general
	switch {
	case in.SDK != nil && in.SDK.Root != "":
		v := in.SDK.Version
		if v == "" {
			v = "?"
		}
		ch := ""
		if in.SDK.Channel != "" {
			ch = " (" + in.SDK.Channel + ")"
		}
		r.General = append(r.General, Line{OK, fmt.Sprintf("Flutter %s%s · %s  [%s]", v, ch, in.SDK.Root, in.SDK.Source), ""})
		if in.SDK.DartVersion != "" {
			r.General = append(r.General, Line{OK, "Dart " + in.SDK.DartVersion, ""})
		}
		if !in.SDK.AtLeast(3, 10) {
			r.General = append(r.General, Line{Warn, i18n.F("Flutter %s is old; fpack is tested with 3.22+", "Flutter %s 版本较旧；fpack 在 3.22+ 上测试", v), "flutter upgrade"})
		}
	default:
		r.General = append(r.General, Line{Fail, i18n.S("Flutter SDK not found", "找不到 Flutter SDK"),
			i18n.S("install Flutter (https://docs.flutter.dev/get-started/install) and add it to PATH, or pass --flutter <sdk> / set FPACK_FLUTTER", "安装 Flutter（https://docs.flutter.dev/get-started/install）并加入 PATH，或使用 --flutter <SDK路径> / 设置 FPACK_FLUTTER")})
	}
	if in.SDKErr != nil && (in.SDK == nil || in.SDK.Root == "") {
		if _, ok := in.SDKErr.(*flutter.NotSDKError); ok {
			r.General[len(r.General)-1].Text = in.SDKErr.Error()
		}
	}
	c := in.Ctx
	if c == nil {
		r.General = append(r.General, Line{Warn, i18n.S("not inside a Flutter project – only toolchain checks are shown", "当前不在 Flutter 项目中 —— 仅显示工具链检查"), i18n.S("cd into your app, or pass -C <dir>", "进入你的项目目录，或使用 -C <目录>")})
	} else {
		p := c.Project
		r.General = append(r.General, Line{OK, i18n.F("project %s %s+%s · %s", "项目 %s %s+%s · %s", p.Name, p.Version, p.BuildNumber, p.Root), ""})
		var plats []string
		for _, pl := range host.AllPlatforms {
			if p.Platforms[pl] {
				plats = append(plats, string(pl))
			}
		}
		r.General = append(r.General, Line{Info, i18n.S("platforms: ", "已启用平台：") + strings.Join(plats, ", "), ""})
		for _, d := range p.MissingPathDeps() {
			r.General = append(r.General, Line{Fail, i18n.F("path dependency %s → %s does not exist", "path 依赖 %s → %s 不存在", d.Name, d.Path),
				i18n.S("clone/copy that package to the expected location (relative to the project)", "把该包克隆/复制到对应位置（相对于项目目录）")})
		}
		if c.Config.File != "" {
			r.General = append(r.General, Line{OK, i18n.S("config: ", "配置：") + c.Rel(c.Config.File), ""})
		} else {
			r.General = append(r.General, Line{Info, i18n.S("no fpack.yaml – zero-config defaults are used", "没有 fpack.yaml —— 使用默认配置"), "fpack init"})
		}
	}
	if in.WrapperOK != "" {
		r.General = append(r.General, Line{Warn, in.WrapperOK, "dart pub global activate fpack"})
	}

	// ---- platform groups
	only := map[string]bool{}
	for _, t := range in.Only {
		only[t.Name()] = true
	}
	for _, pl := range host.AllPlatforms {
		var ts []targets.Target
		for _, t := range targets.ForPlatform(pl) {
			if len(only) == 0 || only[t.Name()] {
				ts = append(ts, t)
			}
		}
		if len(ts) == 0 {
			continue
		}
		g := Group{Platform: string(pl), Title: title(pl)}
		for _, t := range ts {
			g.Targets = append(g.Targets, t.Name())
		}
		ok, need := in.Host.CanBuild(pl)
		if !ok {
			g.Lines = append(g.Lines, Line{Skip, i18n.F("needs %s (this is %s)", "需要 %s（当前是 %s）", host.OSName(need), in.Host.OSName()), ""})
			r.Groups = append(r.Groups, g)
			continue
		}
		g.Buildable = true
		if c != nil && !c.Project.Platforms[pl] {
			g.Lines = append(g.Lines, Line{Skip, i18n.F("project has no %s/ folder", "项目中没有 %s/ 目录", pl), "flutter create --platforms=" + string(pl) + " ."})
			g.Buildable = false
			r.Groups = append(r.Groups, g)
			continue
		}
		g.Lines = append(g.Lines, platformInfo(pl, in)...)
		seen := map[string]bool{}
		for _, l := range g.Lines {
			seen[l.Text] = true
		}
		readyHere := map[string]bool{}
		if c != nil {
			for _, t := range ts {
				fatal := false
				for _, is := range t.Preflight(c) {
					lv := Warn
					if is.Fatal {
						lv = Fail
						fatal = true
						if t.Optional() {
							lv = Warn
						}
					}
					text := is.Msg
					if t.Optional() && is.Fatal {
						text += i18n.F("  (needed for %s)", "（%s 需要）", t.Name())
					}
					if !seen[text] {
						seen[text] = true
						g.Lines = append(g.Lines, Line{lv, text, is.Fix})
					}
				}
				if !fatal {
					readyHere[t.Name()] = true
				}
			}
		}
		for _, t := range ts {
			if readyHere[t.Name()] {
				r.Ready = append(r.Ready, t.Name())
			}
		}
		r.Groups = append(r.Groups, g)
	}
	count := func(lines []Line) {
		for _, l := range lines {
			switch l.Level {
			case Fail:
				r.Problems++
			case Warn:
				r.Warnings++
			}
		}
	}
	count(r.General)
	for _, g := range r.Groups {
		count(g.Lines)
	}
	return r
}

func title(p host.Platform) string {
	return map[host.Platform]string{host.Android: "Android", host.IOS: "iOS", host.MacOS: "macOS", host.Windows: "Windows", host.Linux: "Linux", host.Web: "Web"}[p]
}

var xcodeVer = regexp.MustCompile(`Xcode (\S+)`)

func platformInfo(pl host.Platform, in Input) []Line {
	t := in.Tools
	var out []Line
	switch pl {
	case host.Android:
		var e *targets.AndroidEnv
		if in.Ctx != nil {
			e = in.Ctx.Android()
		} else {
			e = targets.DetectAndroidEnv(t)
		}
		if e.SDK != "" {
			out = append(out, Line{OK, fmt.Sprintf("Android SDK · %s  [%s]", e.SDK, e.SDKSource), ""})
			if e.Licenses {
				out = append(out, Line{OK, i18n.S("Android SDK licenses accepted", "已接受 Android SDK 许可"), ""})
			}
		}
		if e.Java != "" {
			v := "?"
			if e.JavaVersion > 0 {
				v = fmt.Sprint(e.JavaVersion)
			}
			lv := OK
			if e.JavaVersion > 0 && e.JavaVersion < 17 {
				lv = Warn
			}
			out = append(out, Line{lv, fmt.Sprintf("Java %s · %s  [%s]", v, e.Java, e.JavaSource), ""})
		}
		if in.Ctx != nil {
			s := in.Ctx.Signing
			switch {
			case s.Enabled && s.FromBase64():
				out = append(out, Line{OK, i18n.S("release signing: keystore from FPACK_ANDROID_KEYSTORE_BASE64, alias ", "release 签名：keystore 来自 FPACK_ANDROID_KEYSTORE_BASE64，别名 ") + s.KeyAlias, ""})
			case s.Enabled:
				out = append(out, Line{OK, i18n.F("release signing: %s (alias %s) injected by fpack", "release 签名：%s（别名 %s），由 fpack 注入", in.Ctx.Rel(s.StoreFile), s.KeyAlias), ""})
			case in.Ctx.Project.AndroidKeyProperties:
				out = append(out, Line{OK, i18n.S("release signing: project's android/key.properties", "release 签名：使用项目的 android/key.properties"), ""})
			}
			if fl := in.Ctx.Project.AndroidFlavors; len(fl) > 0 {
				out = append(out, Line{Info, "flavors: " + strings.Join(fl, ", "), ""})
			}
		}
	case host.IOS, host.MacOS:
		if t.Find("xcodebuild") != "" {
			v, _ := t.Probe("xcodebuild", "-version")
			if m := xcodeVer.FindStringSubmatch(v); m != nil {
				out = append(out, Line{OK, "Xcode " + m[1], ""})
			}
		}
		if p := t.Find("pod"); p != "" {
			v, _ := t.Probe(p, "--version")
			out = append(out, Line{OK, "CocoaPods " + strings.TrimSpace(lastLine(v)), ""})
		}
		all, _ := targets.AllIdentities(t)
		var ids, revoked []string
		for _, id := range all {
			if id.Problem == "" {
				ids = append(ids, id.Name)
			} else {
				revoked = append(revoked, id.Name+" ("+id.Problem+")")
			}
		}
		if pl == host.IOS {
			var dist, dev int
			for _, id := range ids {
				if strings.HasPrefix(id, "Apple Distribution") || strings.HasPrefix(id, "iPhone Distribution") {
					dist++
				} else if strings.HasPrefix(id, "Apple Development") || strings.HasPrefix(id, "iPhone Developer") {
					dev++
				}
			}
			if dist+dev > 0 {
				out = append(out, Line{OK, i18n.F("signing certificates: %d distribution, %d development", "签名证书：%d 个发布证书，%d 个开发证书", dist, dev), ""})
			}
			if len(revoked) > 0 {
				out = append(out, Line{Info, i18n.F("%d revoked/invalid certificate(s) in the keychain are ignored: %s", "钥匙串中有 %d 个已吊销/无效的证书（已忽略）：%s", len(revoked), strings.Join(revoked, "; ")),
					i18n.S("you can delete them in Keychain Access → My Certificates", "可在「钥匙串访问 → 我的证书」中删除")})
			}
			if in.Ctx != nil {
				ios := in.Ctx.Config.IOS
				if in.Ctx.Project.IOSTeam != "" {
					out = append(out, Line{OK, "DEVELOPMENT_TEAM " + in.Ctx.Project.IOSTeam, ""})
				}
				switch {
				case !in.Ctx.Config.IOSCodesign():
					out = append(out, Line{Info, i18n.S("IPA: unsigned (ios.codesign: false)", "IPA：未签名（ios.codesign: false）"), ""})
				case ios.ExportOptionsPlist != "":
					out = append(out, Line{Info, "IPA export: " + ios.ExportOptionsPlist, ""})
				default:
					m := ios.ExportMethod
					if m == "" {
						m = "app-store (Flutter default)"
					}
					out = append(out, Line{Info, i18n.S("IPA export method: ", "IPA 导出方式：") + m, ""})
				}
			}
		} else if in.Ctx != nil {
			m := in.Ctx.Mac
			if m.Enabled {
				src := ""
				if m.Source != "" {
					src = "  [" + m.Source + "]"
				}
				out = append(out, Line{OK, i18n.S("Developer ID signing: ", "Developer ID 签名：") + m.IdentityLabel() + src, ""})
				if m.Notarize {
					out = append(out, Line{Info, i18n.F("notarization: keychain profile %q (check: xcrun notarytool history --keychain-profile %s)", "公证：钥匙串配置 %q（检查：xcrun notarytool history --keychain-profile %s）", m.Profile, m.Profile), ""})
				}
			} else {
				out = append(out, Line{Info, i18n.S("Developer ID signing: off (Xcode signing is used; DMG will be unsigned)", "Developer ID 签名：关闭（使用 Xcode 签名；DMG 不签名）"), ""})
			}
			if t.Find("create-dmg") != "" {
				out = append(out, Line{OK, "create-dmg", ""})
			} else if in.Host.OS == "darwin" {
				out = append(out, Line{Info, i18n.S("DMG via hdiutil (optional nicer layout: brew install create-dmg)", "DMG 使用 hdiutil 生成（可选更美观的布局：brew install create-dmg）"), ""})
			}
		}
	case host.Windows:
		vswhere := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft Visual Studio", "Installer", "vswhere.exe")
		if v, ok := t.Probe(vswhere, "-latest", "-property", "displayName"); ok && v != "" {
			out = append(out, Line{OK, v, ""})
		}
		if p := t.Find("iscc"); p != "" {
			out = append(out, Line{OK, "Inno Setup · " + p, ""})
		}
	case host.Linux:
		for _, tool := range []string{"clang++", "cmake", "ninja", "pkg-config", "dpkg-deb", "rpmbuild", "appimagetool"} {
			if p := t.Find(tool); p != "" {
				out = append(out, Line{OK, tool + " · " + p, ""})
			}
		}
	case host.Web:
		out = append(out, Line{OK, i18n.S("no extra tools needed", "无需额外工具"), ""})
	}
	return out
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// Print renders the report.
func Print(u *ui.UI, r *Report, projErr error) {
	u.Println(u.Bold("fpack doctor") + u.Dim("  ·  fpack "+r.FpackVersion+"  ·  "+r.Host))
	u.Blank()
	u.Title(i18n.S("General", "通用"))
	printLines(u, r.General)
	for _, g := range r.Groups {
		u.Blank()
		u.Title(fmt.Sprintf("%s  %s", g.Title, u.Dim("("+strings.Join(g.Targets, ", ")+")")))
		printLines(u, g.Lines)
	}
	u.Blank()
	if len(r.Ready) > 0 {
		u.Println(u.Green("✓ ") + i18n.S("ready to build here: ", "当前可构建：") + u.Bold(strings.Join(r.Ready, " ")))
	}
	msg := i18n.F("%d problem(s), %d warning(s)", "%d 个问题，%d 个警告", r.Problems, r.Warnings)
	if r.Problems > 0 {
		u.Println(u.Red("✗ " + msg))
	} else if r.Warnings > 0 {
		u.Println(u.Yellow("! " + msg))
	} else {
		u.Println(u.Green("✓ " + i18n.S("everything looks good", "一切正常")))
	}
}

func printLines(u *ui.UI, lines []Line) {
	for _, l := range lines {
		text := l.Text
		switch l.Level {
		case OK:
			u.Success(text)
		case Warn:
			u.Warn(text)
		case Fail:
			u.Fail(text)
		case Skip:
			u.Skip(u.Dim(text))
		default:
			u.Info(u.Dim("• ") + text)
		}
		if l.Fix != "" && l.Level != OK {
			u.Hint(l.Fix)
		}
	}
}

var _ = project.SplitVersion
