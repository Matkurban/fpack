package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Matkurban/fpack/go/internal/targets"
)

// fakeXcrun puts scripts/fake-apple/xcrun first on PATH.
func fakeXcrun(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell shim")
	}
	src, err := filepath.Abs("../../../scripts/fake-apple/xcrun")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(bin, "xcrun"), data, 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	logf := filepath.Join(bin, "calls.log")
	t.Setenv("FAKE_NOTARY_LOG", logf)
	t.Setenv("FAKE_NOTARY_DELAY", "0")
	return logf
}

func pendingDMG(t *testing.T, proj string) (dir, dmg string) {
	t.Helper()
	dir = filepath.Join(proj, "dist", "2.3.4+5")
	dmg = filepath.Join(dir, "my_app-2.3.4+5-macos-universal.dmg")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(dmg, []byte("dmg"), 0o644)
	r := &targets.NotaryRecord{App: "my_app", Version: "2.3.4+5", Submissions: []targets.NotarySubmission{{
		Target: "dmg", Artifact: dmg, Uploaded: dmg, ID: "2efe2717-52ef-43a5-96dc-0797e4ca1041", SubmittedAt: time.Now().Add(-3 * time.Minute),
		Auth: "keychain profile XueHua", AuthArgs: []string{"--keychain-profile", "XueHua"}, Status: targets.NotarySubmitted, SHA256: "x",
	}}}
	if _, err := targets.SaveNotary(dir, r); err != nil {
		t.Fatal(err)
	}
	return dir, dmg
}

func TestNotarizeStatusAndFinish(t *testing.T) {
	calls := fakeXcrun(t)
	proj, sdk := fixture(t, "")
	dir, dmg := pendingDMG(t, proj)

	t.Setenv("FAKE_NOTARY_INFO_STATUS", "In Progress")
	r := run(t, nil, "notarize", "status", "-C", proj, "--flutter", sdk)
	if r.code != 0 || !strings.Contains(r.all(), "In Progress") || !strings.Contains(r.all(), "NOTARIZATION.md") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	log, _ := os.ReadFile(calls)
	if !strings.Contains(string(log), "notarytool info 2efe2717-52ef-43a5-96dc-0797e4ca1041 --keychain-profile XueHua") {
		t.Fatal(string(log))
	}

	t.Setenv("FAKE_NOTARY_INFO_STATUS", "")
	r = run(t, nil, "notarize", "finish", dir, "-C", proj, "--flutter", sdk)
	if r.code != 0 || !strings.Contains(r.all(), "stapled") || !strings.Contains(r.all(), "Ctrl-C") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	log, _ = os.ReadFile(calls)
	for _, w := range []string{"notarytool wait 2efe2717", "stapler staple " + dmg, "stapler validate " + dmg} {
		if !strings.Contains(string(log), w) {
			t.Errorf("missing %q in\n%s", w, log)
		}
	}
	rec, _ := targets.LoadNotary(dir)
	if s := rec.Submissions[0]; !s.Stapled || s.Status != targets.NotaryAccepted || s.FinalSHA256 == "" {
		t.Fatalf("%+v", s)
	}
	if sums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS")); err != nil || !strings.Contains(string(sums), filepath.Base(dmg)) || strings.Contains(string(sums), "NOTARIZATION") {
		t.Fatal(string(sums), err)
	}
	// nothing left to do
	r = run(t, nil, "notarize", "finish", "-C", proj, "--flutter", sdk)
	if r.code != 0 || !strings.Contains(r.all(), "stapled") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
}

func TestNotarizeFinishInvalidSummarizesLog(t *testing.T) {
	fakeXcrun(t)
	t.Setenv("FAKE_NOTARY_STATUS", "Invalid")
	proj, sdk := fixture(t, "")
	dir, _ := pendingDMG(t, proj)
	r := run(t, nil, "notarize", "finish", "-C", proj, "--flutter", sdk)
	if r.code != 1 || !strings.Contains(r.all(), "hardened runtime") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	md, _ := os.ReadFile(filepath.Join(dir, targets.NotaryMDFile))
	if !strings.Contains(string(md), "`Invalid`") || !strings.Contains(string(md), "notary-log-") {
		t.Fatal(string(md))
	}
}

func TestNotarizeUsage(t *testing.T) {
	proj, sdk := fixture(t, "")
	if r := run(t, nil, "notarize", "-C", proj, "--flutter", sdk); r.code != 2 {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	if r := run(t, nil, "notarize", "status", "-C", proj, "--flutter", sdk); r.code != 3 || !strings.Contains(r.all(), "notarization.json") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	if r := run(t, nil, "help", "notarize"); r.code != 0 || !strings.Contains(r.all(), "fpack notarize finish") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
}
