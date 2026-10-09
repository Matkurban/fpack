package targets

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Matkurban/fpack/go/internal/runner"
)

// fakeNotary answers notarytool like Apple would.
type fakeNotary struct {
	status  string // wait/info result
	calls   []string
	printed []string
	waitErr error
	// submitFail makes the first n submits fail with submitOut.
	submitFail int
	submitOut  string
}

func (f *fakeNotary) env() OpEnv {
	return OpEnv{
		Run: func(cm runner.Cmd) (runner.Result, error) {
			f.calls = append(f.calls, cm.String())
			if cm.Name != "xcrun" || cm.Args[0] != "notarytool" {
				return runner.Result{}, nil
			}
			switch cm.Args[1] {
			case "submit":
				if f.submitFail > 0 {
					f.submitFail--
					return runner.Result{Tail: []string{f.submitOut}}, errors.New("xcrun exited with code 1")
				}
				return runner.Result{Output: "Conducting pre-submission checks...\n{\"id\":\"2efe2717-52ef-43a5-96dc-0797e4ca1041\",\"message\":\"Successfully uploaded file\",\"path\":\"x\"}\n"}, nil
			case "wait", "info":
				if f.waitErr != nil {
					return runner.Result{}, f.waitErr
				}
				return runner.Result{Output: `{"id":"2efe2717-52ef-43a5-96dc-0797e4ca1041","status":"` + f.status + `","message":"Processing complete"}`}, nil
			case "log":
				path := cm.Args[len(cm.Args)-1]
				os.WriteFile(path, []byte(`{"status":"Invalid","statusSummary":"Archive contains critical validation errors","issues":[{"severity":"error","path":"a.dmg/XueHua.app/Contents/MacOS/XueHua","message":"The executable does not have the hardened runtime enabled.","architecture":"arm64"},{"severity":"error","path":"a.dmg/XueHua.app/Contents/MacOS/XueHua","message":"The executable does not have the hardened runtime enabled.","architecture":"x86_64"}]}`), 0o644)
				return runner.Result{}, nil
			}
			return runner.Result{}, nil
		},
		Print:  func(kind, msg string) { f.printed = append(f.printed, kind+": "+msg) },
		Status: func(string) {},
	}
}

func notaryCtx(t *testing.T, cfg string) (*Context, string, string) {
	t.Helper()
	c := newCtx(t, "darwin", "macos:\n  sign:\n    identity: \""+devID+"\"\n    notary_profile: XueHua\n"+cfg, "")
	c.DryRun = false
	file := filepath.Join(c.WorkDir, "stage", "dmg", "app.dmg")
	os.MkdirAll(filepath.Dir(file), 0o755)
	os.WriteFile(file, []byte("dmg bytes"), 0o644)
	return c, file, filepath.Join(c.OutDir, "app.dmg")
}

func readRecord(t *testing.T, dir string) (*NotaryRecord, string) {
	t.Helper()
	r, err := LoadNotary(dir)
	if err != nil || len(r.Submissions) != 1 {
		t.Fatalf("record: %v %+v", err, r)
	}
	md, err := os.ReadFile(filepath.Join(dir, NotaryMDFile))
	if err != nil {
		t.Fatal(err)
	}
	return r, string(md)
}

func TestNotarizeWaitAccepted(t *testing.T) {
	c, file, dst := notaryCtx(t, "")
	f := &fakeNotary{status: "Accepted"}
	note, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, true)
	if err != nil || !strings.Contains(note, "notarized") {
		t.Fatal(note, err)
	}
	if len(f.calls) != 2 || !strings.Contains(f.calls[1], "notarytool wait 2efe2717-52ef-43a5-96dc-0797e4ca1041 --keychain-profile XueHua") {
		t.Fatal(f.calls)
	}
	all := strings.Join(f.printed, "\n")
	for _, w := range []string{"2efe2717", "NOTARIZATION.md", "Ctrl-C", "several minutes"} {
		if !strings.Contains(all, w) {
			t.Errorf("printed output lacks %q:\n%s", w, all)
		}
	}
	r, md := readRecord(t, c.OutDir)
	s := r.Submissions[0]
	if s.Status != NotaryAccepted || s.SHA256 == "" || s.ID == "" || s.Auth != "keychain profile XueHua" || s.Artifact != dst || s.Target != "dmg" {
		t.Fatalf("%+v", s)
	}
	for _, w := range []string{
		"xcrun notarytool info 2efe2717-52ef-43a5-96dc-0797e4ca1041 --keychain-profile XueHua",
		"xcrun notarytool wait 2efe2717-52ef-43a5-96dc-0797e4ca1041 --keychain-profile XueHua",
		"xcrun notarytool log 2efe2717-52ef-43a5-96dc-0797e4ca1041 --keychain-profile XueHua notary-log-app.dmg.json",
		"xcrun stapler staple app.dmg", "xcrun stapler validate app.dmg", "spctl -a -vv -t open",
		"fpack notarize status", "fpack notarize finish", "`Accepted`",
	} {
		if !strings.Contains(md, w) {
			t.Errorf("NOTARIZATION.md lacks %q:\n%s", w, md)
		}
	}
	// after the move the done op marks it stapled
	os.MkdirAll(c.OutDir, 0o755)
	os.Rename(file, dst)
	if err := notaryDoneOp(c, dst, true).Fn(); err != nil {
		t.Fatal(err)
	}
	r, _ = readRecord(t, c.OutDir)
	if !r.Submissions[0].Stapled || r.Submissions[0].State() != "stapled" || r.Submissions[0].FinalSHA256 == "" {
		t.Fatalf("%+v", r.Submissions[0])
	}
}

