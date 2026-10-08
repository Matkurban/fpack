package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Matkurban/fpack/go/internal/build"
	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/runner"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
)

var buildFlags = []flagSpec{
	{names: []string{"--all", "-a"}, kind: kBool, en: "build every target this machine can build (others are skipped with a reason)", zh: "构建本机能构建的所有目标（其余会被跳过并说明原因）"},
	{names: []string{"--dry-run", "-n"}, kind: kBool, en: "print the plan and exact commands, run nothing", zh: "只打印计划和具体命令，不执行"},
	{names: []string{"--mode", "-m"}, kind: kString, metavar: "MODE", en: "release (default) | profile | debug", zh: "release（默认）| profile | debug"},
	{names: []string{"--release"}, kind: kBool, en: "same as --mode release", zh: "等同 --mode release"},
	{names: []string{"--profile"}, kind: kBool, en: "same as --mode profile", zh: "等同 --mode profile"},
	{names: []string{"--debug"}, kind: kBool, en: "same as --mode debug", zh: "等同 --mode debug"},
	{names: []string{"--flavor"}, kind: kString, metavar: "NAME", en: "build flavor (Android productFlavor / Xcode scheme)", zh: "构建 flavor（Android productFlavor / Xcode scheme）"},
	{names: []string{"--target", "-t"}, kind: kString, metavar: "FILE", en: "entry point, e.g. lib/main_prod.dart", zh: "入口文件，例如 lib/main_prod.dart"},
	{names: []string{"--dart-define"}, kind: kList, metavar: "K=V", en: "compile-time variable (repeatable)", zh: "编译期变量（可重复）"},
	{names: []string{"--dart-define-from-file"}, kind: kList, metavar: "FILE", en: "JSON/.env file with defines (repeatable)", zh: "包含 define 的 JSON/.env 文件（可重复）"},
	{names: []string{"--build-name"}, kind: kString, metavar: "X.Y.Z", en: "override version name from pubspec.yaml", zh: "覆盖 pubspec.yaml 中的版本名"},
	{names: []string{"--build-number"}, kind: kString, metavar: "N", en: "override build number from pubspec.yaml", zh: "覆盖 pubspec.yaml 中的构建号"},
	{names: []string{"--split-per-abi"}, kind: kOptional, metavar: "true|both", en: "APK: one file per ABI; =both also keeps the universal APK", zh: "APK：按 ABI 拆分；=both 同时保留通用包"},
	{names: []string{"--abis"}, kind: kString, metavar: "LIST", en: "Android ABIs, e.g. arm64-v8a,armeabi-v7a", zh: "Android ABI 列表，例如 arm64-v8a,armeabi-v7a"},
	{names: []string{"--obfuscate"}, kind: kBool, en: "obfuscate Dart code (symbols go to <output>/debug-info)", zh: "混淆 Dart 代码（符号文件保存到 <输出目录>/debug-info）"},
	{names: []string{"--split-debug-info"}, kind: kString, metavar: "DIR", en: "where to store debug symbols", zh: "调试符号保存目录"},
	{names: []string{"--output", "-o"}, kind: kString, metavar: "DIR", en: "output directory (default dist/{version}{+build})", zh: "输出目录（默认 dist/{version}{+build}）"},
	{names: []string{"--force", "-f"}, kind: kBool, en: "overwrite existing artifacts", zh: "覆盖已存在的产物"},
	{names: []string{"--export-method"}, kind: kString, metavar: "M", en: "iOS: app-store | ad-hoc | development | enterprise …", zh: "iOS：app-store | ad-hoc | development | enterprise 等"},
	{names: []string{"--export-options-plist"}, kind: kString, metavar: "FILE", en: "iOS: ExportOptions.plist", zh: "iOS：ExportOptions.plist"},
	{names: []string{"--no-codesign"}, kind: kBool, en: "iOS: build an unsigned IPA", zh: "iOS：构建未签名 IPA"},
	{names: []string{"--sign"}, kind: kBool, en: "macOS: Developer ID sign the app/DMG", zh: "macOS：使用 Developer ID 签名 App/DMG"},
	{names: []string{"--no-sign"}, kind: kBool, en: "macOS: do not sign (also disables notarization)", zh: "macOS：不签名（同时关闭公证）"},
	{names: []string{"--sign-identity"}, kind: kString, metavar: "ID", en: "macOS: codesign identity", zh: "macOS：codesign 证书名"},
	{names: []string{"--notarize"}, kind: kBool, en: "macOS: notarize + staple the DMG", zh: "macOS：公证并装订 DMG"},
	{names: []string{"--no-notarize"}, kind: kBool, en: "macOS: skip notarization (faster local builds)", zh: "macOS：跳过公证（本地构建更快）"},
	{names: []string{"--notary-profile"}, kind: kString, metavar: "NAME", en: "macOS: notarytool keychain profile", zh: "macOS：notarytool 钥匙串配置名"},
	{names: []string{"--dmg-tool"}, kind: kString, metavar: "T", en: "macOS: auto | hdiutil | create-dmg", zh: "macOS：auto | hdiutil | create-dmg"},
	{names: []string{"--base-href"}, kind: kString, metavar: "PATH", en: "web: base href, e.g. /app/", zh: "web：base href，例如 /app/"},
	{names: []string{"--wasm"}, kind: kBool, en: "web: build with WebAssembly", zh: "web：使用 WebAssembly 构建"},
}

