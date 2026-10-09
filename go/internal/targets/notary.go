package targets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
	"github.com/Matkurban/fpack/go/internal/version"
)

// Notarization records: every submission to Apple's notary service is
// written to <output>/notarization.json (machine-readable) and
// <output>/NOTARIZATION.md (copy-paste commands) right after the upload,
// and updated when the result is known. `fpack notarize status|finish`
// read them later (after Ctrl-C or with macos.notarize.wait: false).

const (
	NotaryJSONFile = "notarization.json"
	NotaryMDFile   = "NOTARIZATION.md"
)

// Apple's status values plus fpack's own "Submitted" (uploaded, no verdict yet).
const (
	NotarySubmitted = "Submitted"
	NotaryAccepted  = "Accepted"
	NotaryInvalid   = "Invalid"
	NotaryRejected  = "Rejected"
	NotaryProgress  = "In Progress"
)

// NotarySubmission is one uploaded file.
type NotarySubmission struct {
	Target      string    `json:"target"`                 // macos | dmg | pkg
	Artifact    string    `json:"artifact"`               // final file in the output directory
	Uploaded    string    `json:"uploaded"`               // file sent to Apple (a staging copy until the build finishes)
	App         string    `json:"app,omitempty"`          // .app name inside a zip (staple target)
	SHA256      string    `json:"sha256"`                 // of the uploaded file
	FinalSHA256 string    `json:"final_sha256,omitempty"` // of the artifact after stapling
	ID          string    `json:"id"`
	SubmittedAt time.Time `json:"submitted_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Auth        string    `json:"auth"`      // credential kind and name, never secrets
	AuthArgs    []string  `json:"auth_args"` // notarytool credential arguments; secrets as $ENV placeholders
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	WaitSeconds int       `json:"wait_seconds,omitempty"`
	Stapled     bool      `json:"stapled"`
	Log         string    `json:"log,omitempty"` // Apple's log (JSON) for rejected submissions
	Issues      []string  `json:"issues,omitempty"`
}

// Final reports whether Apple has answered.
func (s NotarySubmission) Final() bool {
	return s.Status == NotaryAccepted || s.Status == NotaryInvalid || s.Status == NotaryRejected
}

// State is the short summary state: submitted, accepted, stapled, invalid.
func (s NotarySubmission) State() string {
	switch {
	case s.Stapled:
		return "stapled"
	case s.Status == NotaryAccepted:
		return "accepted"
	case s.Status == NotaryInvalid || s.Status == NotaryRejected:
		return "invalid"
	}
	return "submitted"
}

// NotaryRecord is the content of notarization.json.
type NotaryRecord struct {
	Schema      int                `json:"schema"`
	Tool        string             `json:"tool"`
	App         string             `json:"app,omitempty"`
	Version     string             `json:"version,omitempty"`
	Updated     time.Time          `json:"updated"`
	Submissions []NotarySubmission `json:"submissions"`
}

var notaryFileMu sync.Mutex

// LoadNotary reads dir/notarization.json (an empty record if missing).
func LoadNotary(dir string) (*NotaryRecord, error) {
	r := &NotaryRecord{Schema: 1}
	data, err := os.ReadFile(filepath.Join(dir, NotaryJSONFile))
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("%s: %v", filepath.Join(dir, NotaryJSONFile), err)
	}
	return r, nil
}

// Upsert adds or replaces the submission for the same artifact.
func (r *NotaryRecord) Upsert(s NotarySubmission) {
	for i := range r.Submissions {
		if r.Submissions[i].Artifact == s.Artifact || (s.ID != "" && r.Submissions[i].ID == s.ID) {
			r.Submissions[i] = s
			return
		}
	}
	r.Submissions = append(r.Submissions, s)
	sort.SliceStable(r.Submissions, func(i, j int) bool { return r.Submissions[i].SubmittedAt.Before(r.Submissions[j].SubmittedAt) })
}

// SaveNotary writes notarization.json and NOTARIZATION.md; it returns the
// Markdown path.
func SaveNotary(dir string, r *NotaryRecord) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	r.Schema, r.Tool, r.Updated = 1, "fpack "+version.Version, time.Now()
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, NotaryJSONFile), append(data, '\n')); err != nil {
		return "", err
	}
	md := filepath.Join(dir, NotaryMDFile)
	return md, writeAtomic(md, []byte(NotaryMarkdown(r, dir)))
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".partial"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// recordNotary merges s into the output directory's record.
func recordNotary(c *Context, s NotarySubmission) (string, error) {
	notaryFileMu.Lock()
	defer notaryFileMu.Unlock()
	s.UpdatedAt = time.Now()
	r, err := LoadNotary(c.OutDir)
	if err != nil {
		r = &NotaryRecord{} // a corrupt record must not fail the build
	}
	r.App, r.Version = c.DisplayName(), c.BuildName+"+"+c.BuildNumber
	r.Upsert(s)
	c.notaryMu.Lock()
	if c.notary == nil {
		c.notary = map[string]NotarySubmission{}
	}
	c.notary[s.Artifact] = s
	c.notaryMu.Unlock()
	return SaveNotary(c.OutDir, r)
}

// NotaryFor returns the submission recorded in this run for an artifact.
func (c *Context) NotaryFor(artifact string) (NotarySubmission, bool) {
	c.notaryMu.Lock()
	defer c.notaryMu.Unlock()
	s, ok := c.notary[artifact]
	return s, ok
}

// ------------------------------------------------------------- commands --

// NotaryAuthDisplay is NotaryAuth with secrets replaced by an environment
// variable reference, safe to write to files.
func (m MacSigning) NotaryAuthDisplay() []string {
	a, _, _ := m.NotaryAuth()
	out := append([]string(nil), a...)
	for i := range out {
		if i > 0 && out[i-1] == "--password" {
			out[i] = "$FPACK_NOTARY_PASSWORD"
		}
	}
	return out
}

func shellArgs(args []string) string {
	var p []string
	for _, a := range args {
		if strings.HasPrefix(a, "$") && !strings.ContainsAny(a[1:], " $'\"") {
			p = append(p, `"`+a+`"`)
		} else {
			p = append(p, runner.Quote(a))
		}
	}
	return strings.Join(p, " ")
}

// NotaryCommands are copy-paste commands for a submission (run from dir).
type NotaryCommands struct {
	Status, Wait, Log, Staple, Validate, Assess string
}

// Commands returns the commands for a submission; paths are relative to
// base when possible.
func (s NotarySubmission) Commands(base string) NotaryCommands {
	rel := func(p string) string {
		if r, err := filepath.Rel(base, p); err == nil && !strings.HasPrefix(r, "..") {
			return r
		}
		return p
	}
	auth := shellArgs(s.AuthArgs)
	file := runner.Quote(rel(s.Artifact))
	logName := "notary-log-" + filepath.Base(s.Artifact) + ".json"
	nc := NotaryCommands{
		Status: fmt.Sprintf("xcrun notarytool info %s %s", s.ID, auth),
		Wait:   fmt.Sprintf("xcrun notarytool wait %s %s", s.ID, auth),
		Log:    fmt.Sprintf("xcrun notarytool log %s %s %s", s.ID, auth, runner.Quote(logName)),
	}
	switch s.Target {
	case "macos":
		app := runner.Quote(orDefault(s.App, "App.app"))
		nc.Staple = fmt.Sprintf("ditto -x -k %s stapled && xcrun stapler staple stapled/%s && rm %s && ditto -c -k --sequesterRsrc --keepParent stapled/%s %s && rm -rf stapled", file, app, file, app, file)
		nc.Validate = fmt.Sprintf("xcrun stapler validate stapled/%s   # %s", app, i18n.S("before the final rm -rf", "在最后的 rm -rf 之前执行"))
		nc.Assess = fmt.Sprintf("spctl -a -vv -t exec stapled/%s", app)
	case "pkg":
		nc.Staple = "xcrun stapler staple " + file
		nc.Validate = "xcrun stapler validate " + file
		nc.Assess = "spctl -a -vv -t install " + file
	default:
		nc.Staple = "xcrun stapler staple " + file
		nc.Validate = "xcrun stapler validate " + file
		nc.Assess = "spctl -a -vv -t open --context context:primary-signature " + file
	}
	return nc
}

// NotaryMarkdown renders NOTARIZATION.md (zh/en per --lang).
func NotaryMarkdown(r *NotaryRecord, dir string) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	w("# %s", i18n.S("Notarization", "公证记录"))
	w("")
	w("%s", i18n.F("Written by %s for %s %s. Updated %s. Machine-readable copy: `%s`.", "由 %s 为 %s %s 生成，更新于 %s。机器可读版本：`%s`。",
		r.Tool, orDefault(r.App, "app"), r.Version, r.Updated.Local().Format("2006-01-02 15:04:05 MST"), NotaryJSONFile))
	w("")
	w("%s", i18n.S("Apple keeps processing a submission even if fpack stops waiting (Ctrl-C or `macos.notarize.wait: false`). Run the commands below from this directory, or let fpack do it:",
		"即使 fpack 停止等待（Ctrl-C 或 `macos.notarize.wait: false`），Apple 也会继续处理提交。可以在本目录执行下面的命令，或交给 fpack："))
	w("")
	w("```sh")
	w("fpack notarize status   # %s", i18n.S("ask Apple for the status of every submission below", "向 Apple 查询下面所有提交的状态"))
	w("fpack notarize finish   # %s", i18n.S("wait, staple accepted files, update SHA256SUMS and this file", "等待结果，装订已通过的文件，更新 SHA256SUMS 和本文件"))
	w("```")
	w("")
	w("%s", i18n.S("(Run them in the project directory; add `-C <project>` elsewhere, or pass this directory: `fpack notarize finish <dir>`.)", "（在项目目录中执行；在其他位置请加 `-C <项目>`，或直接传入本目录：`fpack notarize finish <目录>`。）"))
	for _, s := range r.Submissions {
		nc := s.Commands(dir)
		w("")
		w("## %s", filepath.Base(s.Artifact))
		w("")
		status := s.Status
		if s.Stapled {
			status += i18n.S(", ticket stapled", "，已装订票据")
		}
		w("| | |")
		w("| --- | --- |")
		w("| %s | `%s` |", i18n.S("Status", "状态"), status)
		if s.Message != "" {
			w("| %s | %s |", i18n.S("Message", "信息"), strings.ReplaceAll(s.Message, "|", "\\|"))
		}
		w("| %s | `%s` |", i18n.S("Submission ID", "提交 ID"), s.ID)
		w("| %s | %s |", i18n.S("Submitted at", "提交时间"), s.SubmittedAt.Local().Format("2006-01-02 15:04:05 MST"))
		if s.WaitSeconds > 0 {
			w("| %s | %s |", i18n.S("Apple took", "Apple 用时"), (time.Duration(s.WaitSeconds) * time.Second).String())
		}
		w("| %s | `%s` (%s) |", i18n.S("Artifact", "产物"), filepath.Base(s.Artifact), s.Target)
		w("| %s | `%s` |", i18n.S("SHA-256 (uploaded)", "SHA-256（上传时）"), s.SHA256)
		if s.FinalSHA256 != "" && s.FinalSHA256 != s.SHA256 {
			w("| %s | `%s` |", i18n.S("SHA-256 (stapled)", "SHA-256（装订后）"), s.FinalSHA256)
		}
		w("| %s | %s |", i18n.S("Credentials", "凭证"), s.Auth)
		if s.Log != "" {
			w("| %s | `%s` |", i18n.S("Apple's log", "Apple 日志"), filepath.Base(s.Log))
		}
		if len(s.Issues) > 0 {
			w("")
			w("%s", i18n.S("Issues reported by Apple:", "Apple 报告的问题："))
			w("")
			for _, is := range s.Issues {
				w("- %s", is)
			}
		}
		w("")
		w("```sh")
		w("# %s", i18n.S("status", "查询状态"))
		w("%s", nc.Status)
		w("# %s", i18n.S("wait for the result", "等待结果"))
		w("%s", nc.Wait)
		w("# %s", i18n.S("Apple's log (reasons for Invalid)", "Apple 日志（Invalid 的原因）"))
		w("%s", nc.Log)
		w("# %s", i18n.S("after Accepted: staple the ticket", "通过（Accepted）后：装订票据"))
		w("%s", nc.Staple)
		w("# %s", i18n.S("check", "检查"))
		w("%s", nc.Validate)
		w("%s", nc.Assess)
		w("```")
		if strings.Contains(strings.Join(s.AuthArgs, " "), "$FPACK_NOTARY_PASSWORD") {
			w("")
			w("%s", i18n.S("Set FPACK_NOTARY_PASSWORD (the app-specific password) in your shell first.", "请先在 shell 中设置 FPACK_NOTARY_PASSWORD（App 专用密码）。"))
		}
	}
	return b.String()
}

