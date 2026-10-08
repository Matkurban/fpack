package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/build"
	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
	"github.com/Matkurban/fpack/go/internal/targets"
	"github.com/Matkurban/fpack/go/internal/ui"
)

func notarizeCommand() *command {
	c := &command{name: "notarize", en: "check or finish macOS notarization submissions (after Ctrl-C or --notarize-no-wait)", zh: "查询或完成 macOS 公证提交（Ctrl-C 或 --notarize-no-wait 之后）"}
	c.help = func() string {
		return cmdHelp(c, "fpack notarize status [DIR|ID]\n       fpack notarize finish [DIR]",
			i18n.S(`  fpack notarize status                  # every submission in the newest notarization.json
  fpack notarize status dist/1.2.0+5     # a specific output directory
  fpack notarize status 2efe2717-52ef-…  # one submission id
  fpack notarize finish                  # wait, staple accepted files, update SHA256SUMS

status asks Apple for the state of each submission recorded in
<output>/notarization.json (written by fpack build right after the upload).
finish waits for pending submissions, staples the ticket to accepted
DMG/pkg/app zips (the app is unzipped, stapled and zipped again), rewrites
the checksum file, and for rejected ones downloads and summarizes Apple's log.
Credentials come from fpack.yaml macos.sign (or the recorded keychain
profile / API key; Apple ID submissions need FPACK_NOTARY_PASSWORD).
`, `  fpack notarize status                  # 最新 notarization.json 中的所有提交
  fpack notarize status dist/1.2.0+5     # 指定输出目录
  fpack notarize status 2efe2717-52ef-…  # 指定提交 ID
  fpack notarize finish                  # 等待、装订已通过的文件、更新 SHA256SUMS

status 向 Apple 查询 <输出目录>/notarization.json（fpack build 上传后立即写入）
中记录的每个提交的状态。finish 等待未完成的提交，为已通过的 DMG/pkg/App zip
装订票据（App 会先解压、装订再重新压缩），重写校验和文件；对被拒绝的提交下载并
总结 Apple 的日志。凭证来自 fpack.yaml 的 macos.sign（或记录中的钥匙串配置 /
API 密钥；Apple ID 提交需要设置 FPACK_NOTARY_PASSWORD）。
`))
	}
	c.run = runNotarizeCmd
	return c
}