func buildCommand() *command {
	c := &command{name: "build", aliases: []string{"b"}, flags: buildFlags,
		en: "build and package targets into dist/", zh: "构建并打包目标到 dist/"}
	c.help = func() string {
		var rows [][2]string
		for _, t := range targets.All() {
			rows = append(rows, [2]string{t.Name(), t.Description()})
		}
		return cmdHelp(c, "fpack build [targets...] [options] [-- extra flutter args]",
			`  fpack build apk
  fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
  fpack build apk --split-per-abi=both
  fpack build ipa --export-method ad-hoc
  fpack build ipa --no-codesign
  fpack build macos dmg                 # `+i18n.S("uses pubspec.yaml dmg: signing settings if present", "如存在，使用 pubspec.yaml 中 dmg: 的签名配置")+`
  fpack build dmg --no-notarize
  fpack build --all --json > result.json
  fpack build web --base-href /app/ -- --no-web-resources-cdn
`) + "\n" + i18n.S("Targets:", "目标：") + "\n" + fmtRows(rows)
	}
	c.run = runBuild
	return c
}

// applyBuildFlags maps command-line flags onto the config (highest priority).
func applyBuildFlags(p *parsed) (func(*config.Config) error, error) {
	modes := 0
	mode := p.s("mode")
	for _, m := range []string{"release", "profile", "debug"} {
		if p.b(m) {
			modes++
			mode = m
		}
	}
	if modes > 1 || (modes == 1 && p.has("mode") && p.s("mode") != mode) {
		return nil, usagef("choose only one of --release, --profile, --debug, --mode", "--release、--profile、--debug、--mode 只能选一个")
	}
	if p.b("sign") && p.b("no-sign") || p.b("notarize") && p.b("no-notarize") {
		return nil, usagef("conflicting --sign/--no-sign or --notarize/--no-notarize", "--sign/--no-sign 或 --notarize/--no-notarize 相互冲突")
	}
	var abiMode config.ABIMode
	if p.has("split-per-abi") {
		m, err := config.ParseABIMode(p.s("split-per-abi"))
		if err != nil {
			return nil, &UsageError{Msg: "--" + err.Error()}
		}
		abiMode = m
	}
	t := true
	f := false
	return func(c *config.Config) error {
		if mode != "" {
			c.Build.Mode = mode
		}
		set := func(k string, dst *string) {
			if p.has(k) {
				*dst = p.s(k)
			}
		}
		set("flavor", &c.Build.Flavor)
		set("target", &c.Build.Target)
		set("split-debug-info", &c.Build.SplitDebugInfo)
		set("output", &c.Output.Dir)
		set("export-method", &c.IOS.ExportMethod)
		set("export-options-plist", &c.IOS.ExportOptionsPlist)
		set("sign-identity", &c.MacOS.Sign.Identity)
		set("notary-profile", &c.MacOS.Sign.NotaryProfile)
		set("dmg-tool", &c.MacOS.DMG.Tool)
		set("base-href", &c.Web.BaseHref)
		if p.has("build-name") {
			c.Build.BuildName = config.Scalar(p.s("build-name"))
		}
		if p.has("build-number") {
			c.Build.BuildNumber = config.Scalar(p.s("build-number"))
		}
		if len(p.l("dart-define")) > 0 {
			for _, d := range p.l("dart-define") {
				if !strings.Contains(d, "=") {
					return fmt.Errorf("--dart-define %q must be KEY=VALUE", d)
				}
			}
			c.Build.DartDefine = append(c.Build.DartDefine, p.l("dart-define")...)
		}
		if len(p.l("dart-define-from-file")) > 0 {
			c.Build.DartDefineFromFile = append(c.Build.DartDefineFromFile, p.l("dart-define-from-file")...)
		}
		if abiMode != "" {
			c.Android.SplitPerABI = abiMode
		}
		if p.has("abis") {
			c.Android.ABIs = config.List(strings.Split(strings.ReplaceAll(p.s("abis"), " ", ""), ","))
		}
		if p.b("obfuscate") {
			c.Build.Obfuscate = &t
		}
		if p.b("force") {
			c.Output.Overwrite = &t
		}
		if p.b("no-codesign") {
			c.IOS.Codesign = &f
		}
		if p.b("sign") {
			c.MacOS.Sign.Enabled = &t
		}
		if p.b("no-sign") {
			c.MacOS.Sign.Enabled = &f
			c.MacOS.Sign.Notarize = &f
		}
		if p.b("notarize") {
			c.MacOS.Sign.Notarize = &t
		}
		if p.b("no-notarize") {
			c.MacOS.Sign.Notarize = &f
		}
		if p.b("wasm") {
			c.Web.Wasm = &t
		}
		return nil
	}, nil
}

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