// ------------------------------------------------------------- parsing --

type notaryJSON struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// parseNotaryJSON extracts the last JSON object with an id from notarytool
// output (submit prints {"id","message","path"}, wait/info add "status").
func parseNotaryJSON(out string) (notaryJSON, bool) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if strings.HasPrefix(l, "{") {
			var r notaryJSON
			if json.Unmarshal([]byte(l), &r) == nil && r.ID != "" {
				return r, true
			}
		}
	}
	var r notaryJSON
	if json.Unmarshal([]byte(strings.TrimSpace(out)), &r) == nil && r.ID != "" {
		return r, true
	}
	return notaryJSON{}, false
}

// NotaryLogSummary turns Apple's log JSON into short lines and fix hints.
func NotaryLogSummary(data []byte) (issues, fixes []string) {
	var lg struct {
		Status        string `json:"status"`
		StatusSummary string `json:"statusSummary"`
		Issues        []struct {
			Severity     string `json:"severity"`
			Path         string `json:"path"`
			Message      string `json:"message"`
			Architecture string `json:"architecture"`
		} `json:"issues"`
	}
	if json.Unmarshal(data, &lg) != nil {
		return nil, nil
	}
	seen := map[string]bool{}
	if lg.StatusSummary != "" {
		issues = append(issues, lg.StatusSummary)
	}
	for _, is := range lg.Issues {
		where := is.Path
		if parts := strings.Split(where, "/"); len(parts) > 3 {
			start := len(parts) - 3
			for i, p := range parts {
				if strings.HasSuffix(p, ".app") {
					start = i
					break
				}
			}
			if start > 0 {
				where = ".../" + strings.Join(parts[start:], "/")
			}
		}
		line := fmt.Sprintf("%s: %s", orDefault(is.Severity, "error"), is.Message)
		if where != "" {
			line += " (" + where + ")"
		}
		if !seen[line] && len(issues) < 12 {
			seen[line] = true
			issues = append(issues, line)
		}
		m := strings.ToLower(is.Message)
		add := func(f string) {
			if !seen["fix:"+f] {
				seen["fix:"+f] = true
				fixes = append(fixes, f)
			}
		}
		switch {
		case strings.Contains(m, "developer id"):
			add(i18n.S("sign with a \"Developer ID Application\" certificate (macos.sign.identity), not Apple Development/Distribution", "使用 “Developer ID Application” 证书签名（macos.sign.identity），不能用 Apple Development/Distribution"))
		case strings.Contains(m, "hardened runtime"):
			add(i18n.S("enable the hardened runtime (macos.sign.hardened_runtime: true)", "启用 hardened runtime（macos.sign.hardened_runtime: true）"))
		case strings.Contains(m, "timestamp"):
			add(i18n.S("the signature needs a secure timestamp: check network access to timestamp.apple.com while signing", "签名需要安全时间戳：签名时请确认能访问 timestamp.apple.com"))
		case strings.Contains(m, "get-task-allow"):
			add(i18n.S("remove com.apple.security.get-task-allow: sign with Release.entitlements (macos.sign.entitlements)", "去掉 com.apple.security.get-task-allow：使用 Release.entitlements 签名（macos.sign.entitlements）"))
		case strings.Contains(m, "not signed") || strings.Contains(m, "signature") && strings.Contains(m, "invalid"):
			add(i18n.S("a nested binary is unsigned or broken: rebuild and let fpack re-sign (macos.sign.enabled: true)", "有内嵌二进制未签名或签名损坏：重新构建并让 fpack 重新签名（macos.sign.enabled: true）"))
		}
	}
	return issues, fixes
}