func TestNotarizeInvalidFetchesLog(t *testing.T) {
	c, file, dst := notaryCtx(t, "")
	f := &fakeNotary{status: "Invalid"}
	_, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, true)
	if err == nil || !strings.Contains(err.Error(), "hardened runtime") || !strings.Contains(err.Error(), "notary-log-app.dmg.json") {
		t.Fatal(err)
	}
	r, md := readRecord(t, c.OutDir)
	s := r.Submissions[0]
	if s.Status != NotaryInvalid || s.Log == "" || len(s.Issues) != 3 || s.State() != "invalid" {
		t.Fatalf("%+v", s) // summary + one deduplicated issue + one fix
	}
	if !strings.Contains(md, "Archive contains critical validation errors") || !strings.Contains(md, "hardened_runtime: true") {
		t.Fatal(md)
	}
}

func TestNotarizeNoWait(t *testing.T) {
	c, file, dst := notaryCtx(t, "  notarize:\n    wait: false\n")
	if c.Config.NotarizeWait() {
		t.Fatal("wait should be off")
	}
	p := plan(t, &DMG{}, c)
	cs := strings.Join(cmds(p), "\n")
	if !strings.Contains(cs, "notarytool submit") || strings.Contains(cs, "notarytool wait") || strings.Contains(cs, "stapler") {
		t.Fatal(cs)
	}
	if p.Artifacts[0].Kind != "DMG (signed, notarization submitted)" {
		t.Fatal(p.Artifacts[0].Kind)
	}
	f := &fakeNotary{status: "Accepted"}
	note, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, false)
	if err != nil || !strings.Contains(note, "fpack notarize finish") || len(f.calls) != 1 {
		t.Fatal(note, err, f.calls)
	}
	r, _ := readRecord(t, c.OutDir)
	if s := r.Submissions[0]; s.Status != NotarySubmitted || s.State() != "submitted" || s.Stapled {
		t.Fatalf("%+v", s)
	}
}

func TestNotarizeCtrlCKeepsFileAndPointsToRecord(t *testing.T) {
	c, file, dst := notaryCtx(t, "")
	f := &fakeNotary{waitErr: runner.ErrInterrupted}
	_, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, true)
	if err != runner.ErrInterrupted {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal("uploaded file should be kept as the artifact for `fpack notarize finish`")
	}
	if all := strings.Join(f.printed, "\n"); !strings.Contains(all, "continues on Apple's side") || !strings.Contains(all, "NOTARIZATION.md") {
		t.Fatal(all)
	}
	r, _ := readRecord(t, c.OutDir)
	if s := r.Submissions[0]; s.Status != NotarySubmitted || s.Uploaded != dst {
		t.Fatalf("%+v", s)
	}
}

func TestNotaryAppleIDNeverWritesPassword(t *testing.T) {
	c := newCtx(t, "darwin", "macos:\n  sign:\n    identity: \""+devID+"\"\n    notary_apple_id: dev@example.com\n    notary_team_id: ABCDE12345\n    notary_password: s3cret-pass\n", "")
	c.DryRun = false
	file := filepath.Join(c.WorkDir, "app.pkg")
	os.MkdirAll(c.WorkDir, 0o755)
	os.WriteFile(file, []byte("pkg"), 0o644)
	auth, secrets, _ := c.Mac.NotaryAuth()
	f := &fakeNotary{status: "Accepted"}
	if _, err := runNotarize(context.Background(), c, f.env(), "pkg", file, filepath.Join(c.OutDir, "app.pkg"), "", auth, secrets, true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(c.OutDir, NotaryJSONFile))
	md, _ := os.ReadFile(filepath.Join(c.OutDir, NotaryMDFile))
	if strings.Contains(string(data)+string(md), "s3cret-pass") {
		t.Fatal("password leaked")
	}
	if !strings.Contains(string(md), `--password "$FPACK_NOTARY_PASSWORD"`) || !strings.Contains(string(md), "spctl -a -vv -t install app.pkg") {
		t.Fatal(string(md))
	}
	// later queries take the password from the environment
	var r NotaryRecord
	json.Unmarshal(data, &r)
	c2 := newCtx(t, "darwin", "", "")
	if _, _, err := c2.NotaryAuthFor(r.Submissions[0], func(string) string { return "" }); err == nil {
		t.Fatal("expected missing password error")
	}
	args, sec, err := c2.NotaryAuthFor(r.Submissions[0], func(k string) string { return map[string]string{"FPACK_NOTARY_PASSWORD": "pw2"}[k] })
	if err != nil || strings.Join(args, " ") != "--apple-id dev@example.com --team-id ABCDE12345 --password pw2" || sec[0] != "pw2" {
		t.Fatal(args, sec, err)
	}
}

