package targets

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/runner"
)

func signCtx(t *testing.T, cfg string) *Context {
	t.Helper()
	c := newCtx(t, "windows", cfg, "")
	c.Tools.(*fakeTools).bins["signtool"] = `C:\SDK\signtool.exe`
	return c
}

func TestSigntoolArgsPerCertificateSource(t *testing.T) {
	for name, tc := range map[string]struct{ cfg, want, not string }{
		"pfx":        {"windows:\n  sign:\n    certificate: certs/a.pfx\n    password: s3cret\n", "/f ", "/sha1"},
		"thumbprint": {"windows:\n  sign:\n    thumbprint: 1A 2B 3C\n    machine_store: true\n    store: Root\n", "/s Root /sm /sha1 1A2B3C", "/f "},
		"subject":    {"windows:\n  sign:\n    subject: XueHua Tech\n", "/n XueHua Tech", "/sha1"},
		"dlib":       {"windows:\n  sign:\n    dlib: tools/Azure.CodeSigning.Dlib.dll\n    dlib_metadata: meta.json\n", "/dlib ", "/f "},
	} {
		c := signCtx(t, tc.cfg)
		args, secrets := signtoolArgs(c, func(s string) string { return s })
		got := strings.Join(args, " ")
		if !strings.HasPrefix(got, "/fd sha256 /tr "+DefaultTimestampURL+" /td sha256") || !strings.Contains(got, tc.want) || strings.Contains(got, tc.not) {
			t.Errorf("%s: %s", name, got)
		}
		if name == "pfx" && (len(secrets) != 1 || secrets[0] != "s3cret") {
			t.Errorf("password must be a secret: %v", secrets)
		}
	}
	c := signCtx(t, "windows:\n  sign:\n    subject: X\n    url: https://x.example\n    description: My App\n    timestamp_url: http://ts.example\n")
	args, _ := signtoolArgs(c, func(s string) string { return s })
	if got := strings.Join(args, " "); !strings.Contains(got, "/tr http://ts.example") || !strings.Contains(got, "/d My App /du https://x.example") {
		t.Fatal(got)
	}
}

func TestSignedWindowsBuildSignsAndVerifiesBinaries(t *testing.T) {
	c := signCtx(t, "windows:\n  sign:\n    thumbprint: ABC\n")
	steps, _ := (&WinZip{}).Steps(c)
	after := steps[0].After
	if len(after) != 2 || after[0].Run == nil || after[1].Run == nil || !after[1].Optional {
		t.Fatalf("want sign + optional verify ops, got %+v", after)
	}
	if p := after[0].Preview[0].String(); !strings.Contains(p, "sign") || !strings.Contains(p, "*.exe|*.dll") {
		t.Fatal(p)
	}
	c = signCtx(t, "windows:\n  sign:\n    thumbprint: ABC\n    verify: false\n")
	if steps, _ := (&WinZip{}).Steps(c); len(steps[0].After) != 1 {
		t.Fatal("verify: false must skip verification")
	}
}

// The sign op really runs signtool once with every matching file (a fake
// signtool records its arguments).
func TestSignFilesRunsFakeSigntool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake signtool is a shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.txt")
	fake := filepath.Join(dir, "signtool")
	must(t, os.WriteFile(fake, []byte("#!/bin/sh\necho \"$@\" >> '"+log+"'\n"), 0o755))
	rel := filepath.Join(dir, "Release")
	for _, f := range []string{"app.exe", "flutter_windows.dll", "data/app.so", "plugins/x/p.DLL", "readme.txt"} {
		must(t, os.MkdirAll(filepath.Join(rel, filepath.Dir(f)), 0o755))
		must(t, os.WriteFile(filepath.Join(rel, f), []byte("x"), 0o644))
	}
	c := signCtx(t, "windows:\n  sign:\n    certificate: a.pfx\n    password: pw\n    signtool: "+fake+"\n")
	env := OpEnv{Run: func(cmd runner.Cmd) (runner.Result, error) {
		return runner.Run(context.Background(), cmd, runner.Options{})
	}}
	for _, op := range signFilesOps(c, rel) {
		if _, err := op.Run(context.Background(), env); err != nil {
			t.Fatal(op.Desc, err)
		}
	}
	b, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "sign /fd sha256") || !strings.HasPrefix(lines[1], "verify /pa") {
		t.Fatalf("calls: %q", lines)
	}
	for _, want := range []string{"app.exe", "flutter_windows.dll", "p.DLL", "/p pw"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("sign call misses %s: %s", want, lines[0])
		}
	}
	if strings.Contains(lines[0], "app.so") || strings.Contains(lines[0], "readme.txt") {
		t.Error(lines[0])
	}

	// nothing to sign is an error, not a silent unsigned build
	c.Config.Windows.Sign.Files = []string{"*.sys"}
	if _, err := signFilesOps(c, rel)[0].Run(context.Background(), env); err == nil {
		t.Fatal("want error when no file matches")
	}
}