// ---------------------------------------------------------- the actions --

// NotaryAuthFor returns credentials for a recorded submission: the current
// configuration if it names credentials, else the recorded arguments with
// $FPACK_NOTARY_PASSWORD filled from the environment.
func (c *Context) NotaryAuthFor(s NotarySubmission, getenv func(string) string) (args, secrets []string, err error) {
	sg := c.Config.MacOS.Sign
	if sg.NotaryProfile != "" || sg.NotaryAPIKey != "" || sg.NotaryAppleID != "" {
		a, sec, _ := c.Mac.NotaryAuth()
		return a, sec, nil
	}
	if len(s.AuthArgs) == 0 {
		return nil, nil, fmt.Errorf("%s", i18n.S("no notary credentials recorded or configured (macos.sign.notary_*)", "没有记录或配置公证凭证（macos.sign.notary_*）"))
	}
	for _, a := range s.AuthArgs {
		if a == "$FPACK_NOTARY_PASSWORD" {
			pw := getenv("FPACK_NOTARY_PASSWORD")
			if pw == "" {
				return nil, nil, fmt.Errorf("%s", i18n.S("set FPACK_NOTARY_PASSWORD (app-specific password) to query this Apple ID submission", "请设置 FPACK_NOTARY_PASSWORD（App 专用密码）以查询此 Apple ID 提交"))
			}
			args = append(args, pw)
			secrets = append(secrets, pw)
			continue
		}
		args = append(args, a)
	}
	return args, secrets, nil
}

