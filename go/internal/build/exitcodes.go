package build

// ExitCodeDoc documents one exit code. The tables in README.md,
// README.ZH.md, doc/configuration.md and the website's troubleshooting page
// are generated from ExitCodeDocs (FPACK_UPDATE=1 go test ./internal/cli).
type ExitCodeDoc struct {
	Code   int
	EN, ZH string
}

// ExitCodeDocs lists every exit code fpack uses.
var ExitCodeDocs = []ExitCodeDoc{
	{0, "success (`--dry-run`: the plan was produced)", "成功（`--dry-run`：计划成功生成）"},
	{ExitFailed,
		"at least one target failed to build/package, or a hook failed; `fpack doctor`: problems were found; `fpack notarize`: a submission was rejected or a query failed",
		"至少一个目标在构建/打包时失败，或钩子失败；`fpack doctor`：发现问题；`fpack notarize`：有提交被拒绝或查询失败"},
	{ExitUsage,
		"usage or configuration error (unknown flag or target, no targets and no `build.targets`, invalid fpack.yaml, no Flutter project found)",
		"用法或配置错误（未知参数或目标、未指定目标且没有 `build.targets`、fpack.yaml 有误、找不到 Flutter 项目）"},
	{ExitPrereq,
		"prerequisites missing / artifact already exists / nothing buildable / Flutter SDK not found; `fpack notarize`: no notarization record found",
		"前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter SDK；`fpack notarize`：找不到公证记录"},
	{ExitInterrupted,
		"interrupted by Ctrl-C (an interrupted notarization wait keeps processing on Apple's side)",
		"被 Ctrl-C 中断（公证等待被中断时，Apple 端仍会继续处理）"},
}