// signalContext cancels on the first Ctrl-C (children are stopped
// gracefully) and force-kills on the second.
func signalContext(u *ui.UI) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		n := 0
		for range ch {
			n++
			if n == 1 {
				u.Blank()
				u.Warn(i18n.S("stopping… (press Ctrl-C again to force)", "正在停止…（再按一次 Ctrl-C 强制退出）"))
				cancel()
				continue
			}
			runner.KillAll()
			os.Exit(build.ExitInterrupted)
		}
	}()
	return ctx, func() { signal.Stop(ch); cancel() }
}

func runBuild(e *Env, p *parsed) int {
	u := newUI(e, p)
	for _, n := range p.pos {
		if _, ok := targets.Get(n); !ok {
			msg := i18n.F("unknown target %q", "未知目标 %q", n)
			if s := config.Suggest(n, targets.Names()); s != "" {
				msg += i18n.F(" (did you mean %q?)", "（你是不是想用 %q？）", s)
			}
			u.Errorf("%s", msg)
			u.Info(i18n.S("available: ", "可用目标：") + strings.Join(targets.Names(), ", "))
			return build.ExitUsage
		}
	}
	if p.b("all") && len(p.pos) > 0 {
		u.Errorf("%s", i18n.S("use either --all or a list of targets, not both", "--all 与目标列表不能同时使用"))
		return build.ExitUsage
	}
	override, err := applyBuildFlags(p)
	if err != nil {
		u.Errorf("%v", err)
		return build.ExitUsage
	}
	ctxT, err := build.NewContext(build.Options{
		ProjectDir: p.s("project"), ConfigPath: p.s("config"), FlutterPath: p.s("flutter"),
		Override: override, PassArgs: p.pass, DryRun: p.b("dry-run"), RequireSDK: !p.b("dry-run"),
		Interactive: isTTY(os.Stdout) && isTTY(os.Stdin), Getenv: e.Getenv,
	})
	if err != nil {
		return contextError(u, err)
	}
	names := p.pos
	if !p.b("all") && len(names) == 0 {
		names = ctxT.Config.Build.Targets
		for _, n := range names {
			if _, ok := targets.Get(n); !ok {
				u.Errorf("%s", i18n.F("fpack.yaml build.targets: unknown target %q", "fpack.yaml build.targets：未知目标 %q", n))
				return build.ExitUsage
			}
		}
	}
	if !p.b("all") && len(names) == 0 {
		u.Errorf("%s", i18n.S("which targets? e.g. `fpack build apk`, `fpack build apk ipa`, or `fpack build --all`", "要构建哪些目标？例如 `fpack build apk`、`fpack build apk ipa` 或 `fpack build --all`"))
		u.Info(i18n.S("available: ", "可用目标：") + strings.Join(targets.Names(), ", "))
		u.Hint(i18n.S("set default targets with build.targets in fpack.yaml (fpack init)", "可在 fpack.yaml 的 build.targets 中设置默认目标（fpack init）"))
		return build.ExitUsage
	}
	ctx, stop := signalContext(u)
	defer stop()
	s := build.Run(ctx, ctxT, u, build.Request{Names: names, All: p.b("all"), DryRun: p.b("dry-run"), Verbose: p.b("verbose")})
	if p.b("json") {
		printJSON(e, s)
	}
	return s.ExitCode
}