func notaryCmd(sub string, id string, auth, secrets []string, extra ...string) runner.Cmd {
	a := append([]string{"notarytool", sub}, id)
	a = append(append(a, auth...), extra...)
	return runner.Cmd{Name: "xcrun", Args: a, Capture: true, Secret: secrets}
}

// NotaryWaitHint is shown while waiting.
func NotaryWaitHint(md string) string {
	return i18n.F("Notarization usually takes several minutes. You can press Ctrl-C to stop waiting — the submission continues on Apple's side; check it later with the commands in %s (or: fpack notarize status / finish).",
		"公证通常需要几分钟。可以按 Ctrl-C 停止等待 —— Apple 端会继续处理；之后可用 %s 中的命令查询（或：fpack notarize status / finish）。", md)
}

// NotaryWait waits for Apple's verdict on s, printing elapsed time, and
// updates s (Status, Message, WaitSeconds, Log, Issues). On Ctrl-C it
// returns runner.ErrInterrupted with s unchanged.
func NotaryWait(ctx context.Context, env OpEnv, s *NotarySubmission, auth, secrets []string, logDir string) error {
	start := time.Now()
	waitText := i18n.S("waiting for Apple · usually several minutes · Ctrl-C stops waiting", "等待 Apple 公证 · 通常需要几分钟 · Ctrl-C 可停止等待")
	env.Status(waitText)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(heartbeatEvery)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				el := time.Since(start).Truncate(time.Second)
				if env.Interactive {
					env.Status(fmt.Sprintf("%s · %s", waitText, el))
				} else {
					env.Print("detail", i18n.F("… still waiting for Apple (%s, id %s)", "… 仍在等待 Apple（%s，id %s）", el, s.ID))
				}
			}
		}
	}()
	res, err := env.Run(notaryCmd("wait", s.ID, auth, secrets, "--output-format", "json"))
	close(stop)
	<-done
	if errors.Is(err, runner.ErrInterrupted) || ctx.Err() != nil {
		return runner.ErrInterrupted
	}
	r, ok := parseNotaryJSON(res.Output)
	if !ok || r.Status == "" {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", i18n.S("notarytool wait returned no status", "notarytool wait 没有返回状态"))
	}
	s.Status, s.Message = r.Status, r.Message
	s.WaitSeconds = int(time.Since(start).Seconds())
	if s.Status == NotaryInvalid || s.Status == NotaryRejected {
		NotaryFetchLog(env, s, auth, secrets, logDir)
	}
	return nil
}

