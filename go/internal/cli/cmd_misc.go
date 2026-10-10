package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Matkurban/fpack/go/internal/build"
	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/doctor"
	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/runner"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
	"github.com/Matkurban/fpack/go/internal/version"
)

// ------------------------------------------------------------------ doctor --

func doctorCommand() *command {
	c := &command{name: "doctor", aliases: []string{"check"}, en: "check prerequisites per target and show how to fix what is missing", zh: "按目标检查环境，并给出缺失项的修复命令"}
	c.help = func() string {
		return cmdHelp(c, i18n.S("fpack doctor [targets...]", "fpack doctor [目标...]"), "  fpack doctor\n  fpack doctor apk ipa dmg\n  fpack doctor --json\n")
	}
	c.run = func(e *Env, p *parsed) int {
		u := newUI(e, p)
		var only []targets.Target
		for _, n := range p.pos {
			t, ok := targets.Get(n)
			if !ok {
				u.Errorf("%s", i18n.F("unknown target %q", "未知目标 %q", n))
				return build.ExitUsage
			}
			only = append(only, t)
		}
		h := host.Current()
		ctxT, err := build.NewContext(build.Options{ProjectDir: p.s("project"), ConfigPath: p.s("config"), FlutterPath: p.s("flutter"), Getenv: e.Getenv})
		var nf *project.ErrNotFound
		in := doctor.Input{Host: h, Tools: targets.SystemTools{}, Only: only}
		switch {
		case err == nil:
			in.Ctx = ctxT
			in.SDK = ctxT.SDK
		case errors.As(err, &nf):
			in.ProjErr = err
			in.SDK, in.SDKErr = flutter.Locate(".", []flutter.Candidate{{Path: p.s("flutter"), Source: "--flutter"}, {Path: e.Getenv("FPACK_FLUTTER"), Source: "FPACK_FLUTTER"}})
		default:
			return contextError(u, err)
		}
		if in.SDK != nil && in.SDK.Root != "" {
			in.SDK.QueryVersion()
		}
		if w := e.Getenv("FPACK_WRAPPER_VERSION"); w != "" && w != version.Version {
			in.WrapperOK = i18n.F("Dart wrapper %s ≠ core %s", "Dart 包装器 %s ≠ 核心 %s", w, version.Version)
		}
		r := doctor.Run(in)
		doctor.Print(u, r, in.ProjErr)
		if p.b("json") {
			printJSON(e, r)
		}
		if r.Problems > 0 {
			return build.ExitFailed
		}
		return 0
	}
	return c
}

// -------------------------------------------------------------------- list --

type listRow struct {
	Target      string   `json:"target"`
	Platform    string   `json:"platform"`
	Formats     []string `json:"formats"`
	Description string   `json:"description"`
	Buildable   bool     `json:"buildable"`
	Reason      string   `json:"reason,omitempty"`
}

