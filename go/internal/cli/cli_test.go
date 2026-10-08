package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/build"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
)

// fixture creates a Flutter-like project and a fake Flutter SDK.
func fixture(t *testing.T, fpackYAML string) (proj, sdk string) {
	t.Helper()
	dir := t.TempDir()
	proj = filepath.Join(dir, "app")
	sdk = filepath.Join(dir, "flutter")
	files := map[string]string{
		"app/pubspec.yaml":                       "name: my_app\nversion: 2.3.4+5\nenvironment:\n  sdk: ^3.0.0\ndependencies:\n  flutter:\n    sdk: flutter\n",
		"app/lib/main.dart":                      "void main() {}\n",
		"app/web/index.html":                     "<html></html>\n",
		"app/android/app/build.gradle.kts":       "android {\n  buildTypes {\n    release {\n      signingConfig = signingConfigs.getByName(\"debug\")\n    }\n  }\n}\n",
		"app/ios/Runner/Info.plist":              "<plist/>\n",
		"flutter/bin/flutter":                    "#!/bin/sh\necho fake flutter \"$@\"\n",
		"flutter/bin/flutter.bat":                "@echo fake\n",
		"flutter/packages/flutter_tools/x":       "",
		"flutter/bin/cache/flutter.version.json": `{"frameworkVersion":"3.47.6","flutterVersion":"3.47.6","channel":"stable","dartSdkVersion":"3.13.5"}`,
	}
	if fpackYAML != "" {
		files["app/fpack.yaml"] = fpackYAML
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return proj, sdk
}

type result struct {
	code           int
	stdout, stderr string
}

// all is the combined human output (stdout without --json, stderr with it).
func (r result) all() string { return r.stdout + r.stderr }

func run(t *testing.T, env map[string]string, args ...string) result {
	t.Helper()
	defer i18n.Set(i18n.EN)
	var out, errb bytes.Buffer
	if env == nil {
		env = map[string]string{}
	}
	if _, ok := env["FPACK_LANG"]; !ok {
		env["FPACK_LANG"] = "en"
	}
	code := Main(args, &Env{Stdout: &out, Stderr: &errb, Stdin: strings.NewReader(""), Getenv: func(k string) string { return env[k] }})
	return result{code, out.String(), errb.String()}
}

func summary(t *testing.T, r result) build.Summary {
	t.Helper()
	var s build.Summary
	if err := json.Unmarshal([]byte(r.stdout), &s); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s\nstderr:\n%s", err, r.stdout, r.stderr)
	}
	return s
}

func target(s build.Summary, name string) *build.TargetResult {
	for _, t := range s.Targets {
		if t.Target == name {
			return t
		}
	}
	return nil
}

func TestDryRunJSONWeb(t *testing.T) {
	proj, sdk := fixture(t, "")
	r := run(t, nil, "build", "web", "--dry-run", "--json", "-C", proj, "--flutter", sdk)
	if r.code != 0 {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	s := summary(t, r)
	web := target(s, "web")
	if web == nil || web.Status != build.Planned || len(web.Artifacts) != 1 {
		t.Fatalf("%+v", web)
	}
	if got := web.Artifacts[0].File; got != "my_app-2.3.4+5-web.zip" {
		t.Fatal(got)
	}
	if !strings.Contains(strings.Join(web.Commands, "\n"), "build web --release") {
		t.Fatalf("%v", web.Commands)
	}
	if s.Version != "2.3.4" || s.BuildNumber != "5" || !s.DryRun {
		t.Fatalf("%+v", s)
	}
	// Nothing written by a dry run.
	if _, err := os.Stat(filepath.Join(proj, "dist")); err == nil {
		t.Fatal("dry run created dist/")
	}
}

func TestPrecedenceFlagsEnvConfig(t *testing.T) {
	proj, sdk := fixture(t, "build:\n  mode: profile\n  build_number: 11\noutput:\n  dir: out/{version}\n")
	plan := func(env map[string]string, extra ...string) *build.TargetResult {
		args := append([]string{"build", "web", "--dry-run", "--json", "-C", proj, "--flutter", sdk}, extra...)
		r := run(t, env, args...)
		if r.code != 0 {
			t.Fatalf("exit %d\n%s", r.code, r.all())
		}
		return target(summary(t, r), "web")
	}
	cmd := func(tr *build.TargetResult) string { return strings.Join(tr.Commands, "\n") }

	// config only
	w := plan(nil)
	if !strings.Contains(cmd(w), "--profile") || !strings.Contains(cmd(w), "--build-number 11") {
		t.Fatal(cmd(w))
	}
	if !strings.Contains(w.Artifacts[0].Path, filepath.Join("out", "2.3.4")) || w.Artifacts[0].File != "my_app-2.3.4+11-web-profile.zip" {
		t.Fatalf("%+v", w.Artifacts[0])
	}
	// env beats config
	w = plan(map[string]string{"FPACK_MODE": "debug", "FPACK_BUILD_NUMBER": "12"})
	if !strings.Contains(cmd(w), "--debug") || !strings.Contains(cmd(w), "--build-number 12") {
		t.Fatal(cmd(w))
	}
	// flags beat env
	w = plan(map[string]string{"FPACK_MODE": "debug", "FPACK_BUILD_NUMBER": "12"}, "--release", "--build-number", "13")
	if !strings.Contains(cmd(w), "--release") || !strings.Contains(cmd(w), "--build-number 13") || w.Artifacts[0].File != "my_app-2.3.4+13-web.zip" {
		t.Fatalf("%s %+v", cmd(w), w.Artifacts[0])
	}
}

func TestConflictWithoutForce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake flutter is a shell script")
	}
	proj, sdk := fixture(t, "")
	existing := filepath.Join(proj, "dist", "2.3.4+5", "my_app-2.3.4+5-web.zip")
	os.MkdirAll(filepath.Dir(existing), 0o755)
	os.WriteFile(existing, []byte("old"), 0o644)
	r := run(t, nil, "build", "web", "-C", proj, "--flutter", sdk)
	if r.code != build.ExitPrereq || !strings.Contains(r.all(), "already exist") || !strings.Contains(r.all(), "--force") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	if b, _ := os.ReadFile(existing); string(b) != "old" {
		t.Fatal("artifact was overwritten")
	}
	// dry run reports the conflict but succeeds
	r = run(t, nil, "build", "web", "--dry-run", "--json", "-C", proj, "--flutter", sdk)
	if r.code != 0 || len(summary(t, r).Conflicts) != 1 {
		t.Fatalf("exit %d %s", r.code, r.stdout)
	}
}