// NotaryFetchLog downloads Apple's log next to the artifact and summarizes it.
func NotaryFetchLog(env OpEnv, s *NotarySubmission, auth, secrets []string, dir string) {
	path := filepath.Join(dir, "notary-log-"+filepath.Base(s.Artifact)+".json")
	if _, err := env.Run(notaryCmd("log", s.ID, auth, secrets, path)); err != nil {
		return
	}
	if data, err := os.ReadFile(path); err == nil {
		s.Log = path
		issues, fixes := NotaryLogSummary(data)
		s.Issues = append(issues, fixes...)
	}
}

// NotaryInfo asks Apple for the current status.
func NotaryInfo(env OpEnv, s *NotarySubmission, auth, secrets []string) error {
	res, err := env.Run(notaryCmd("info", s.ID, auth, secrets, "--output-format", "json"))
	r, ok := parseNotaryJSON(res.Output)
	if !ok || r.Status == "" {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", i18n.S("notarytool info returned no status", "notarytool info 没有返回状态"))
	}
	s.Status, s.Message = r.Status, r.Message
	return nil
}

// NotaryStaple staples the ticket to an accepted artifact (for the app
// zip: unzip, staple the .app, zip again) and validates it.
func NotaryStaple(env OpEnv, s *NotarySubmission, workDir string) error {
	file := s.Artifact
	if s.Target == "macos" {
		tmp := filepath.Join(workDir, "notarize-"+s.ID)
		_ = os.RemoveAll(tmp)
		defer os.RemoveAll(tmp)
		if _, err := env.Run(runner.Cmd{Name: "ditto", Args: []string{"-x", "-k", file, tmp}}); err != nil {
			return err
		}
		app := filepath.Join(tmp, s.App)
		if s.App == "" || !exists(app) {
			if m, _ := filepath.Glob(filepath.Join(tmp, "*.app")); len(m) > 0 {
				app = m[0]
			}
		}
		if _, err := env.Run(runner.Cmd{Name: "xcrun", Args: []string{"stapler", "staple", app}}); err != nil {
			return err
		}
		if _, err := env.Run(runner.Cmd{Name: "xcrun", Args: []string{"stapler", "validate", app}}); err != nil {
			return err
		}
		zip := file + ".partial"
		if _, err := env.Run(runner.Cmd{Name: "ditto", Args: []string{"-c", "-k", "--sequesterRsrc", "--keepParent", app, zip}}); err != nil {
			return err
		}
		if err := os.Rename(zip, file); err != nil {
			return err
		}
	} else {
		if _, err := env.Run(runner.Cmd{Name: "xcrun", Args: []string{"stapler", "staple", file}}); err != nil {
			return err
		}
		if _, err := env.Run(runner.Cmd{Name: "xcrun", Args: []string{"stapler", "validate", file}}); err != nil {
			return err
		}
	}
	s.Stapled = true
	if h, err := pack.SHA256File(file); err == nil {
		s.FinalSHA256 = h
	}
	return nil
}

