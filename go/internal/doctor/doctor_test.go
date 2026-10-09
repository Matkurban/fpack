package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/targets"
)

type noTools struct{}

func (noTools) Find(string) string                                  { return "" }
func (noTools) Probe(string, ...string) (string, bool)              { return "", false }
func (noTools) ProbeEnv([]string, string, ...string) (string, bool) { return "", false }

func androidCtx(t *testing.T, env *targets.AndroidEnv) *targets.Context {
	t.Helper()
	root := t.TempDir()
	for rel, content := range map[string]string{
		"pubspec.yaml":                 "name: demo\nversion: 1.0.0+1\ndependencies:\n  flutter:\n    sdk: flutter\n",
		"android/app/build.gradle.kts": "android { }\n",
	} {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	p, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	c := &targets.Context{
		Project: p, Config: &config.Config{}, SDK: &flutter.SDK{Root: "/sdk", Flutter: "/sdk/bin/flutter"},
		Host: host.Host{OS: "linux", Arch: "amd64"}, Tools: noTools{}, AppName: "demo", BuildName: "1.0.0", BuildNumber: "1",
		OutDir: filepath.Join(root, "dist"), WorkDir: filepath.Join(root, "build", "fpack"), DryRun: true,
	}
	c.SetAndroidEnv(env)
	return c
}

func TestMissingJavaMeansAndroidNotReady(t *testing.T) {
	apk, _ := targets.Get("apk")
	aab, _ := targets.Get("aab")
	run := func(env *targets.AndroidEnv) *Report {
		c := androidCtx(t, env)
		return Run(Input{Ctx: c, SDK: c.SDK, Host: c.Host, Tools: noTools{}, Only: []targets.Target{apk, aab}})
	}
	r := run(&targets.AndroidEnv{SDK: "/android", Licenses: true})
	if len(r.Ready) != 0 {
		t.Fatalf("without Java apk/aab must not be ready: %v", r.Ready)
	}
	var java []string
	for _, g := range r.Groups {
		for _, l := range g.Lines {
			if strings.Contains(l.Text, "no Java") {
				java = append(java, l.Text)
			}
		}
	}
	if len(java) != 1 {
		t.Fatalf("one Java line expected, got %v", java)
	}
	r = run(&targets.AndroidEnv{SDK: "/android", Licenses: true, Java: "/jdk/bin/java", JavaVersion: 21})
	if strings.Join(r.Ready, " ") != "apk aab" {
		t.Fatalf("with Java: %v", r.Ready)
	}
}