func TestAllSkipsWhatHostCannotBuild(t *testing.T) {
	proj, sdk := fixture(t, "")
	r := run(t, nil, "build", "--all", "--dry-run", "--json", "-C", proj, "--flutter", sdk)
	if r.code != 0 {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	s := summary(t, r)
	h := host.Current()
	for _, tr := range s.Targets {
		ok, _ := h.CanBuild(host.Platform(tr.Platform))
		switch {
		case !ok && (tr.Status != build.Skipped || !strings.Contains(tr.Reason, "needs")):
			t.Errorf("%s should be skipped on %s: %+v", tr.Target, h.OS, tr)
		case tr.Platform == "linux" || tr.Platform == "windows" || tr.Platform == "macos":
			// fixture has no desktop folders
			if ok && tr.Status != build.Skipped {
				t.Errorf("%s: project has no folder, want skipped: %+v", tr.Target, tr)
			}
		}
	}
	if w := target(s, "web"); w.Status != build.Planned {
		t.Fatalf("web: %+v", w)
	}
}

func TestExplicitUnsupportedTargetIsReferenceInDryRunAndFailsForReal(t *testing.T) {
	proj, sdk := fixture(t, "")
	other := "ipa"
	if runtime.GOOS == "darwin" {
		t.Skip("ipa is buildable here")
	}
	r := run(t, nil, "build", other, "--dry-run", "--json", "-C", proj, "--flutter", sdk)
	tr := target(summary(t, r), other)
	if r.code != 0 || tr.Status != build.Planned || !tr.ReferenceOnly || len(tr.Commands) == 0 {
		t.Fatalf("exit %d %+v", r.code, tr)
	}
	r = run(t, nil, "build", other, "--json", "-C", proj, "--flutter", sdk)
	tr = target(summary(t, r), other)
	if r.code != build.ExitPrereq || tr.Status != build.Failed || tr.Fix == "" {
		t.Fatalf("exit %d %+v", r.code, tr)
	}
}

func TestUsageErrors(t *testing.T) {
	proj, sdk := fixture(t, "")
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"build", "apkk", "-C", proj}, `did you mean "apk"`},
		{[]string{"build", "--all", "web", "-C", proj}, "either --all"},
		{[]string{"build", "web", "--flavour", "x", "-C", proj}, "--flavor"},
		{[]string{"biuld", "web"}, "build"},
		{[]string{"build", "web", "--mode", "fast", "-C", proj, "--flutter", sdk}, "mode"},
		{[]string{"--lang", "fr", "list"}, "--lang"},
	}
	for _, c := range cases {
		r := run(t, nil, c.args...)
		if r.code != build.ExitUsage || !strings.Contains(r.stderr+r.stdout, c.want) {
			t.Errorf("%v: exit %d, want %q in\n%s%s", c.args, r.code, c.want, r.stdout, r.stderr)
		}
	}
}

func TestNoTargetsAndNoConfigIsUsageError(t *testing.T) {
	proj, sdk := fixture(t, "")
	r := run(t, nil, "build", "-C", proj, "--flutter", sdk)
	if r.code != build.ExitUsage || !strings.Contains(r.all(), "which targets") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
}

