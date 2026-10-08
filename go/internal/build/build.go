package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Matkurban/fpack/go/internal/hints"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
	"github.com/Matkurban/fpack/go/internal/version"
)

// Request is what the user asked for.
type Request struct {
	Names   []string // explicit target names (already validated)
	All     bool
	DryRun  bool
	Verbose bool
}

// Status of a target.
type Status string

// Target statuses.
const (
	Success Status = "success"
	Failed  Status = "failed"
	Skipped Status = "skipped"
	Planned Status = "planned"
)

// Exit codes.
const (
	ExitOK          = 0
	ExitFailed      = 1
	ExitUsage       = 2
	ExitPrereq      = 3
	ExitInterrupted = 130
)

// ArtifactResult is a produced (or planned) file.
type ArtifactResult struct {
	Path   string `json:"path"`
	File   string `json:"file"`
	Kind   string `json:"kind"`
	Arch   string `json:"arch,omitempty"`
	Size   int64  `json:"size,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	SHA512 string `json:"sha512,omitempty"`
	// Notarization is set for macOS artifacts sent to Apple's notary service.
	Notarization *Notarization `json:"notarization,omitempty"`
}

// Notarization summarizes a notary submission (details in NOTARIZATION.md).
type Notarization struct {
	State  string `json:"state"` // submitted | accepted | stapled | invalid
	ID     string `json:"id"`
	Status string `json:"status"` // Apple's status (or Submitted)
	Record string `json:"record"` // path of NOTARIZATION.md
}

// TargetResult is the outcome of one target.
type TargetResult struct {
	Target   string `json:"target"`
	Platform string `json:"platform"`
	Status   Status `json:"status"`
	// ReferenceOnly marks a dry-run plan for a target this host cannot build.
	ReferenceOnly bool             `json:"referenceOnly,omitempty"`
	Reason        string           `json:"reason,omitempty"`
	Fix           string           `json:"fix,omitempty"`
	Error         string           `json:"error,omitempty"`
	Hint          string           `json:"hint,omitempty"`
	Excerpt       []string         `json:"errorExcerpt,omitempty"`
	Log           string           `json:"log,omitempty"`
	DurationMs    int64            `json:"durationMs"`
	Artifacts     []ArtifactResult `json:"artifacts"`
	Warnings      []string         `json:"warnings,omitempty"`
	Notes         []string         `json:"notes,omitempty"`
	Commands      []string         `json:"commands,omitempty"`

	target    targets.Target
	ifFails   []string // warnings shown only if the build fails (Issue.IfFails)
	steps     []string
	attempted bool
	start     time.Time
	duration  time.Duration
}

// Summary is the result of `fpack build` (also the --json document).
type Summary struct {
	FpackVersion string          `json:"fpackVersion"`
	Project      string          `json:"project"`
	ProjectRoot  string          `json:"projectRoot"`
	Version      string          `json:"version"`
	BuildNumber  string          `json:"buildNumber"`
	Flutter      string          `json:"flutter"`
	FlutterRoot  string          `json:"flutterRoot"`
	Host         string          `json:"host"`
	Mode         string          `json:"mode"`
	Flavor       string          `json:"flavor,omitempty"`
	OutputDir    string          `json:"outputDir"`
	DryRun       bool            `json:"dryRun"`
	Success      bool            `json:"success"`
	ExitCode     int             `json:"exitCode"`
	Interrupted  bool            `json:"interrupted,omitempty"`
	Checksums    string          `json:"checksumsFile,omitempty"`
	LogDir       string          `json:"logDir,omitempty"`
	DurationMs   int64           `json:"durationMs"`
	Targets      []*TargetResult `json:"targets"`
	Conflicts    []string        `json:"conflicts,omitempty"`
	// Hooks lists pre/post build hook commands (dry run).
	Hooks []string `json:"hooks,omitempty"`
	// HookError is set when a post_build hook failed.
	HookError string `json:"hookError,omitempty"`
}

type stepState struct {
	step    targets.FlutterStep
	users   []*TargetResult
	done    bool
	failed  bool
	start   time.Time
	logPath string
}

// Run executes a build request.
func Run(ctx context.Context, c *targets.Context, u *ui.UI, req Request) *Summary {
	start := time.Now()
	if c.Started.IsZero() {
		c.Started = start
	}
	s := &Summary{
		FpackVersion: version.Version, Project: c.Project.Name, ProjectRoot: c.Project.Root,
		Version: c.BuildName, BuildNumber: c.BuildNumber, Flutter: c.SDK.Version, FlutterRoot: c.SDK.Root,
		Host: c.Host.OS + "/" + c.Host.Arch, Mode: c.Mode(), Flavor: c.Flavor(), OutputDir: c.OutDir, DryRun: req.DryRun,
	}
	stamp := time.Now().Format("20060102-150405")
	logDir := filepath.Join(c.WorkDir, "logs")

	// 1. Resolve targets.
	var list []targets.Target
	explicit := map[string]bool{}
	if req.All {
		list = targets.All()
	} else {
		for _, n := range req.Names {
			t, _ := targets.Get(n)
			if !explicit[t.Name()] {
				explicit[t.Name()] = true
				list = append(list, t)
			}
		}
	}

	printHeader(c, u, list, req)

	// 2. Classify: host, project platform, preflight.
	steps := map[string]*stepState{}
	var order []string
	for _, t := range list {
		tr := &TargetResult{Target: t.Name(), Platform: string(t.Platform()), Status: Planned, target: t, Artifacts: []ArtifactResult{}}
		s.Targets = append(s.Targets, tr)
		isExplicit := explicit[t.Name()]
		hostOK, _ := c.Host.CanBuild(t.Platform())
		if ok, need := c.Host.CanBuild(t.Platform()); !ok {
			tr.Status = Skipped
			tr.Reason = i18n.F("needs %s (this is %s) – Flutter cannot cross-compile %s apps", "需要 %s（当前是 %s）—— Flutter 无法交叉编译 %s 应用", host.OSName(need), c.Host.OSName(), t.Platform())
			if isExplicit {
				if req.DryRun {
					tr.Notes = append(tr.Notes, i18n.S("planned for reference only – it would be skipped on this host", "仅供参考的计划 —— 在当前系统上会被跳过"))
					tr.Status = Planned
					tr.ReferenceOnly = true
				} else {
					tr.Status = Failed
					tr.Fix = i18n.F("run fpack on %s (e.g. a %s CI runner, see README → CI)", "请在 %s 上运行 fpack（例如 %s CI 机器，见 README → CI）", host.OSName(need), host.OSName(need))
					continue
				}
			} else {
				continue
			}
		}
		if !c.Project.Platforms[t.Platform()] {
			tr.Status = Skipped
			tr.Reason = i18n.F("project has no %s/ folder", "项目中没有 %s/ 目录", t.Platform())
			tr.Fix = "flutter create --platforms=" + string(t.Platform()) + " ."
			if isExplicit {
				tr.Status = Failed
			}
			continue
		}
		var issues []targets.Issue
		if hostOK {
			issues = append(projectIssues(c), targets.ConfigIssues(c, t.Name())...)
			issues = append(issues, t.Preflight(c)...)
		}
		var fatalIssue *targets.Issue
		for i := range issues {
			if issues[i].Fatal && fatalIssue == nil {
				fatalIssue = &issues[i]
			} else if !issues[i].Fatal {
				w := issues[i].Msg
				if issues[i].Fix != "" {
					w += "\n" + i18n.S("fix: ", "解决：") + issues[i].Fix
				}
				if issues[i].IfFails && !c.DryRun {
					tr.ifFails = append(tr.ifFails, w)
					continue
				}
				tr.Warnings = append(tr.Warnings, w)
			}
		}
		if fatalIssue != nil {
			tr.Reason = fatalIssue.Msg
			tr.Fix = fatalIssue.Fix
			if req.All && t.Optional() {
				tr.Status = Skipped
				continue
			}
			tr.Status = Failed
			if !req.DryRun {
				continue
			}
		}
		st, err := t.Steps(c)
		if err != nil {
			tr.Status, tr.Error = Failed, err.Error()
			continue
		}
		for _, fs := range st {
			tr.steps = append(tr.steps, fs.Key)
			if _, ok := steps[fs.Key]; !ok {
				steps[fs.Key] = &stepState{step: fs}
				order = append(order, fs.Key)
			}
			steps[fs.Key].users = append(steps[fs.Key].users, tr)
			tr.Warnings = appendUnique(tr.Warnings, fs.Warnings...)
		}
	}

	// 3. Predict artifacts: conflicts + dry-run plan.
	planned := map[string]string{}
	for _, tr := range s.Targets {
		if tr.Status != Planned && !(req.DryRun && len(tr.steps) > 0) {
			continue
		}
		in, err := tr.target.Locate(c, true, time.Time{})
		if err != nil {
			continue
		}
		p, err := targets.PackageFor(c, tr.target, in)
		if err != nil {
			tr.Status, tr.Error = Failed, err.Error()
			continue
		}
		for _, a := range p.Artifacts {
			tr.Artifacts = append(tr.Artifacts, ArtifactResult{Path: a.Path, File: filepath.Base(a.Path), Kind: a.Kind, Arch: a.Arch})
			if other, dup := planned[a.Path]; dup {
				tr.Status = Failed
				tr.Error = i18n.F("%s and %s would both write %s", "%s 与 %s 会写入同一个文件 %s", other, tr.Target, filepath.Base(a.Path))
				tr.Fix = i18n.S("make output.name / output.names unique, e.g. add {target} or {-arch}", "让 output.name / output.names 互不相同，例如加入 {target} 或 {-arch}")
			}
			planned[a.Path] = tr.Target
			if exists(a.Path) && !c.Config.Overwrite() {
				s.Conflicts = append(s.Conflicts, a.Path)
			}
		}
		tr.Notes = appendUnique(tr.Notes, p.Notes...)
		if req.DryRun {
			for _, key := range tr.steps {
				for _, op := range steps[key].step.Prepare {
					for _, cm := range op.Commands() {
						tr.Commands = append(tr.Commands, cm.String())
					}
				}
				tr.Commands = append(tr.Commands, flutterCmd(c, steps[key].step).String())
				for _, op := range steps[key].step.After {
					for _, cm := range op.Commands() {
						tr.Commands = append(tr.Commands, cm.String())
					}
				}
			}
			for _, op := range p.Ops {
				for _, cm := range op.Commands() {
					tr.Commands = append(tr.Commands, cm.String())
				}
			}
		}
	}

	if req.DryRun {
		for _, op := range targets.BuildHookOps(c, false, nil, true) {
			s.Hooks = append(s.Hooks, op.Cmd.String())
		}
		for _, op := range targets.BuildHookOps(c, true, nil, true) {
			s.Hooks = append(s.Hooks, op.Cmd.String())
		}
		printPlan(c, u, s, steps)
		return finish(c, u, s, start, "", logDir)
	}

	if len(s.Conflicts) > 0 {
		u.Blank()
		u.Errorf("%s", i18n.S("these artifacts already exist and would be overwritten:", "以下产物已存在，继续会被覆盖："))
		for _, p := range s.Conflicts {
			u.Info(c.Rel(p))
		}
		u.Hint(i18n.S("bump the version in pubspec.yaml (or --build-number N), or pass --force, or set output.overwrite: true",
			"请提升 pubspec.yaml 中的版本号（或使用 --build-number N），或加 --force，或设置 output.overwrite: true"))
		for _, tr := range s.Targets {
			if tr.Status == Planned {
				tr.Status, tr.Reason = Skipped, i18n.S("artifact already exists", "产物已存在")
			}
		}
		s.ExitCode = ExitPrereq
		s.DurationMs = time.Since(start).Milliseconds()
		return s
	}

	// 4. Execute.
	if err := c.Signing.Materialize(); err != nil {
		for _, tr := range s.Targets {
			if tr.Status == Planned && tr.Platform == string(host.Android) {
				tr.Status, tr.Error = Failed, err.Error()
			}
		}
	}
	defer c.Signing.Cleanup()

	// pre_build hooks run once, before the first flutter build.
	if hasPlanned(s) {
		if ops := targets.BuildHookOps(c, false, nil, true); len(ops) > 0 {
			u.Blank()
			u.Step("hooks.pre_build")
			if f := execOps(ctx, c, u, ops, filepath.Join(logDir, stamp+"-hooks.log"), req.Verbose); f != nil {
				for _, tr := range s.Targets {
					if tr.Status == Planned {
						tr.Status, tr.Error, tr.Log, tr.Excerpt = Failed, f.err, f.log, f.excerpt
					}
				}
				u.Fail(f.err)
				printFailure(c, u, f.excerpt, f.hint, f.hint != "", f.log)
			}
		}
	}

	interrupted := false
	for _, key := range order {
		st := steps[key]
		alive := false
		for _, tr := range st.users {
			if tr.Status == Planned {
				alive = true
			}
		}
		if !alive {
			continue
		}
		if ctx.Err() != nil {
			interrupted = true
			break
		}
		if f := execOps(ctx, c, u, st.step.Prepare, filepath.Join(logDir, stamp+"-"+st.step.Key+"-prepare.log"), req.Verbose); f != nil {
			failStep(c, u, st, f)
			continue
		}
		runStep(ctx, c, u, st, logDir, stamp, req.Verbose)
		if errors.Is(ctx.Err(), context.Canceled) {
			interrupted = true
		}
		if !st.failed && !interrupted {
			if f := execOps(ctx, c, u, st.step.After, filepath.Join(logDir, stamp+"-"+st.step.Key+"-after.log"), req.Verbose); f != nil {
				failStep(c, u, st, f)
				continue
			}
		}
		for _, tr := range st.users {
			if tr.Status != Planned || !allDone(tr, steps) || interrupted {
				continue
			}
			packageTarget(ctx, c, u, tr, steps, logDir, stamp, req.Verbose)
			if ctx.Err() != nil {
				interrupted = true
			}
		}
		if interrupted {
			break
		}
	}
	if interrupted {
		s.Interrupted = true
		for _, tr := range s.Targets {
			if tr.Status == Planned {
				tr.Status, tr.Reason = Skipped, i18n.S("interrupted", "已中断")
			}
		}
	}

	// 5. Checksums.
	checksums := ""
	if c.Config.Checksums() && anySuccess(s) {
		alg := orSHA256(c.Config.Output.ChecksumAlgorithm)
		p, sums, err := pack.WriteChecksumsAlg(c.OutDir, alg)
		if err == nil {
			checksums = p
			for _, tr := range s.Targets {
				for i := range tr.Artifacts {
					if alg == "sha512" {
						tr.Artifacts[i].SHA512 = sums[tr.Artifacts[i].File]
					} else {
						tr.Artifacts[i].SHA256 = sums[tr.Artifacts[i].File]
					}
				}
			}
		} else {
			u.Warn(i18n.S("could not write checksums: ", "无法写入校验和：") + err.Error())
		}
	}

	// 6. post_build hooks (also after failures; FPACK_SUCCESS tells which).
	if !interrupted {
		var paths []string
		for _, tr := range s.Targets {
			for _, a := range tr.Artifacts {
				if tr.Status == Success {
					paths = append(paths, a.Path)
				}
			}
		}
		if ops := targets.BuildHookOps(c, true, paths, !anyFailed(s)); len(ops) > 0 && (anySuccess(s) || anyFailed(s)) {
			u.Blank()
			u.Step("hooks.post_build")
			if f := execOps(ctx, c, u, ops, filepath.Join(logDir, stamp+"-hooks.log"), req.Verbose); f != nil {
				s.HookError = f.err
				u.Fail(f.err)
				printFailure(c, u, f.excerpt, f.hint, f.hint != "", f.log)
			}
		}
	}
	return finish(c, u, s, start, checksums, logDir)
}

func appendUnique(list []string, add ...string) []string {
	for _, a := range add {
		dup := false
		for _, l := range list {
			if l == a {
				dup = true
			}
		}
		if !dup && a != "" {
			list = append(list, a)
		}
	}
	return list
}

func allDone(tr *TargetResult, steps map[string]*stepState) bool {
	for _, k := range tr.steps {
		if !steps[k].done {
			return false
		}
	}
	return true
}

func anyFailed(s *Summary) bool {
	for _, t := range s.Targets {
		if t.Status == Failed {
			return true
		}
	}
	return false
}

func hasPlanned(s *Summary) bool {
	for _, t := range s.Targets {
		if t.Status == Planned {
			return true
		}
	}
	return false
}

func orSHA256(a string) string {
	if a == "" {
		return "sha256"
	}
	return a
}

func anySuccess(s *Summary) bool {
	for _, t := range s.Targets {
		if t.Status == Success {
			return true
		}
	}
	return false
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func flutterCmd(c *targets.Context, st targets.FlutterStep) runner.Cmd {
	return runner.Cmd{Name: c.SDK.Flutter, Args: st.Args, Dir: c.Project.Root, Env: st.Env, Secret: st.Secret}
}

func shortFlutter(st targets.FlutterStep) string {
	cmd := runner.Cmd{Name: "flutter", Args: st.Args, Secret: st.Secret}
	return cmd.String()
}

func platformTitle(p host.Platform) string {
	switch p {
	case host.Android:
		return "Android"
	case host.IOS:
		return "iOS"
	case host.MacOS:
		return "macOS"
	case host.Windows:
		return "Windows"
	case host.Linux:
		return "Linux"
	case host.Web:
		return "Web"
	}
	return string(p)
}

// heartbeat is the periodic progress line printed in non-interactive runs.
// Flutter is often silent for minutes (xcodebuild archive, Gradle), so a
// stale last line is labelled with its age instead of looking current.
func heartbeat(elapsed time.Duration, last string, age time.Duration) string {
	run := ui.Duration(elapsed.Truncate(time.Second))
	if last == "" {
		return i18n.F("… still running (%s)", "… 仍在运行（%s）", run)
	}
	if age >= 45*time.Second {
		quiet := ui.Duration(age.Truncate(time.Second))
		if age < time.Minute {
			quiet = fmt.Sprintf("%ds", int(age.Seconds()))
		}
		return i18n.F("… still running (%s) · no output for %s, last: %s", "… 仍在运行（%s）· 已 %s 无输出，最后一行：%s", run, quiet, ui.Truncate(last, 60))
	}
	return i18n.F("… still running (%s) %s", "… 仍在运行（%s）%s", run, ui.Truncate(last, 80))
}

func runStep(ctx context.Context, c *targets.Context, u *ui.UI, st *stepState, logDir, stamp string, verbose bool) {
	var names []string
	for _, tr := range st.users {
		if tr.Status == Planned {
			names = append(names, tr.Target)
		}
	}
	u.Blank()
	u.Step(fmt.Sprintf("%s · %s  %s", platformTitle(st.step.Platform), strings.Join(names, ", "), u.Dim(shortFlutter(st.step))))
	for _, w := range st.step.Warnings {
		u.Warn(w)
	}
	st.logPath = filepath.Join(logDir, stamp+"-flutter-"+st.step.Key+".log")
	st.start = time.Now()
	for _, tr := range st.users {
		if tr.start.IsZero() {
			tr.start = st.start
		}
		tr.attempted = true
	}
	label := "flutter build " + st.step.Args[1]
	sp := u.StartSpinner(label)
	var (
		mu     sync.Mutex
		last   string
		lastAt = time.Now()
	)
	stopBeat := make(chan struct{})
	if !u.Interactive() && !verbose {
		go func() {
			t := time.NewTicker(60 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-stopBeat:
					return
				case <-t.C:
					mu.Lock()
					l, at := last, lastAt
					mu.Unlock()
					u.Detail(heartbeat(time.Since(st.start), l, time.Since(at)))
				}
			}
		}()
	}
	opts := runner.Options{LogPath: st.logPath, OnLine: func(l string) {
		l = strings.TrimSpace(l)
		mu.Lock()
		last, lastAt = l, time.Now()
		mu.Unlock()
		sp.Update(l)
	}}
	if verbose {
		opts.Stream = u.Writer()
	}
	res, err := runner.Run(ctx, flutterCmd(c, st.step), opts)
	close(stopBeat)
	sp.Stop()
	st.done = true
	if err != nil {
		st.failed = true
		if errors.Is(err, runner.ErrInterrupted) {
			u.Fail(i18n.S("interrupted", "已中断"))
			for _, tr := range st.users {
				tr.Status, tr.Reason = Skipped, i18n.S("interrupted", "已中断")
			}
			return
		}
		u.Fail(fmt.Sprintf("%s %s", label, i18n.F("failed after %s (exit %d)", "失败，用时 %s（退出码 %d）", ui.Duration(res.Duration), res.ExitCode)))
		excerpt := hints.Excerpt(res.Tail, 25)
		h, hasHint := hints.Match(res.Tail)
		for _, tr := range st.users {
			if tr.Status != Planned {
				continue
			}
			tr.Status = Failed
			tr.Error = fmt.Sprintf("%s: %v", label, err)
			if res.ExitCode == 127 {
				tr.Error = err.Error()
			}
			tr.Excerpt = excerpt
			tr.Log = st.logPath
			if hasHint {
				tr.Hint = h.Text()
			}
			tr.duration = time.Since(tr.start)
			// Possible causes found before the build (missing certificates, …).
			for _, w := range tr.ifFails {
				tr.Warnings = appendUnique(tr.Warnings, w)
				u.Warn(strings.ReplaceAll(w, "\n", "\n    "))
			}
		}
		printFailure(c, u, excerpt, h.Text(), hasHint, st.logPath)
		return
	}
	u.Success(fmt.Sprintf("%s %s", label, u.Dim("("+ui.Duration(res.Duration)+")")))
}

func printFailure(c *targets.Context, u *ui.UI, excerpt []string, hint string, hasHint bool, log string) {
	if len(excerpt) > 0 {
		u.Println("")
		for _, l := range excerpt {
			u.Println("    " + u.Red("│") + " " + l)
		}
		u.Println("")
	}
	if hasHint {
		u.Hint(hint)
	}
	if log != "" {
		u.Detail(i18n.S("full log: ", "完整日志：") + c.Rel(log))
	}
}

func packageTarget(ctx context.Context, c *targets.Context, u *ui.UI, tr *TargetResult, steps map[string]*stepState, logDir, stamp string, verbose bool) {
	var since time.Time
	for _, k := range tr.steps {
		if since.IsZero() || steps[k].start.Before(since) {
			since = steps[k].start
		}
	}
	defer func() { tr.duration = time.Since(tr.start) }()
	in, err := tr.target.Locate(c, false, since)
	if err != nil {
		tr.Status, tr.Error = Failed, err.Error()
		u.Fail(tr.Target + ": " + err.Error())
		return
	}
	p, err := tr.target.Package(c, in)
	if err != nil {
		tr.Status, tr.Error = Failed, err.Error()
		u.Fail(tr.Target + ": " + err.Error())
		return
	}
	tr.Notes = appendUnique(tr.Notes, p.Notes...)
	tr.Warnings = appendUnique(tr.Warnings, p.Warnings...)
	logPath := filepath.Join(logDir, stamp+"-"+tr.Target+".log")
	var notes []string
	f := runOps(ctx, c, u, p.Ops, logPath, verbose, &notes, &tr.Warnings)
	if f != nil {
		if f.interrupted {
			tr.Status, tr.Reason = Skipped, i18n.S("interrupted", "已中断")
			return
		}
		tr.Status, tr.Error, tr.Excerpt, tr.Hint = Failed, f.err, f.excerpt, f.hint
		if f.cmd {
			tr.Log = logPath
		}
		u.Fail(tr.Target + ": " + tr.Error)
		if f.cmd {
			printFailure(c, u, tr.Excerpt, tr.Hint, tr.Hint != "", logPath)
		}
		return
	}
	for _, n := range notes {
		if strings.HasPrefix(n, "WARN:") {
			tr.Warnings = appendUnique(tr.Warnings, strings.TrimPrefix(n, "WARN:"))
		} else {
			tr.Notes = appendUnique(tr.Notes, n)
		}
	}
	tr.Artifacts = tr.Artifacts[:0]
	for _, a := range p.Artifacts {
		st, err := os.Stat(a.Path)
		if err != nil {
			tr.Status, tr.Error = Failed, i18n.F("expected artifact missing: %s", "缺少预期产物：%s", c.Rel(a.Path))
			u.Fail(tr.Target + ": " + tr.Error)
			return
		}
		ar := ArtifactResult{Path: a.Path, File: filepath.Base(a.Path), Kind: a.Kind, Arch: a.Arch, Size: st.Size()}
		extra := ""
		if ns, ok := c.NotaryFor(a.Path); ok {
			ar.Notarization = &Notarization{State: ns.State(), ID: ns.ID, Status: ns.Status, Record: filepath.Join(c.OutDir, targets.NotaryMDFile)}
			extra = "  " + u.Dim(i18n.F("notarization: %s", "公证：%s", ns.State()))
		}
		tr.Artifacts = append(tr.Artifacts, ar)
		u.Success(fmt.Sprintf("%s  %s%s", u.Bold(c.Rel(a.Path)), u.Dim(ui.Size(st.Size())), extra))
	}
	for _, w := range tr.Warnings {
		u.Warn(firstLineOf(w))
	}
	tr.Status = Success
}

func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// projectIssues are problems that make every flutter build fail early.
func projectIssues(c *targets.Context) []targets.Issue {
	var out []targets.Issue
	for _, d := range c.Project.MissingPathDeps() {
		out = append(out, targets.Issue{Fatal: true,
			Msg: i18n.F("path dependency %s → %s does not exist", "path 依赖 %s → %s 不存在", d.Name, d.Path),
			Fix: i18n.S("clone/copy that package to the expected location (relative to the project), then rerun", "请把该包克隆/复制到对应位置（相对于项目目录）后重试")})
	}
	return out
}

// opFailure describes a failed operation.
type opFailure struct {
	err, hint, log string
	excerpt        []string
	cmd            bool // an external command failed (log available)
	interrupted    bool
}

// execOps runs ops (hooks, step prepare/after) and reports the first failure.
func execOps(ctx context.Context, c *targets.Context, u *ui.UI, ops []targets.Op, logPath string, verbose bool) *opFailure {
	if len(ops) == 0 {
		return nil
	}
	var notes, warns []string
	f := runOps(ctx, c, u, ops, logPath, verbose, &notes, &warns)
	for _, w := range warns {
		u.Warn(firstLineOf(w))
	}
	if f != nil {
		f.log = logPath
	}
	return f
}

// runOps executes operations in order. Notes from Check functions and
// warnings from optional ops are appended to notes/warns.
func runOps(ctx context.Context, c *targets.Context, u *ui.UI, ops []targets.Op, logPath string, verbose bool, notes, warns *[]string) *opFailure {
	for _, op := range ops {
		if ctx.Err() != nil {
			return &opFailure{interrupted: true, err: i18n.S("interrupted", "已中断")}
		}
		if op.Fn != nil {
			if err := op.Fn(); err != nil {
				return &opFailure{err: op.Desc + ": " + err.Error(), hint: op.Hint}
			}
			continue
		}
		if op.Run != nil {
			sp := u.StartSpinner(op.Desc)
			var last runner.Result
			env := OpEnvFor(ctx, c, u, sp, logPath, verbose, &last)
			note, err := op.Run(ctx, env)
			sp.Stop()
			if note != "" {
				*notes = append(*notes, note)
			}
			if err != nil {
				if errors.Is(err, runner.ErrInterrupted) || ctx.Err() != nil {
					return &opFailure{interrupted: true, err: i18n.S("interrupted", "已中断")}
				}
				if op.Optional {
					*warns = append(*warns, op.Desc+": "+err.Error())
					continue
				}
				f := &opFailure{err: op.Desc + ": " + err.Error(), excerpt: hints.Excerpt(last.Tail, 20), cmd: true, hint: op.Hint}
				if h, ok := hints.Match([]string{err.Error()}); ok {
					f.hint = h.Text()
				}
				return f
			}
			if verbose || !u.Interactive() {
				u.Detail("✓ " + op.Desc)
			}
			continue
		}
		if op.Cmd == nil {
			continue
		}
		cmdCopy := *op.Cmd
		if cmdCopy.Dir == "" {
			cmdCopy.Dir = c.Project.Root
		}
		sp := u.StartSpinner(op.Desc)
		opts := runner.Options{LogPath: logPath, OnLine: func(l string) { sp.Update(l) }}
		if verbose {
			opts.Stream = u.Writer()
		}
		res, err := runner.Run(ctx, cmdCopy, opts)
		sp.Stop()
		note := ""
		if op.Check != nil && !errors.Is(err, runner.ErrInterrupted) {
			var cerr error
			note, cerr = op.Check(res)
			if cerr != nil && err == nil {
				err = cerr
			} else if cerr != nil {
				err = fmt.Errorf("%v: %v", err, cerr)
			}
		}
		if note != "" {
			*notes = append(*notes, note)
		}
		if err != nil {
			if errors.Is(err, runner.ErrInterrupted) {
				return &opFailure{interrupted: true, err: i18n.S("interrupted", "已中断")}
			}
			if op.Optional {
				*warns = append(*warns, op.Desc+": "+err.Error())
				continue
			}
			f := &opFailure{err: op.Desc + ": " + err.Error(), excerpt: hints.Excerpt(res.Tail, 20), cmd: true}
			h, ok := hints.Match(res.Tail)
			if !ok {
				h, ok = hints.Match([]string{err.Error()})
			}
			if ok {
				f.hint = h.Text()
			} else {
				f.hint = op.Hint
			}
			return f
		}
		if verbose || !u.Interactive() {
			u.Detail("✓ " + op.Desc)
		}
	}
	return nil
}

// OpEnvFor gives Op.Run functions access to the runner and the UI. The
// spinner may be nil (no animation).
func OpEnvFor(ctx context.Context, c *targets.Context, u *ui.UI, sp *ui.Spinner, logPath string, verbose bool, last *runner.Result) targets.OpEnv {
	return targets.OpEnv{
		Interactive: u.Interactive() && !verbose,
		Run: func(cm runner.Cmd) (runner.Result, error) {
			if cm.Dir == "" && c.Project != nil {
				cm.Dir = c.Project.Root
			}
			opts := runner.Options{LogPath: logPath}
			if sp != nil {
				opts.OnLine = func(l string) { sp.Update(l) }
			}
			if verbose {
				opts.Stream = u.Writer()
			}
			res, err := runner.Run(ctx, cm, opts)
			if last != nil {
				*last = res
			}
			return res, err
		},
		Status: func(d string) {
			if sp != nil {
				sp.Update(d)
			}
		},
		Print: func(kind, msg string) {
			switch kind {
			case "ok":
				u.Success(msg)
			case "warn":
				u.Warn(msg)
			case "hint":
				u.Hint(msg)
			case "detail":
				u.Detail(msg)
			default:
				u.Info(msg)
			}
		},
	}
}

// failStep marks every target of a step as failed by a prepare/after op.
func failStep(c *targets.Context, u *ui.UI, st *stepState, f *opFailure) {
	st.done, st.failed = true, true
	for _, tr := range st.users {
		if tr.Status != Planned {
			continue
		}
		tr.attempted = true
		if tr.start.IsZero() {
			tr.start = time.Now()
		}
		tr.Status, tr.Error, tr.Excerpt, tr.Hint, tr.Log = Failed, f.err, f.excerpt, f.hint, f.log
		tr.duration = time.Since(tr.start)
	}
	u.Fail(f.err)
	printFailure(c, u, f.excerpt, f.hint, f.hint != "", f.log)
}