// submitRetryDelays are the pauses before retrying an upload that failed
// because of the network (tests shorten them). Apple's upload goes to S3 and
// large files occasionally time out; nothing is submitted in that case.
var submitRetryDelays = []time.Duration{20 * time.Second, 60 * time.Second}

// notaryTransient matches notarytool output for network/upload failures that
// are worth retrying (no submission was created).
var notaryTransient = regexp.MustCompile(`(?i)(abortedUpload|deadlineExceeded|HTTPClientError|NSURLErrorDomain|network connection was lost|request timed out|timed out|Could not connect to the server|connection reset|remoteConnectionClosed|HTTP status code: 5\d\d|Service Unavailable)`)

// NotaryTransient reports whether notarytool failed because of the network.
func NotaryTransient(output string) bool { return notaryTransient.MatchString(output) }

// submitNotary runs `notarytool submit`, retrying network/upload failures.
func submitNotary(ctx context.Context, env OpEnv, file string, auth, secrets []string) (runner.Result, error) {
	cmd := runner.Cmd{Name: "xcrun", Args: append(append([]string{"notarytool", "submit", file}, auth...), "--output-format", "json"), Capture: true, Secret: secrets}
	attempts := len(submitRetryDelays) + 1
	for i := 0; ; i++ {
		env.Status(i18n.S("uploading to Apple…", "正在上传到 Apple…"))
		res, err := env.Run(cmd)
		if err == nil || errors.Is(err, runner.ErrInterrupted) || ctx.Err() != nil {
			return res, err
		}
		text := res.Output + "\n" + strings.Join(res.Tail, "\n") + "\n" + err.Error()
		if !NotaryTransient(text) {
			return res, err
		}
		if i == len(submitRetryDelays) {
			return res, fmt.Errorf("%w – %s", err, i18n.F("upload to Apple failed %d times (network timeout); nothing was submitted", "上传到 Apple 失败 %d 次（网络超时），没有创建任何提交", attempts))
		}
		d := submitRetryDelays[i]
		env.Print("warn", i18n.F("upload to Apple failed (network timeout); nothing was submitted – retrying in %s (attempt %d/%d)", "上传到 Apple 失败（网络超时），没有创建提交 —— %s 后重试（第 %d/%d 次）", d, i+2, attempts))
		select {
		case <-ctx.Done():
			return res, runner.ErrInterrupted
		case <-time.After(d):
		}
	}
}

