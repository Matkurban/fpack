// Package cli implements the fpack command line.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/Matkurban/fpack/go/internal/build"
	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
	"github.com/Matkurban/fpack/go/internal/version"
)

// Env abstracts process environment for tests.
type Env struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	Getenv func(string) string
}

type command struct {
	name    string
	aliases []string
	en, zh  string
	flags   []flagSpec
	run     func(e *Env, p *parsed) int
	help    func() string
}

func commands() []*command {
	return []*command{buildCommand(), doctorCommand(), listCommand(), initCommand(), schemaCommand(), cleanCommand(), versionCommand()}
}

func findCommand(name string) *command {
	for _, c := range commands() {
		if c.name == name {
			return c
		}
		for _, a := range c.aliases {
			if a == name {
				return c
			}
		}
	}
	return nil
}

// Main runs fpack and returns the exit code.
func Main(args []string, e *Env) int {
	if e == nil {
		e = &Env{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Getenv: os.Getenv}
	}
	// Language and color must be known before any message is printed.
	langFlag := ""
	for i, a := range args {
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "--lang=") {
			langFlag = strings.TrimPrefix(a, "--lang=")
		} else if a == "--lang" && i+1 < len(args) {
			langFlag = args[i+1]
		}
	}
	if langFlag != "" {
		if _, ok := i18n.Parse(langFlag); !ok {
			fmt.Fprintf(e.Stderr, "error: --lang must be zh or en (got %q)\n", langFlag)
			return build.ExitUsage
		}
	}
	i18n.Set(i18n.Detect(langFlag, e.Getenv))

	if w := e.Getenv("FPACK_WRAPPER_VERSION"); w != "" && w != version.Version {
		fmt.Fprintln(e.Stderr, i18n.F("fpack: warning: Dart wrapper %s is running core %s. Reinstall with: dart pub global activate fpack", "fpack：警告：Dart 包装器版本 %s 与核心版本 %s 不一致。请重新安装：dart pub global activate fpack", w, version.Version))
	}

	if len(args) == 0 {
		fmt.Fprint(e.Stdout, mainHelp())
		return 0
	}
	switch args[0] {
	case "--core-version":
		fmt.Fprintln(e.Stdout, version.Version)
		return 0
	case "--version", "-V":
		return versionCommand().run(e, &parsed{})
	case "--help", "-h", "help":
		if len(args) > 1 {
			if c := findCommand(args[1]); c != nil {
				fmt.Fprint(e.Stdout, c.help())
				return 0
			}
		}
		fmt.Fprint(e.Stdout, mainHelp())
		return 0
	}
	name := args[0]
	if strings.HasPrefix(name, "-") {
		// Global flags before the command: fpack --lang zh build apk
		for i := 0; i < len(args); i++ {
			if takesValue(args[i]) {
				i++ // skip the value: fpack --lang zh build
				continue
			}
			if !strings.HasPrefix(args[i], "-") {
				if c := findCommand(args[i]); c != nil {
					rest := append(append([]string{}, args[:i]...), args[i+1:]...)
					return dispatch(c, rest, e)
				}
				break
			}
		}
		if hasFlag(args, "--help", "-h") {
			fmt.Fprint(e.Stdout, mainHelp())
			return 0
		}
		fmt.Fprintln(e.Stderr, i18n.F("error: missing command. Run `fpack --help`.", "错误：缺少命令。运行 `fpack --help` 查看用法。"))
		return build.ExitUsage
	}
	c := findCommand(name)
	if c == nil {
		msg := i18n.F("unknown command %q", "未知命令 %q", name)
		if _, ok := targets.Get(name); ok {
			msg += i18n.F(" – did you mean `fpack build %s`?", " —— 你是不是想运行 `fpack build %s`？", name)
		} else {
			var names []string
			for _, c := range commands() {
				names = append(names, c.name)
			}
			if s := config.Suggest(name, names); s != "" {
				msg += i18n.F(" (did you mean %q?)", "（你是不是想用 %q？）", s)
			}
		}
		fmt.Fprintln(e.Stderr, "error: "+msg+"\n"+i18n.S("Run `fpack --help` for usage.", "运行 `fpack --help` 查看用法。"))
		return build.ExitUsage
	}
	return dispatch(c, args[1:], e)
}

