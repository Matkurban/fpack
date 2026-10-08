package build

import (
	"fmt"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
	"github.com/Matkurban/fpack/go/internal/version"
)

func printHeader(c *targets.Context, u *ui.UI, list []targets.Target, req Request) {
	fl := c.SDK.Version
	if fl == "" {
		fl = "?"
	}
	ver := c.BuildName
	if c.BuildNumber != "" {
		ver += "+" + c.BuildNumber
	}
	u.Println(u.Bold("fpack "+version.Version) + u.Dim("  ·  ") + u.Bold(c.Project.Name) + " " + ver +
		u.Dim("  ·  ") + "Flutter " + fl + u.Dim(" ("+c.SDK.Source+")") + u.Dim("  ·  ") + c.Host.OS + "/" + c.Host.Arch)
	var names []string
	for _, t := range list {
		names = append(names, t.Name())
	}
	what := strings.Join(names, ", ")
	if req.All {
		what = "--all"
	}
	line := fmt.Sprintf("%s %s   %s %s   %s %s", u.Dim(i18n.S("Targets:", "目标：")), what, u.Dim(i18n.S("Mode:", "模式：")), c.Mode(), u.Dim(i18n.S("Output:", "输出：")), c.Rel(c.OutDir))
	if f := c.Flavor(); f != "" {
		line += fmt.Sprintf("   %s %s", u.Dim("Flavor:"), f)
	}
	u.Println(line)
	if c.Config.File != "" {
		u.Println(u.Dim(i18n.S("Config: ", "配置：") + c.Rel(c.Config.File)))
	}
}

func printPlan(c *targets.Context, u *ui.UI, s *Summary, steps map[string]*stepState) {
	u.Blank()
	u.Title(i18n.S("Plan (dry run – nothing will be executed)", "构建计划（dry run —— 不会执行任何命令）"))
	shown := map[string]string{}
	for _, tr := range s.Targets {
		u.Blank()
		head := fmt.Sprintf("%s  %s", u.Bold(tr.Target), u.Dim(tr.target.Description()))
		switch {
		case tr.Status == Skipped && len(tr.steps) == 0:
			u.Println(u.Dim("–") + " " + head)
			u.Detail(i18n.S("skip: ", "跳过：") + tr.Reason)
			continue
		case tr.Status == Failed:
			u.Println(u.Red("✗") + " " + head)
			msg := tr.Reason
			if msg == "" {
				msg = tr.Error
			}
			u.Println("    " + u.Red(i18n.S("would fail: ", "将会失败：")+msg))
			if tr.Fix != "" {
				u.Hint(tr.Fix)
			}
		default:
			u.Println(u.Cyan("▶") + " " + head)
		}
		if tr.Reason != "" && tr.Status != Failed {
			u.Detail(tr.Reason)
		}
		for _, key := range tr.steps {
			st := steps[key]
			if owner, ok := shown[key]; ok {
				u.Info(u.Dim(i18n.F("(reuses the flutter build of %s)", "（复用 %s 的 flutter 构建）", owner)))
				continue
			}
			shown[key] = tr.Target
			u.Info("$ " + flutterCmd(c, st.step).String())
			for _, w := range st.step.Warnings {
				u.Warn(w)
			}
		}
		in, err := tr.target.Locate(c, true, time.Time{})
		if err == nil {
			if p, err := tr.target.Package(c, in); err == nil {
				for _, op := range p.Ops {
					if op.Cmd != nil {
						opt := ""
						if op.Optional {
							opt = u.Dim(i18n.S("  (optional)", "  （可选）"))
						}
						u.Info("$ " + op.Cmd.String() + opt)
					} else {
						u.Info(u.Dim("· " + op.Desc))
					}
				}
			}
		}
		for _, a := range tr.Artifacts {
			u.Info(u.Green("= ") + c.Rel(a.Path) + u.Dim("  ["+a.Kind+"]"))
		}
		for _, w := range tr.Warnings {
			u.Warn(firstLineOf(w))
		}
		for _, n := range tr.Notes {
			u.Detail(n)
		}
	}
	if len(s.Conflicts) > 0 {
		u.Blank()
		u.Warn(i18n.S("already exists (a real run stops unless --force):", "以下产物已存在（真正执行时会停止，除非加 --force）："))
		for _, p := range s.Conflicts {
			u.Detail(c.Rel(p))
		}
	}
}