// heartbeatEvery is how often waiting progress is reported (tests shorten it).
var heartbeatEvery = 30 * time.Second

// notarizeOp uploads file (staged; it becomes artifact), records the
// submission and, unless macos.notarize.wait is false, waits for Apple.
// Stapling follows as separate ops when waiting.
func notarizeOp(c *Context, target, file, artifact, app string) Op {
	auth, secrets, _ := c.Mac.NotaryAuth()
	wait := c.Config.NotarizeWait()
	submit := runner.Cmd{Name: "xcrun", Args: append(append([]string{"notarytool", "submit", file}, auth...), "--output-format", "json"), Capture: true, Secret: secrets}
	waitCmd := notaryCmd("wait", "<submission-id>", auth, secrets, "--output-format", "json")
	preview := []*runner.Cmd{&submit}
	desc := i18n.F("notarize with %s (upload, record in %s, wait for Apple)", "使用 %s 公证（上传，记录到 %s，等待 Apple）", c.Mac.NotaryLabel(), NotaryMDFile)
	if wait {
		preview = append(preview, &waitCmd)
	} else {
		desc = i18n.F("submit for notarization with %s (record in %s, don't wait)", "使用 %s 提交公证（记录到 %s，不等待）", c.Mac.NotaryLabel(), NotaryMDFile)
	}
	hint := i18n.S("check the notarization credentials (macos.sign.notary_*) and the network; details are in the log", "请检查公证凭证（macos.sign.notary_*）和网络；详情见日志")
	if c.Mac.Profile != "" || (c.Mac.APIKey == "" && c.Mac.AppleID == "") {
		hint = i18n.F("check the profile with `xcrun notarytool history --keychain-profile %s` (create it once with: xcrun notarytool store-credentials %s --apple-id <apple-id> --team-id <team-id>) and the network; details are in the log",
			"用 `xcrun notarytool history --keychain-profile %s` 检查凭证（首次创建：xcrun notarytool store-credentials %s --apple-id <Apple ID> --team-id <团队ID>），并检查网络；详情见日志", auth[1], auth[1])
	}
	return Op{Desc: desc, Preview: preview, Hint: hint, Run: func(ctx context.Context, env OpEnv) (string, error) {
		return runNotarize(ctx, c, env, target, file, artifact, app, auth, secrets, wait)
	}}
}