func TestMatchFiles(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.exe", "b.dll", "plugins/c.dll", "data/d.dat"} {
		must(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(f)), 0o755))
		must(t, os.WriteFile(filepath.Join(dir, f), nil, 0o644))
	}
	got, _ := matchFiles(dir, []string{"plugins/*.dll", "*.EXE"})
	if len(got) != 2 || filepath.Base(got[0]) != "a.exe" || filepath.Base(got[1]) != "c.dll" {
		t.Fatal(got)
	}
}

func TestSignedInstallerAndMsix(t *testing.T) {
	c := signCtx(t, "windows:\n  sign:\n    subject: XueHua Tech\n")
	c.Tools.(*fakeTools).bins["iscc"] = `C:\Inno\ISCC.exe`
	cs := strings.Join(cmds(plan(t, &WinExe{}, c)), "\n")
	if !strings.Contains(cs, "/Sfpack=") || !strings.Contains(cs, "/n $qXueHua Tech$q") || !strings.Contains(cs, "verify /pa") {
		t.Fatal(cs)
	}
	p := plan(t, &Msix{}, c)
	cs = strings.Join(cmds(p), "\n")
	if !strings.Contains(cs, `--signtool-options '/fd sha256`) || !strings.Contains(cs, `/n "XueHua Tech"`) || !strings.Contains(cs, "verify /pa") {
		t.Fatal(cs)
	}
	// dlib: msix:create cannot use the plugin, fpack signs the package itself
	c = signCtx(t, "windows:\n  sign:\n    dlib: d.dll\n    dlib_metadata: m.json\n  msix:\n    publisher: CN=XueHua\n")
	cs = strings.Join(cmds(plan(t, &Msix{}, c)), "\n")
	if !strings.Contains(cs, "--sign-msix false") || !strings.Contains(cs, "sign /fd sha256") || !strings.Contains(cs, "/dlib") {
		t.Fatal(cs)
	}
}

func TestSignPreflight(t *testing.T) {
	c := signCtx(t, "windows:\n  sign:\n    certificate: missing.pfx\n")
	c.Tools.(*fakeTools).bins["signtool"] = ""
	msgs := ""
	for _, is := range signPreflight(c) {
		msgs += is.Msg + " | " + is.Fix + "\n"
	}
	if !strings.Contains(msgs, "signtool.exe was not found") || !strings.Contains(msgs, "winget install Microsoft.WindowsSDK") || !strings.Contains(msgs, "missing.pfx") {
		t.Fatal(msgs)
	}
	c = signCtx(t, "windows:\n  sign:\n    dlib: d.dll\n")
	if is := signPreflight(c); len(is) != 1 || !is[0].Fatal || !strings.Contains(is[0].Msg, "dlib_metadata") {
		t.Fatal(is)
	}
	if is := signPreflight(signCtx(t, "")); len(is) != 0 {
		t.Fatal(is)
	}
}

func TestSdkToolsNewestFirst(t *testing.T) {
	pf := t.TempDir()
	for _, v := range []string{"10.0.9200.0", "10.0.26100.0", "10.0.22621.0"} {
		p := filepath.Join(pf, "Windows Kits", "10", "bin", v, "x64")
		must(t, os.MkdirAll(p, 0o755))
		must(t, os.WriteFile(filepath.Join(p, "signtool.exe"), nil, 0o644))
	}
	got := sdkTools(pf, "signtool.exe")
	if len(got) != 3 || sdkVersion(got[0]) != "10.0.26100.0" || sdkVersion(got[2]) != "10.0.9200.0" {
		t.Fatal(got)
	}
}