// takesValue reports whether a (global) flag consumes the next argument.
func takesValue(a string) bool {
	if strings.Contains(a, "=") {
		return false
	}
	for _, f := range globalFlags {
		for _, n := range f.names {
			if n == a {
				return f.kind == kString || f.kind == kList
			}
		}
	}
	return false
}

func hasFlag(args []string, names ...string) bool {
	for _, a := range args {
		for _, n := range names {
			if a == n {
				return true
			}
		}
	}
	return false
}

func dispatch(c *command, args []string, e *Env) int {
	p, err := parse(args, c.flags)
	if err != nil {
		fmt.Fprintln(e.Stderr, "error: "+err.Error())
		fmt.Fprintln(e.Stderr, i18n.F("Run `fpack %s --help` for usage.", "运行 `fpack %s --help` 查看用法。", c.name))
		return build.ExitUsage
	}
	if p.b("help") {
		fmt.Fprint(e.Stdout, c.help())
		return 0
	}
	return c.run(e, p)
}

// newUI creates the human output UI. With --json, human output goes to stderr.
func newUI(e *Env, p *parsed) *ui.UI {
	w := e.Stdout
	if p.b("json") {
		w = e.Stderr
	}
	return ui.New(ui.Options{Writer: w, NoColor: p.b("no-color"), Verbose: p.b("verbose")})
}