func listCommand() *command {
	c := &command{name: "list", aliases: []string{"ls", "targets"}, en: "list targets, output formats and what this machine can build", zh: "列出所有目标、输出格式，以及本机能构建哪些"}
	c.help = func() string { return cmdHelp(c, "fpack list", "  fpack list\n  fpack list --json\n") }
	c.run = func(e *Env, p *parsed) int {
		u := newUI(e, p)
		h := host.Current()
		ctxT, err := build.NewContext(build.Options{ProjectDir: p.s("project"), ConfigPath: p.s("config"), FlutterPath: p.s("flutter"), Getenv: e.Getenv})
		var nf *project.ErrNotFound
		if err != nil && !errors.As(err, &nf) {
			return contextError(u, err)
		}
		var rows []listRow
		var table [][]string
		for _, t := range targets.All() {
			r := listRow{Target: t.Name(), Platform: string(t.Platform()), Formats: t.Formats(), Description: t.Description(), Buildable: true}
			if ok, need := h.CanBuild(t.Platform()); !ok {
				r.Buildable, r.Reason = false, i18n.F("needs %s", "需要 %s", host.OSName(need))
			} else if ctxT != nil && !ctxT.Project.Platforms[t.Platform()] {
				r.Buildable, r.Reason = false, i18n.F("no %s/ in project", "项目无 %s/", t.Platform())
			} else if ctxT != nil {
				for _, is := range t.Preflight(ctxT) {
					if is.Fatal || is.NotReady {
						r.Buildable, r.Reason = false, firstLine(is.Msg)
						break
					}
				}
			}
			rows = append(rows, r)
			status := u.Green("✓")
			if !r.Buildable {
				status = u.Dim("– " + ui.Truncate(r.Reason, 40))
			}
			table = append(table, []string{u.Bold(r.Target), r.Platform, strings.Join(r.Formats, " "), status, u.Dim(r.Description)})
		}
		if p.b("json") {
			printJSON(e, rows)
			return 0
		}
		title := i18n.F("Targets on %s/%s", "%s/%s 上的目标", h.OSName(), h.Arch)
		if ctxT != nil {
			title += "  ·  " + ctxT.Project.Name
		}
		u.Title(title)
		u.Table([]string{i18n.S("TARGET", "目标"), i18n.S("PLATFORM", "平台"), i18n.S("OUTPUT", "输出"), i18n.S("HERE", "本机"), i18n.S("DESCRIPTION", "说明")}, table)
		if ctxT != nil {
			for _, pl := range []host.Platform{host.Android, host.IOS, host.MacOS} {
				if fl := ctxT.Project.Flavors(pl); len(fl) > 0 {
					u.Info(u.Dim(i18n.F("%s flavors: %s", "%s flavor：%s", pl, strings.Join(fl, ", "))))
				}
			}
		}
		u.Blank()
		u.Info(u.Dim(i18n.S("Details: fpack doctor   ·   Build: fpack build <target…> | --all", "详情：fpack doctor   ·   构建：fpack build <目标…> | --all")))
		return 0
	}
	return c
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ------------------------------------------------------------------- clean --

func cleanCommand() *command {
	c := &command{name: "clean", en: "remove fpack's temporary files (and optionally dist/ or flutter build output)", zh: "删除 fpack 的临时文件（可选删除 dist/ 或 flutter 构建产物）",
		flags: []flagSpec{
			{names: []string{"--dist"}, kind: kBool, en: "also delete the output directory (asks first)", zh: "同时删除输出目录（会先确认）"},
			{names: []string{"--flutter"}, kind: kBool, en: "also run `flutter clean`", zh: "同时运行 `flutter clean`"},
			{names: []string{"--all"}, kind: kBool, en: "--dist + --flutter-clean", zh: "等同 --dist + --flutter-clean"},
			{names: []string{"--dry-run", "-n"}, kind: kBool, en: "show what would be deleted", zh: "只显示将删除的内容"},
		}}
	// --flutter conflicts with the global --flutter <sdk>: use a distinct key.
	c.flags[1].names = []string{"--flutter-clean"}
	c.help = func() string {
		return cmdHelp(c, "fpack clean [--dist] [--flutter-clean] [--all]", "  fpack clean\n  fpack clean --dist --yes\n  fpack clean --all\n")
	}
	c.run = func(e *Env, p *parsed) int {
		u := newUI(e, p)
		ctxT, err := build.NewContext(build.Options{ProjectDir: p.s("project"), ConfigPath: p.s("config"), FlutterPath: p.s("flutter"), Getenv: e.Getenv})
		if err != nil {
			return contextError(u, err)
		}
		dry := p.b("dry-run")
		var paths []string
		paths = append(paths, ctxT.WorkDir)
		if p.b("dist") || p.b("all") {
			if d := staticOutputRoot(ctxT); d != "" {
				paths = append(paths, d)
			}
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err != nil {
				continue
			}
			if path != ctxT.WorkDir && !dry && !p.b("yes") {
				if !confirm(e, u, i18n.F("Delete %s with all artifacts?", "确定删除 %s 及其中所有产物？", ctxT.Rel(path))) {
					u.Skip(i18n.F("kept %s", "保留 %s", ctxT.Rel(path)))
					continue
				}
			}
			if dry {
				u.Info(i18n.S("would delete ", "将删除 ") + ctxT.Rel(path))
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				u.Fail(err.Error())
				return build.ExitFailed
			}
			u.Success(i18n.S("deleted ", "已删除 ") + ctxT.Rel(path))
		}
		if p.b("flutter-clean") || p.b("all") {
			cmd := runner.Cmd{Name: ctxT.SDK.Flutter, Args: []string{"clean"}, Dir: ctxT.Project.Root}
			if dry {
				u.Info("$ " + cmd.String())
			} else {
				u.Step(cmd.String())
				ctx, stop := signalContext(u)
				defer stop()
				if _, err := runner.Run(ctx, cmd, runner.Options{Stream: u.Writer()}); err != nil {
					u.Fail(err.Error())
					return build.ExitFailed
				}
			}
		}
		return 0
	}
	return c
}

// staticOutputRoot returns the non-templated prefix of output.dir, e.g.
// "dist" for "dist/{version}{+build}"; never the project root itself.
func staticOutputRoot(c *targets.Context) string {
	tmpl := c.Config.OutputDir()
	if i := strings.Index(tmpl, "{"); i >= 0 {
		tmpl = tmpl[:i]
	}
	tmpl = strings.TrimRight(tmpl, `/\`)
	if tmpl == "" || tmpl == "." {
		return c.OutDir
	}
	p := c.Project.Abs(tmpl)
	if p == c.Project.Root || !strings.HasPrefix(p, c.Project.Root) {
		return c.OutDir
	}
	return p
}

func confirm(e *Env, u *ui.UI, q string) bool {
	if f, ok := e.Stdin.(*os.File); !ok || !isTTY(f) {
		u.Warn(i18n.S("not a terminal – pass --yes to confirm", "非交互终端 —— 请加 --yes 确认"))
		return false
	}
	u.Println(q + " [y/N] ")
	line, _ := bufio.NewReader(e.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes" || line == "是"
}

// -------------------------------------------------------------------- init --

func initCommand() *command {
	c := &command{name: "init", en: "create a commented fpack.yaml for this project (interactive, or --yes for defaults)", zh: "为项目生成带注释的 fpack.yaml（交互式，或用 --yes 使用默认值）",
		flags: []flagSpec{{names: []string{"--force", "-f"}, kind: kBool, en: "overwrite an existing fpack.yaml", zh: "覆盖已有的 fpack.yaml"}}}
	c.help = func() string { return cmdHelp(c, "fpack init [--yes] [--force]", "  fpack init\n  fpack init --yes\n") }
	c.run = runInit
	return c
}

type prompter struct {
	r           *bufio.Reader
	u           *ui.UI
	interactive bool
}

func (pr *prompter) ask(q, def string) string {
	if !pr.interactive {
		return def
	}
	d := ""
	if def != "" {
		d = " " + pr.u.Dim("["+def+"]")
	}
	fmt.Fprint(pr.u.Writer(), pr.u.Cyan("? ")+q+d+" ")
	line, err := pr.r.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && err != io.EOF {
		return def
	}
	if line == "" {
		return def
	}
	return line
}

func runInit(e *Env, p *parsed) int {
	u := newUI(e, p)
	ctxT, err := build.NewContext(build.Options{ProjectDir: p.s("project"), FlutterPath: p.s("flutter"), Getenv: func(string) string { return "" }})
	if err != nil {
		return contextError(u, err)
	}
	proj := ctxT.Project
	dest := filepath.Join(proj.Root, "fpack.yaml")
	existing, readErr := os.ReadFile(dest)
	exists := readErr == nil
	if exists && !p.b("force") {
		u.Warn(i18n.F("%s already exists – it will not be overwritten; writing fpack.yaml.new instead (use --force to replace it)", "%s 已存在 —— 不会覆盖，改为生成 fpack.yaml.new（使用 --force 可直接替换）", ctxT.Rel(dest)))
	}
	interactive := !p.b("yes")
	if f, ok := e.Stdin.(*os.File); interactive && (!ok || !isTTY(f)) {
		interactive = false
		u.Info(u.Dim(i18n.S("(stdin is not a terminal – using defaults)", "（stdin 不是终端 —— 使用默认值）")))
	}
	pr := &prompter{r: bufio.NewReader(e.Stdin), u: u, interactive: interactive}
	h := host.Current()

	u.Title(i18n.F("fpack init · %s %s+%s", "fpack init · %s %s+%s", proj.Name, proj.Version, proj.BuildNumber))
	var plats []string
	for _, pl := range host.AllPlatforms {
		if proj.Platforms[pl] {
			plats = append(plats, string(pl))
		}
	}
	u.Info(i18n.S("platforms: ", "平台：") + strings.Join(plats, ", "))
	for _, pl := range []host.Platform{host.Android, host.IOS, host.MacOS} {
		if fl := proj.Flavors(pl); len(fl) > 0 {
			u.Info(i18n.F("%s flavors: %s", "%s flavor：%s", pl, strings.Join(fl, ", ")))
		}
	}
	u.Blank()

	// Suggested default targets: what this project + host can build.
	var suggested []string
	for _, t := range targets.All() {
		ok, _ := h.CanBuild(t.Platform())
		if !ok || !proj.Platforms[t.Platform()] {
			continue
		}
		if t.Optional() {
			fatal := false
			for _, is := range t.Preflight(ctxT) {
				fatal = fatal || is.Fatal
			}
			if fatal {
				continue
			}
		}
		suggested = append(suggested, t.Name())
	}
	var tlist []string
	for {
		ans := pr.ask(i18n.S("Default targets for `fpack build` (comma separated)", "`fpack build` 的默认目标（逗号分隔）"), strings.Join(suggested, ","))
		tlist = nil
		bad := ""
		for _, n := range strings.FieldsFunc(ans, func(r rune) bool { return r == ',' || r == ' ' }) {
			t, ok := targets.Get(n)
			if !ok {
				bad = n
				break
			}
			tlist = append(tlist, t.Name())
		}
		if bad == "" {
			break
		}
		u.Warn(i18n.F("unknown target %q (available: %s)", "未知目标 %q（可用：%s）", bad, strings.Join(targets.Names(), ", ")))
		if !interactive {
			return build.ExitUsage
		}
	}
	display := proj.Name
	if proj.MacProductName != "" && !strings.Contains(proj.MacProductName, "$") {
		display = proj.MacProductName
	}
	display = pr.ask(i18n.S("App display name (installers, menus)", "应用显示名称（安装程序、菜单中显示）"), display)

	split := "false"
	keystore, alias := "", "upload"
	flavor := ""
	if fl := proj.AndroidFlavors; proj.Platforms[host.Android] && len(fl) > 0 {
		// Without a flavor flutter cannot build a project with productFlavors.
		for {
			flavor = pr.ask(i18n.F("Default flavor (%s)", "默认 flavor（%s）", strings.Join(fl, " / ")), fl[0])
			if contains(fl, flavor) || !interactive {
				break
			}
			u.Warn(i18n.F("unknown flavor %q", "未知 flavor %q", flavor))
		}
	}
	if proj.Platforms[host.Android] {
		split = pr.ask(i18n.S("Android APK split per ABI? false = one universal APK, true = per ABI, both", "Android APK 是否按 ABI 拆分？false = 单个通用包，true = 按 ABI，both = 两者都要"), "false")
		if _, err := config.ParseABIMode(split); err != nil {
			split = "false"
		}
		keystore = pr.ask(i18n.S("Android release keystore path (empty = configure later)", "Android release keystore 路径（留空 = 稍后配置）"), "")
		if keystore != "" {
			alias = pr.ask(i18n.S("Key alias", "Key 别名"), "upload")
		}
	}
	exportMethod := ""
	if proj.Platforms[host.IOS] {
		exportMethod = pr.ask(i18n.S("iOS export method (app-store / ad-hoc / development / enterprise; empty = Flutter default app-store)", "iOS 导出方式（app-store / ad-hoc / development / enterprise；留空 = Flutter 默认 app-store）"), "")
		if exportMethod != "" && !contains(config.ExportMethods, exportMethod) {
			u.Warn(i18n.F("unknown export method %q – leaving it commented out", "未知导出方式 %q —— 将保持注释状态", exportMethod))
			exportMethod = ""
		}
	}
	// Developer ID identities in the keychain, listed as a hint in the
	// generated macos.sign section.
	var devIDs, installerIDs []string
	keychainSeen := false
	if proj.Platforms[host.MacOS] && h.OS == "darwin" {
		if ids, ok := targets.CodesignIdentities(ctxT.Tools); ok {
			keychainSeen = true
			for _, id := range ids {
				if strings.HasPrefix(id, "Developer ID Application") {
					devIDs = append(devIDs, id)
				}
			}
		}
		if ids, ok := targets.InstallerIdentities(ctxT.Tools); ok {
			for _, id := range ids {
				if strings.HasPrefix(id, "Developer ID Installer") {
					installerIDs = append(installerIDs, id)
				}
			}
		}
	}
	outDir := pr.ask(i18n.S("Output directory", "输出目录"), config.DefaultOutputDir)

	content := renderInitYAML(initValues{
		Targets: tlist, Display: display, Split: split, Keystore: keystore, Alias: alias,
		ExportMethod: exportMethod, Flavor: flavor, OutDir: outDir, DevIDs: devIDs, KeychainSeen: keychainSeen, InstallerIDs: installerIDs, Proj: proj,
	})
	var check config.Config
	if err := config.Parse([]byte(content), &check); err != nil {
		u.Errorf(i18n.S("internal error: generated config is invalid: %v", "内部错误：生成的配置无效：%v"), err)
		return build.ExitFailed
	}
	if exists && !p.b("force") {
		newPath := dest + ".new"
		if err := os.WriteFile(newPath, []byte(content), 0o644); err != nil {
			u.Errorf("%v", err)
			return build.ExitFailed
		}
		u.Blank()
		u.Success(i18n.F("wrote %s", "已生成 %s", ctxT.Rel(newPath)))
		u.Println(i18n.S("Differences (- your fpack.yaml, + new template):", "差异（- 现有 fpack.yaml，+ 新模板）："))
		for _, l := range lineDiff(string(existing), content, 80) {
			switch {
			case strings.HasPrefix(l, "+"):
				u.Println("  " + u.Green(l))
			case strings.HasPrefix(l, "-"):
				u.Println("  " + u.Red(l))
			default:
				u.Println("  " + u.Dim(l))
			}
		}
		u.Hint(i18n.S("merge what you need into fpack.yaml, or replace it: fpack init --force", "把需要的部分合并进 fpack.yaml，或直接替换：fpack init --force"))
		return 0
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		u.Errorf("%v", err)
		return build.ExitFailed
	}
	u.Blank()
	u.Success(i18n.F("wrote %s", "已生成 %s", ctxT.Rel(dest)))
	if gi, err := os.ReadFile(filepath.Join(proj.Root, ".gitignore")); err == nil && !strings.Contains(string(gi), "dist") {
		u.Hint(i18n.S("tip: add `dist/` to .gitignore (fpack never edits your files)", "提示：建议把 `dist/` 加入 .gitignore（fpack 不会修改你的文件）"))
	}
	u.Blank()
	u.Println(i18n.S("Next:", "下一步："))
	u.Info("fpack doctor        " + u.Dim(i18n.S("# check prerequisites", "# 检查环境")))
	if len(tlist) > 0 {
		u.Info("fpack build         " + u.Dim("# "+strings.Join(tlist, ", ")))
	}
	u.Info("fpack build --all   " + u.Dim(i18n.S("# everything this machine can build", "# 本机能构建的全部目标")))
	return 0
}

func contains(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// lineDiff returns a compact line diff (LCS based): "-" removed, "+" added,
// "…" for skipped unchanged runs. At most max lines are returned.
func lineDiff(a, b string, max int) []string {
	x, y := strings.Split(strings.TrimRight(a, "\n"), "\n"), strings.Split(strings.TrimRight(b, "\n"), "\n")
	if len(x)*len(y) > 4_000_000 {
		return []string{fmt.Sprintf("(%d → %d lines)", len(x), len(y))}
	}
	n, m := len(x), len(y)
	lcs := make([][]int32, n+1)
	for i := range lcs {
		lcs[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var out []string
	same := 0
	flush := func() {
		if same > 0 {
			out = append(out, fmt.Sprintf("… %d %s", same, i18n.S("unchanged line(s)", "行未变")))
			same = 0
		}
	}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && x[i] == y[j]:
			same++
			i, j = i+1, j+1
			continue
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			flush()
			out = append(out, "+ "+y[j])
			j++
		default:
			flush()
			out = append(out, "- "+x[i])
			i++
		}
		if len(out) >= max {
			return append(out, i18n.S("… (diff truncated)", "…（差异过长已截断）"))
		}
	}
	flush()
	return out
}

// ------------------------------------------------------------------ schema --

func schemaCommand() *command {
	c := &command{name: "schema", en: "print the JSON schema of fpack.yaml (editor autocompletion and validation)", zh: "输出 fpack.yaml 的 JSON Schema（用于编辑器补全与校验）",
		flags: []flagSpec{{names: []string{"--output", "-o"}, kind: kString, metavar: "FILE", en: "write to FILE instead of stdout", zh: "写入 FILE 而不是标准输出"}}}
	c.help = func() string {
		return cmdHelp(c, i18n.S("fpack schema [-o FILE]", "fpack schema [-o 文件]"), "  fpack schema -o .vscode/fpack.schema.json\n"+
			"  # "+i18n.S("fpack init adds this line so VS Code / IntelliJ (YAML plugin) use it automatically:", "fpack init 会在 fpack.yaml 顶部加入下面这行，VS Code / IntelliJ（YAML 插件）会自动使用：")+"\n"+
			"  # yaml-language-server: $schema="+config.SchemaURL+"\n")
	}
	c.run = func(e *Env, p *parsed) int {
		b, err := config.Schema()
		if err != nil {
			fmt.Fprintln(e.Stderr, err)
			return build.ExitFailed
		}
		b = append(b, '\n')
		if out := p.s("output"); out != "" {
			if err := os.WriteFile(out, b, 0o644); err != nil {
				fmt.Fprintln(e.Stderr, err)
				return build.ExitFailed
			}
			fmt.Fprintln(e.Stderr, i18n.F("wrote %s", "已生成 %s", out))
			return 0
		}
		e.Stdout.Write(b)
		return 0
	}
	return c
}