var uuidLike = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F-]{4,}$`)

func runNotarizeCmd(e *Env, p *parsed) int {
	u := newUI(e, p)
	if len(p.pos) == 0 || (p.pos[0] != "status" && p.pos[0] != "finish") {
		u.Errorf("%s", i18n.S("usage: fpack notarize status [DIR|ID] | fpack notarize finish [DIR]", "用法：fpack notarize status [目录|ID] | fpack notarize finish [目录]"))
		return build.ExitUsage
	}
	sub, arg := p.pos[0], ""
	if len(p.pos) > 1 {
		arg = p.pos[1]
	}
	if len(p.pos) > 2 {
		u.Errorf("%s", i18n.S("too many arguments", "参数过多"))
		return build.ExitUsage
	}
	id := ""
	if arg != "" && uuidLike.MatchString(arg) {
		if _, err := os.Stat(arg); err != nil {
			id, arg = arg, ""
		}
	}
	if sub == "finish" && id != "" {
		u.Errorf("%s", i18n.S("finish takes an output directory, not a submission id", "finish 的参数是输出目录，而不是提交 ID"))
		return build.ExitUsage
	}

	ctxT, cerr := build.NewContext(build.Options{ProjectDir: p.s("project"), ConfigPath: p.s("config"), FlutterPath: p.s("flutter"), Getenv: e.Getenv})
	if cerr != nil && arg == "" {
		return contextError(u, cerr)
	}
	if cerr != nil {
		// Outside a project: work from the recorded data only.
		ctxT = &targets.Context{Config: &config.Config{}, WorkDir: filepath.Join(os.TempDir(), "fpack")}
	}
	dir, err := notaryDir(ctxT, arg)
	if err != nil {
		u.Errorf("%v", err)
		u.Hint(i18n.S("notarization.json is written into the output directory by `fpack build macos|dmg|pkg` when notarization is on", "开启公证时，`fpack build macos|dmg|pkg` 会把 notarization.json 写入输出目录"))
		return build.ExitPrereq
	}
	if ctxT.OutDir == "" {
		ctxT.OutDir = dir
	}
	rec, err := targets.LoadNotary(dir)
	if err != nil {
		u.Errorf("%v", err)
		return build.ExitFailed
	}
	if id != "" && !hasSubmission(rec, id) {
		rec.Submissions = []targets.NotarySubmission{{ID: id, Artifact: filepath.Join(dir, id), Status: targets.NotarySubmitted}}
		if len(ctxT.Mac.NotaryAuthDisplay()) > 0 {
			rec.Submissions[0].AuthArgs = ctxT.Mac.NotaryAuthDisplay()
		}
		dir = "" // nothing to write back
	}
	if len(rec.Submissions) == 0 {
		u.Info(i18n.F("no submissions recorded in %s", "%s 中没有提交记录", filepath.Join(dir, targets.NotaryJSONFile)))
		return 0
	}
	rel := func(path string) string {
		if ctxT.Project != nil {
			return ctxT.Rel(path)
		}
		return path
	}
	ctx, stop := signalContext(u)
	defer stop()
	logPath := filepath.Join(ctxT.WorkDir, "logs", time.Now().Format("20060102-150405")+"-notarize.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	verbose := p.b("verbose")
	md := filepath.Join(dir, targets.NotaryMDFile)
	if dir != "" {
		u.Title(i18n.F("Notarization · %s", "公证 · %s", rel(filepath.Join(dir, targets.NotaryJSONFile))))
	}

	failed, stapled, interrupted := 0, 0, false
	for i := range rec.Submissions {
		s := &rec.Submissions[i]
		if id != "" && s.ID != id {
			continue
		}
		if ctx.Err() != nil {
			interrupted = true
			break
		}
		name := filepath.Base(s.Artifact)
		if s.Stapled {
			u.Success(fmt.Sprintf("%s  %s", u.Bold(name), u.Dim(i18n.F("Accepted, stapled · id %s", "已通过，已装订 · id %s", s.ID))))
			continue
		}
		auth, secrets, err := ctxT.NotaryAuthFor(*s, e.Getenv)
		if err != nil {
			u.Fail(fmt.Sprintf("%s: %v", name, err))
			failed++
			continue
		}
		step := func(label string, fn func(env targets.OpEnv) error) error {
			sp := u.StartSpinner(label)
			env := build.OpEnvFor(ctx, ctxT, u, sp, logPath, verbose, nil)
			err := fn(env)
			sp.Stop()
			return err
		}
		if sub == "status" || s.Final() {
			if err := step(i18n.F("%s: asking Apple", "%s：查询 Apple", name), func(env targets.OpEnv) error { return targets.NotaryInfo(env, s, auth, secrets) }); err != nil {
				if errors.Is(err, runner.ErrInterrupted) {
					interrupted = true
					break
				}
				u.Fail(fmt.Sprintf("%s: %v", name, err))
				u.Detail(i18n.S("log: ", "日志：") + rel(logPath))
				failed++
				continue
			}
		}
		if sub == "finish" && !s.Final() {
			u.Info(i18n.F("%s: waiting for Apple (id %s, submitted %s ago)", "%s：等待 Apple（id %s，%s 前提交）", name, s.ID, ui.Duration(time.Since(s.SubmittedAt).Truncate(time.Second))))
			u.Hint(targets.NotaryWaitHint(rel(md)))
			err := step(i18n.F("%s: notarizing", "%s：公证中", name), func(env targets.OpEnv) error {
				return targets.NotaryWait(ctx, env, s, auth, secrets, filepath.Dir(s.Artifact))
			})
			if errors.Is(err, runner.ErrInterrupted) {
				interrupted = true
				break
			}
			if err != nil {
				u.Fail(fmt.Sprintf("%s: %v", name, err))
				failed++
				continue
			}
		}
		switch {
		case s.Status == targets.NotaryAccepted && sub == "finish":
			if !fileExists(s.Artifact) && s.Uploaded != "" && fileExists(s.Uploaded) {
				_ = pack.MoveFile(s.Uploaded, s.Artifact)
			}
			if !fileExists(s.Artifact) {
				u.Fail(i18n.F("%s: Accepted, but the file is missing: %s", "%s：已通过，但文件不存在：%s", name, rel(s.Artifact)))
				failed++
				continue
			}
			err := step(i18n.F("%s: stapling", "%s：装订票据", name), func(env targets.OpEnv) error { return targets.NotaryStaple(env, s, ctxT.WorkDir) })
			if err != nil {
				if errors.Is(err, runner.ErrInterrupted) {
					interrupted = true
					break
				}
				u.Fail(fmt.Sprintf("%s: %v", name, err))
				u.Detail(i18n.S("log: ", "日志：") + rel(logPath))
				failed++
				continue
			}
			stapled++
			u.Success(fmt.Sprintf("%s  %s", u.Bold(name), i18n.F("Accepted, ticket stapled · id %s", "已通过，票据已装订 · id %s", s.ID)))
		case s.Status == targets.NotaryInvalid || s.Status == targets.NotaryRejected:
			if s.Log == "" && sub == "finish" {
				_ = step(i18n.F("%s: downloading Apple's log", "%s：下载 Apple 日志", name), func(env targets.OpEnv) error {
					targets.NotaryFetchLog(env, s, auth, secrets, filepath.Dir(s.Artifact))
					return nil
				})
			}
			u.Fail(fmt.Sprintf("%s  %s", u.Bold(name), i18n.F("%s: %s · id %s", "%s：%s · id %s", s.Status, s.Message, s.ID)))
			for _, is := range s.Issues {
				u.Detail(is)
			}
			if s.Log != "" {
				u.Detail(i18n.S("Apple's log: ", "Apple 日志：") + rel(s.Log))
			}
			failed++
		case s.Status == targets.NotaryAccepted:
			u.Success(fmt.Sprintf("%s  %s", u.Bold(name), i18n.F("Accepted, not stapled yet · id %s → fpack notarize finish", "已通过，尚未装订 · id %s → fpack notarize finish", s.ID)))
		default:
			u.Info(fmt.Sprintf("%s %s  %s", u.Yellow("…"), u.Bold(name), i18n.F("%s · id %s · submitted %s ago", "%s · id %s · %s 前提交", s.Status, s.ID, ui.Duration(time.Since(s.SubmittedAt).Truncate(time.Second)))))
		}
	}
	if dir != "" {
		if _, err := targets.SaveNotary(dir, rec); err != nil {
			u.Warn(err.Error())
		}
	}
	if stapled > 0 && dir != "" && ctxT.Config.Checksums() {
		alg := ctxT.Config.Output.ChecksumAlgorithm
		if path, _, err := pack.WriteChecksumsAlg(dir, alg); err == nil {
			u.Success(i18n.F("updated %s", "已更新 %s", rel(path)))
		} else {
			u.Warn(err.Error())
		}
	}
	if p.b("json") {
		printJSON(e, rec)
	}
	if dir != "" {
		u.Detail(i18n.F("details and commands: %s", "详情与命令：%s", rel(md)))
	}
	if interrupted {
		u.Warn(i18n.F("stopped waiting – Apple keeps processing; check later with the commands in %s", "已停止等待 —— Apple 会继续处理；之后可用 %s 中的命令查询", rel(md)))
		return build.ExitInterrupted
	}
	if failed > 0 {
		return build.ExitFailed
	}
	return 0
}

func hasSubmission(r *targets.NotaryRecord, id string) bool {
	for _, s := range r.Submissions {
		if s.ID == id {
			return true
		}
	}
	return false
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// notaryDir finds the directory holding notarization.json: the argument,
// the current output directory, or the newest one below the output root.
func notaryDir(c *targets.Context, arg string) (string, error) {
	if arg != "" {
		abs, _ := filepath.Abs(arg)
		if filepath.Base(abs) == targets.NotaryJSONFile {
			abs = filepath.Dir(abs)
		}
		if fileExists(filepath.Join(abs, targets.NotaryJSONFile)) {
			return abs, nil
		}
		return "", fmt.Errorf("%s", i18n.F("no %s in %s", "%[2]s 中没有 %[1]s", targets.NotaryJSONFile, arg))
	}
	if c.OutDir != "" && fileExists(filepath.Join(c.OutDir, targets.NotaryJSONFile)) {
		return c.OutDir, nil
	}
	if c.Project == nil {
		return "", fmt.Errorf("%s", i18n.F("no %s found", "找不到 %s", targets.NotaryJSONFile))
	}
	root := staticOutputRoot(c)
	var found []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.Count(strings.TrimPrefix(path, root), string(filepath.Separator)) > 3 {
			return filepath.SkipDir
		}
		if d.Name() == targets.NotaryJSONFile {
			found = append(found, filepath.Dir(path))
		}
		return nil
	})
	if len(found) == 0 {
		return "", fmt.Errorf("%s", i18n.F("no %s found in %s", "%[2]s 中找不到 %[1]s", targets.NotaryJSONFile, c.Rel(root)))
	}
	sort.Slice(found, func(i, j int) bool { return modTime(found[i]) > modTime(found[j]) })
	return found[0], nil
}

func modTime(dir string) int64 {
	st, err := os.Stat(filepath.Join(dir, targets.NotaryJSONFile))
	if err != nil {
		return 0
	}
	return st.ModTime().UnixNano()
}