func TestNotaryHeartbeatWhenNotInteractive(t *testing.T) {
	old := heartbeatEvery
	heartbeatEvery = 20 * time.Millisecond
	defer func() { heartbeatEvery = old }()
	f := &fakeNotary{status: "Accepted"}
	env := f.env()
	inner := env.Run
	env.Run = func(cm runner.Cmd) (runner.Result, error) {
		time.Sleep(90 * time.Millisecond)
		return inner(cm)
	}
	s := &NotarySubmission{ID: "id-1", Artifact: "/tmp/x.dmg"}
	if err := NotaryWait(context.Background(), env, s, nil, nil, t.TempDir()); err != nil || s.Status != NotaryAccepted {
		t.Fatal(err, s)
	}
	if !strings.Contains(strings.Join(f.printed, "\n"), "still waiting for Apple") {
		t.Fatal(f.printed)
	}
}

func TestNotaryLogSummary(t *testing.T) {
	issues, fixes := NotaryLogSummary([]byte(`{"statusSummary":"Archive contains critical validation errors","issues":[{"severity":"error","path":"x.zip/A.app/Contents/MacOS/A","message":"The binary is not signed with a valid Developer ID certificate."},{"severity":"error","path":"x.zip/A.app/Contents/MacOS/A","message":"The signature does not include a secure timestamp."}]}`))
	if len(issues) != 3 || len(fixes) != 2 || !strings.Contains(issues[1], ".../A.app/Contents/MacOS/A") {
		t.Fatal(issues, fixes)
	}
}

const s3Timeout = `Error: abortedUpload(resumeRequest: SotoS3.S3.ResumeMultipartUploadRequest(uploadRequest: SotoS3.S3.CreateMultipartUploadRequest(bucket: "notary-submissions-prod"), completedParts: []), error: HTTPClientError.deadlineExceeded)`

func shortRetries(t *testing.T) {
	old := submitRetryDelays
	submitRetryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	t.Cleanup(func() { submitRetryDelays = old })
}

func TestNotarizeRetriesUploadTimeouts(t *testing.T) {
	shortRetries(t)
	c, file, dst := notaryCtx(t, "")
	f := &fakeNotary{status: "Accepted", submitFail: 2, submitOut: s3Timeout}
	note, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, true)
	if err != nil || !strings.Contains(note, "notarized") {
		t.Fatal(note, err)
	}
	submits := 0
	for _, cl := range f.calls {
		if strings.Contains(cl, "notarytool submit") {
			submits++
		}
	}
	if submits != 3 || !strings.Contains(strings.Join(f.printed, "\n"), "retrying") {
		t.Fatalf("submits=%d printed=%v", submits, f.printed)
	}
}

func TestNotarizeGivesUpAfterRetries(t *testing.T) {
	shortRetries(t)
	c, file, dst := notaryCtx(t, "")
	f := &fakeNotary{status: "Accepted", submitFail: 5, submitOut: s3Timeout}
	_, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, true)
	if err == nil || !strings.Contains(err.Error(), "failed 3 times") || f.submitFail != 2 {
		t.Fatalf("err=%v left=%d", err, f.submitFail)
	}
	if _, statErr := os.Stat(filepath.Join(c.OutDir, NotaryJSONFile)); statErr == nil {
		t.Fatal("no submission must be recorded")
	}
}

func TestNotarizeDoesNotRetryAuthErrors(t *testing.T) {
	shortRetries(t)
	c, file, dst := notaryCtx(t, "")
	f := &fakeNotary{submitFail: 5, submitOut: `Error: No Keychain password item found for profile: XueHua`}
	if _, err := runNotarize(context.Background(), c, f.env(), "dmg", file, dst, "", []string{"--keychain-profile", "XueHua"}, nil, true); err == nil || f.submitFail != 4 {
		t.Fatalf("err=%v left=%d", err, f.submitFail)
	}
}