func finish(c *targets.Context, u *ui.UI, s *Summary, start time.Time, checksums, logDir string) *Summary {
	s.DurationMs = time.Since(start).Milliseconds()
	s.Checksums = checksums
	var ok, failed, skipped, planned, reference int
	attemptedFail := false
	for _, t := range s.Targets {
		if t.duration > 0 {
			t.DurationMs = t.duration.Milliseconds()
		}
		switch t.Status {
		case Success:
			ok++
		case Failed:
			failed++
			if t.attempted {
				attemptedFail = true
			}
		case Skipped:
			skipped++
		case Planned:
			if t.ReferenceOnly {
				reference++
			} else {
				planned++
			}
		}
	}
	for _, t := range s.Targets {
		if t.Log != "" {
			s.LogDir = logDir
		}
	}
	if ok > 0 && s.LogDir == "" {
		s.LogDir = logDir
	}
	switch {
	case s.Interrupted:
		s.ExitCode = ExitInterrupted
	case attemptedFail:
		s.ExitCode = ExitFailed
	case failed > 0:
		s.ExitCode = ExitPrereq
	case !s.DryRun && ok == 0:
		s.ExitCode = ExitPrereq // nothing could be built
	}
	s.Success = s.ExitCode == 0

	if s.DryRun {
		u.Blank()
		msg := i18n.F("%d target(s) planned, %d skipped, %d would fail.", "计划 %d 个目标，跳过 %d 个，%d 个将会失败。", planned, skipped, failed)
		if reference > 0 {
			msg += " " + i18n.F("%d shown for reference only (needs another OS).", "另有 %d 个仅供参考（需要其他操作系统）。", reference)
		}
		if failed > 0 {
			u.Println(u.Yellow(msg))
		} else {
			u.Println(u.Green(msg))
		}
		return s
	}

	u.Blank()
	u.Title(i18n.S("Summary", "汇总"))
	var rows [][]string
	for _, t := range s.Targets {
		status := map[Status]string{Success: u.Green("✓ " + i18n.S("done", "完成")), Failed: u.Red("✗ " + i18n.S("failed", "失败")), Skipped: u.Dim("– " + i18n.S("skipped", "跳过")), Planned: "•"}[t.Status]
		dur := ""
		if t.DurationMs > 0 {
			dur = ui.Duration(time.Duration(t.DurationMs) * time.Millisecond)
		}
		if t.Status == Success && len(t.Artifacts) > 0 {
			for i, a := range t.Artifacts {
				name, st, d := t.Target, status, dur
				if i > 0 {
					name, st, d = "", "", ""
				}
				rows = append(rows, []string{name, st, c.Rel(a.Path), ui.Size(a.Size), d})
			}
			continue
		}
		reason := t.Reason
		if reason == "" {
			reason = t.Error
		}
		rows = append(rows, []string{t.Target, status, u.Dim(ui.Truncate(firstLineOf(reason), 64)), "", dur})
	}
	u.Table([]string{i18n.S("TARGET", "目标"), i18n.S("STATUS", "状态"), i18n.S("FILE", "文件"), i18n.S("SIZE", "大小"), i18n.S("TIME", "耗时")}, rows)

	// Details for anything that did not succeed. Targets failing for the
	// same reason (macos + dmg without a Developer ID, …) share one entry.
	type detail struct {
		names []string
		body  string
	}
	var details []*detail
	byBody := map[string]*detail{}
	for _, t := range s.Targets {
		if t.Status == Success || (t.Status == Skipped && t.Reason == i18n.S("interrupted", "已中断")) {
			continue
		}
		reason := t.Reason
		if reason == "" {
			reason = t.Error
		}
		body := reason
		if t.Fix != "" {
			body += "\n      " + u.Yellow("→ ") + strings.ReplaceAll(t.Fix, "\n", "\n        ")
		} else if t.Hint != "" && t.Status == Failed {
			body += "\n      " + u.Yellow("→ ") + t.Hint
		}
		if t.Log != "" {
			body += "\n      " + u.Dim(i18n.S("log: ", "日志：")+c.Rel(t.Log))
		}
		if d, ok := byBody[body]; ok {
			d.names = append(d.names, t.Target)
			continue
		}
		d := &detail{names: []string{t.Target}, body: body}
		byBody[body] = d
		details = append(details, d)
	}
	if len(details) > 0 {
		u.Blank()
		for _, d := range details {
			u.Println("  " + u.Bold(strings.Join(d.names, ", ")) + ": " + d.body)
		}
	}
	var notes []string
	for _, t := range s.Targets {
		if t.Status != Success {
			continue
		}
		for _, n := range t.Notes {
			notes = append(notes, t.Target+": "+n)
		}
	}
	if len(notes) > 0 {
		u.Blank()
		for _, n := range notes {
			u.Detail(n)
		}
	}
	u.Blank()
	if ok > 0 {
		u.Println(u.Dim(i18n.S("Output:    ", "输出目录：")) + c.Rel(c.OutDir))
		if checksums != "" {
			u.Println(u.Dim(i18n.S("Checksums: ", "校验和：  ")) + c.Rel(checksums))
		}
	}
	if s.LogDir != "" {
		u.Println(u.Dim(i18n.S("Logs:      ", "日志：    ")) + c.Rel(s.LogDir))
	}
	total := ui.Duration(time.Duration(s.DurationMs) * time.Millisecond)
	parts := []string{i18n.F("%d succeeded", "%d 个成功", ok)}
	if failed > 0 {
		parts = append(parts, i18n.F("%d failed", "%d 个失败", failed))
	}
	if skipped > 0 {
		parts = append(parts, i18n.F("%d skipped", "%d 个跳过", skipped))
	}
	msg := strings.Join(parts, ", ") + i18n.F(" in %s", "，总耗时 %s", total)
	switch {
	case s.Interrupted:
		u.Println(u.Yellow("■ " + i18n.S("Interrupted. ", "已中断。") + msg))
	case s.ExitCode == 0:
		u.Println(u.Green("✓ " + msg))
	default:
		u.Println(u.Red("✗ " + msg))
	}
	return s
}