func runNotarize(ctx context.Context, c *Context, env OpEnv, target, file, artifact, app string, auth, secrets []string, wait bool) (string, error) {
	sha, err := pack.SHA256File(file)
	if err != nil {
		return "", err
	}
	res, err := submitNotary(ctx, env, file, auth, secrets)
	if err != nil {
		return "", err
	}
	r, ok := parseNotaryJSON(res.Output)
	if !ok {
		return "", fmt.Errorf("%s", i18n.S("notarytool submit returned no submission id", "notarytool submit 没有返回提交 ID"))
	}
	s := NotarySubmission{Target: target, Artifact: artifact, Uploaded: file, App: app, SHA256: sha, ID: r.ID,
		SubmittedAt: time.Now(), Auth: c.Mac.NotaryLabel(), AuthArgs: c.Mac.NotaryAuthDisplay(), Status: NotarySubmitted, Message: r.Message}
	md, err := recordNotary(c, s)
	if err != nil {
		env.Print("warn", i18n.F("could not write %s: %v", "无法写入 %s：%v", NotaryMDFile, err))
	}
	env.Print("ok", i18n.F("submitted to Apple · id %s", "已提交到 Apple · id %s", r.ID))
	env.Print("detail", i18n.F("status and commands: %s", "状态与命令：%s", c.Rel(md)))
	if !wait {
		// Keep the uploaded (not yet stapled) file as the artifact.
		return i18n.F("notarization submitted (id %s), not waiting: check with `fpack notarize status`, staple with `fpack notarize finish` (see %s)", "已提交公证（id %s），未等待：用 `fpack notarize status` 查询，用 `fpack notarize finish` 装订（见 %s）", r.ID, c.Rel(md)), nil
	}
	env.Print("hint", NotaryWaitHint(c.Rel(md)))
	if err := NotaryWait(ctx, env, &s, auth, secrets, c.OutDir); err != nil {
		if errors.Is(err, runner.ErrInterrupted) {
			// Keep the uploaded file so `fpack notarize finish` can staple it.
			if file != artifact && exists(file) && !exists(artifact) {
				if pack.MoveFile(file, artifact) == nil {
					s.Uploaded = artifact
				}
			}
			s.Message = i18n.S("fpack stopped waiting (Ctrl-C); Apple keeps processing", "fpack 已停止等待（Ctrl-C）；Apple 会继续处理")
			_, _ = recordNotary(c, s)
			env.Print("warn", i18n.F("stopped waiting: submission %s continues on Apple's side. Check it later with the commands in %s, or run: fpack notarize finish", "已停止等待：提交 %s 会在 Apple 端继续处理。之后可用 %s 中的命令查询，或运行：fpack notarize finish", s.ID, c.Rel(md)))
		}
		return "", err
	}
	_, _ = recordNotary(c, s)
	el := (time.Duration(s.WaitSeconds) * time.Second).String()
	if s.Status == NotaryAccepted {
		return i18n.F("notarized ✓ (id %s, Apple took %s)", "已公证 ✓（id %s，Apple 用时 %s）", s.ID, el), nil
	}
	msg := i18n.F("notarization %s: %s (id %s)", "公证结果 %s：%s（id %s）", s.Status, s.Message, s.ID)
	for _, is := range s.Issues {
		msg += "\n  " + is
	}
	if s.Log != "" {
		msg += "\n  " + i18n.F("Apple's log: %s · details: %s", "Apple 日志：%s · 详情：%s", c.Rel(s.Log), c.Rel(md))
	}
	return "", fmt.Errorf("%s", msg)
}

// notaryDoneOp updates the record once the artifact is final (moved into the
// output directory, stapled when waited for).
func notaryDoneOp(c *Context, artifact string, stapled bool) Op {
	return Op{Desc: i18n.F("update %s", "更新 %s", NotaryMDFile), Fn: func() error {
		s, ok := c.NotaryFor(artifact)
		if !ok {
			return nil
		}
		s.Uploaded = artifact
		if stapled && s.Status == NotaryAccepted {
			s.Stapled = true
			if h, err := pack.SHA256File(artifact); err == nil {
				s.FinalSHA256 = h
			}
		}
		_, err := recordNotary(c, s)
		return err
	}}
}