func TestConfigTargetsAndUnknownKey(t *testing.T) {
	proj, sdk := fixture(t, "build:\n  targets: [web]\n")
	r := run(t, nil, "build", "--dry-run", "--json", "-C", proj, "--flutter", sdk)
	if r.code != 0 || target(summary(t, r), "web") == nil {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
	proj, sdk = fixture(t, "build:\n  modee: release\n")
	r = run(t, nil, "build", "web", "--dry-run", "-C", proj, "--flutter", sdk)
	if r.code != build.ExitUsage || !strings.Contains(r.all(), "mode") {
		t.Fatalf("exit %d\n%s", r.code, r.all())
	}
}

func TestLanguage(t *testing.T) {
	proj, sdk := fixture(t, "")
	r := run(t, map[string]string{"FPACK_LANG": "zh"}, "build", "web", "--dry-run", "-C", proj, "--flutter", sdk)
	if !strings.Contains(r.stderr+r.stdout, "构建计划") {
		t.Fatalf("%s%s", r.stdout, r.stderr)
	}
	r = run(t, map[string]string{"LANG": "zh_CN.UTF-8", "FPACK_LANG": ""}, "--lang", "en", "build", "web", "--dry-run", "-C", proj, "--flutter", sdk)
	if !strings.Contains(r.stderr+r.stdout, "Plan (dry run") {
		t.Fatalf("%s%s", r.stdout, r.stderr)
	}
}

func TestVersionAndCoreVersion(t *testing.T) {
	r := run(t, nil, "--core-version")
	if r.code != 0 || strings.TrimSpace(r.stdout) == "" || strings.Contains(r.stdout, " ") {
		t.Fatalf("%q", r.stdout)
	}
	r = run(t, map[string]string{"FPACK_WRAPPER_VERSION": "0.0.1"}, "--version")
	if !strings.Contains(r.all(), "0.0.1") {
		t.Fatalf("expected wrapper mismatch warning: %q", r.stderr)
	}
}

func TestInitYesWritesConfigOnce(t *testing.T) {
	proj, sdk := fixture(t, "")
	r := run(t, nil, "init", "--yes", "-C", proj, "--flutter", sdk)
	if r.code != 0 {
		t.Fatalf("exit %d\n%s%s", r.code, r.stdout, r.stderr)
	}
	b, err := os.ReadFile(filepath.Join(proj, "fpack.yaml"))
	if err != nil || !strings.Contains(string(b), "targets:") {
		t.Fatalf("%v %s", err, b)
	}
	// The generated file must be valid config.
	r = run(t, nil, "build", "web", "--dry-run", "-C", proj, "--flutter", sdk)
	if r.code != 0 {
		t.Fatalf("generated fpack.yaml rejected: %s", r.stderr)
	}
	// Second init refuses to overwrite.
	r = run(t, nil, "init", "--yes", "-C", proj, "--flutter", sdk)
	if r.code == 0 {
		t.Fatalf("init overwrote existing fpack.yaml\n%s%s", r.stdout, r.stderr)
	}
	// pubspec untouched
	if b, _ := os.ReadFile(filepath.Join(proj, "pubspec.yaml")); !strings.HasPrefix(string(b), "name: my_app\n") {
		t.Fatal("pubspec modified")
	}
}

func TestRealRunWithFakeFlutter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake flutter is a shell script")
	}
	proj, sdk := fixture(t, "")
	script := `#!/bin/sh
case "$2" in
  web) mkdir -p build/web && echo '<html>ok</html>' > build/web/index.html && echo "Compiling lib/main.dart for the Web..." ;;
  apk) echo "FAILURE: Build failed with an exception."; echo "* What went wrong:"; echo "Execution failed for task ':app:minifyReleaseWithR8'."; echo "> Missing classes detected while running R8."; echo "* Try:"; exit 1 ;;
esac
`
	os.WriteFile(filepath.Join(sdk, "bin", "flutter"), []byte(script), 0o755)

	r := run(t, nil, "build", "web", "--json", "-C", proj, "--flutter", sdk)
	s := summary(t, r)
	if r.code != 0 || !s.Success {
		t.Fatalf("exit %d\n%s", r.code, r.stderr)
	}
	art := target(s, "web").Artifacts[0]
	if art.File != "my_app-2.3.4+5-web.zip" || art.Size == 0 || len(art.SHA256) != 64 {
		t.Fatalf("%+v", art)
	}
	sums, err := os.ReadFile(filepath.Join(proj, "dist", "2.3.4+5", "SHA256SUMS"))
	if err != nil || !strings.Contains(string(sums), art.SHA256+"  my_app-2.3.4+5-web.zip") {
		t.Fatalf("%v %s", err, sums)
	}

	// A failing flutter build: exit 1, excerpt, hint and log path.
	r = run(t, nil, "build", "apk", "--json", "-C", proj, "--flutter", sdk)
	s = summary(t, r)
	apk := target(s, "apk")
	if r.code != build.ExitFailed || apk.Status != build.Failed {
		t.Fatalf("exit %d %+v", r.code, apk)
	}
	if apk.Hint == "" || !strings.Contains(strings.Join(apk.Excerpt, "\n"), "R8") || apk.Log == "" {
		t.Fatalf("%+v", apk)
	}
	if _, err := os.Stat(apk.Log); err != nil {
		t.Fatalf("log missing: %v", err)
	}
}