func printJSON(e *Env, v any) {
	enc := json.NewEncoder(e.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// contextError prints a friendly message for context creation errors and
// returns the exit code.
func contextError(u *ui.UI, err error) int {
	var nf *project.ErrNotFound
	var ce *build.ConfigError
	var ns *flutter.NotSDKError
	switch {
	case errors.As(err, &nf):
		u.Errorf("%s", i18n.F("no Flutter project found in %s or its parents", "在 %s 及其上级目录中找不到 Flutter 项目", nf.Start))
		if nf.NonFlutter != "" {
			u.Info(i18n.F("(%s is not a Flutter app – it has no `flutter: sdk: flutter` dependency)", "（%s 不是 Flutter 应用 —— 没有 `flutter: sdk: flutter` 依赖）", nf.NonFlutter))
		}
		if len(nf.Candidates) > 0 {
			u.Info(i18n.S("Flutter apps found below this directory:", "在当前目录下找到以下 Flutter 应用："))
			for _, cnd := range nf.Candidates {
				u.Info("  " + cnd)
			}
			u.Hint(i18n.F("cd into one of them, or run: fpack -C %s …", "进入其中一个目录，或运行：fpack -C %s …", nf.Candidates[0]))
		} else {
			u.Hint(i18n.S("run fpack inside a Flutter app (the folder with pubspec.yaml), or pass -C <dir>", "请在 Flutter 应用目录（含 pubspec.yaml）中运行，或使用 -C <目录>"))
		}
		return build.ExitUsage
	case errors.As(err, &ce):
		u.Errorf("%s", i18n.S("invalid configuration:", "配置无效："))
		for _, l := range strings.Split(ce.Error(), "\n") {
			u.Info(l)
		}
		u.Hint(i18n.S("see README → Configuration reference", "参见 README → 配置参考"))
		return build.ExitUsage
	case errors.As(err, &ns):
		u.Errorf("%s", ns.Error())
		return build.ExitPrereq
	case errors.Is(err, flutter.ErrNotFound):
		u.Errorf("%s", i18n.S("Flutter SDK not found", "找不到 Flutter SDK"))
		u.Hint(i18n.S("install Flutter and add <sdk>/bin to PATH, or pass --flutter <sdk>, or set FPACK_FLUTTER, or use FVM (.fvm/flutter_sdk)",
			"请安装 Flutter 并把 <SDK>/bin 加入 PATH；或使用 --flutter <SDK路径>、设置 FPACK_FLUTTER，或使用 FVM（.fvm/flutter_sdk）"))
		return build.ExitPrereq
	}
	u.Errorf("%v", err)
	return build.ExitUsage
}

func mainHelp() string {
	var b strings.Builder
	b.WriteString(i18n.S("fpack – package Flutter apps into release files for every platform, with one command.\n",
		"fpack —— 一条命令把 Flutter 应用打包成各平台的发布文件。\n"))
	b.WriteString("\n" + i18n.S("Usage:", "用法：") + "\n  fpack <command> [options]\n\n")
	b.WriteString(i18n.S("Commands:", "命令：") + "\n")
	var rows [][2]string
	for _, c := range commands() {
		n := c.name
		if len(c.aliases) > 0 {
			n += " (" + strings.Join(c.aliases, ", ") + ")"
		}
		rows = append(rows, [2]string{n, i18n.S(c.en, c.zh)})
	}
	rows = append(rows, [2]string{"help <command>", i18n.S("show help for a command", "显示某个命令的帮助")})
	b.WriteString(fmtRows(rows))
	b.WriteString("\n" + i18n.S("Global options:", "全局选项：") + "\n")
	b.WriteString(fmtRows(append(flagHelp(globalFlags), [2]string{"--version, -V", i18n.S("print version", "显示版本")})))
	b.WriteString("\n" + i18n.S("Examples:", "示例：") + "\n")
	b.WriteString(`  fpack init                    # ` + i18n.S("create fpack.yaml (optional – zero config works)", "生成 fpack.yaml（可选，零配置也能用）") + `
  fpack doctor                  # ` + i18n.S("check what this machine can build", "检查本机能构建哪些目标") + `
  fpack build apk               # ` + i18n.S("release APK into dist/<version>/", "构建 release APK 到 dist/<版本>/") + `
  fpack build apk aab ipa       # ` + i18n.S("several targets in one go", "一次构建多个目标") + `
  fpack build --all             # ` + i18n.S("everything this machine can build", "构建本机能构建的所有目标") + `
  fpack build dmg --flavor prod # ` + i18n.S("signed + notarized DMG of a flavor", "构建某个 flavor 的签名 + 公证 DMG") + `
  fpack build apk --dry-run     # ` + i18n.S("show the commands without running them", "只显示将执行的命令，不实际执行") + `
`)
	b.WriteString("\n" + i18n.S("Targets: ", "目标：") + strings.Join(targets.Names(), ", ") + "\n")
	b.WriteString(i18n.S("Docs: https://github.com/Matkurban/fpack#readme\n", "文档：https://github.com/Matkurban/fpack#readme\n"))
	return b.String()
}

func cmdHelp(c *command, usage string, examples string) string {
	var b strings.Builder
	b.WriteString(i18n.S(c.en, c.zh) + "\n\n" + i18n.S("Usage:", "用法：") + "\n  " + usage + "\n")
	if len(c.flags) > 0 {
		b.WriteString("\n" + i18n.S("Options:", "选项：") + "\n" + fmtRows(flagHelp(c.flags)))
	}
	b.WriteString("\n" + i18n.S("Global options:", "全局选项：") + "\n" + fmtRows(flagHelp(globalFlags)))
	if examples != "" {
		b.WriteString("\n" + i18n.S("Examples:", "示例：") + "\n" + examples)
	}
	return b.String()
}

func versionCommand() *command {
	c := &command{name: "version", en: "print version information", zh: "显示版本信息"}
	c.run = func(e *Env, p *parsed) int {
		s := fmt.Sprintf("fpack %s (core %s/%s, %s)", version.Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		if w := e.Getenv("FPACK_WRAPPER_VERSION"); w != "" {
			s += fmt.Sprintf(" · dart wrapper %s", w)
		}
		if p.b("json") {
			printJSON(e, map[string]string{"version": version.Version, "os": runtime.GOOS, "arch": runtime.GOARCH, "go": runtime.Version(), "wrapper": e.Getenv("FPACK_WRAPPER_VERSION")})
			return 0
		}
		fmt.Fprintln(e.Stdout, s)
		return 0
	}
	c.help = func() string { return cmdHelp(c, "fpack version", "  fpack --version\n") }
	return c
}
