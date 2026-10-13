package targets

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func withOSRelease(t *testing.T, content string) {
	t.Helper()
	oldRoot := isRoot
	isRoot = func() bool { return true }
	t.Cleanup(func() { isRoot = oldRoot })
	p := filepath.Join(t.TempDir(), "os-release")
	must(t, os.WriteFile(p, []byte(content), 0o644))
	old := osReleasePath
	osReleasePath = p
	t.Cleanup(func() { osReleasePath = old })
}

const noble = "NAME=\"Ubuntu\"\nVERSION_ID=\"24.04\"\nID=ubuntu\n"

type snapYAML struct {
	Name        string `yaml:"name"`
	Base        string `yaml:"base"`
	Version     string `yaml:"version"`
	Summary     string `yaml:"summary"`
	Grade       string `yaml:"grade"`
	Confinement string `yaml:"confinement"`
	License     string `yaml:"license"`
	Apps        map[string]struct {
		Command    string   `yaml:"command"`
		Extensions []string `yaml:"extensions"`
		Plugs      []string `yaml:"plugs"`
	} `yaml:"apps"`
	Parts map[string]struct {
		Plugin        string   `yaml:"plugin"`
		Source        string   `yaml:"source"`
		StagePackages []string `yaml:"stage-packages"`
	} `yaml:"parts"`
}

func parseSnap(t *testing.T, s string) snapYAML {
	t.Helper()
	var y snapYAML
	if err := yaml.Unmarshal([]byte(s), &y); err != nil {
		t.Fatalf("invalid snapcraft.yaml: %v\n%s", err, s)
	}
	return y
}

func TestSnapcraftYAMLDefaults(t *testing.T) {
	withOSRelease(t, noble)
	c := newCtx(t, "linux", "app:\n  homepage: https://x.dev\n  description: \"A fast: secure \\\"messenger\\\"\"\n  license: MIT\n", "")
	y := parseSnap(t, SnapcraftYAML(c))
	if y.Name != "xue-hua-im" || y.Base != "core24" || y.Version != "1.0.0+1" || y.Grade != "stable" || y.Confinement != "strict" || y.License != "MIT" {
		t.Errorf("header: %+v", y)
	}
	if !strings.Contains(SnapcraftYAML(c), "website: \"https://x.dev\"") {
		t.Error("website missing")
	}
	if y.Summary != `A fast: secure "messenger"` {
		t.Errorf("summary %q", y.Summary)
	}
	app := y.Apps["xue-hua-im"]
	if app.Command != "xue_hua_im" || !slices.Equal(app.Extensions, []string{"gnome"}) || !slices.Contains(app.Plugs, "wayland") || !slices.Contains(app.Plugs, "opengl") {
		t.Errorf("app: %+v", app)
	}
	if p := y.Parts["xue-hua-im"]; p.Plugin != "dump" || p.Source != "bundle" {
		t.Errorf("part: %+v", p)
	}
	if m := c.SnapBuildMode(); m != SnapDestructive {
		t.Errorf("auto on Ubuntu 24.04 = %s", m)
	}
	p := plan(t, &Snap{}, c)
	if cs := cmds(p); len(cs) != 1 || !strings.HasPrefix(cs[0], "snapcraft pack --destructive-mode --output ") {
		t.Errorf("cmds %v", cs)
	}
	if n := names(p); len(n) != 1 || n[0] != "xue_hua_im-1.0.0+1-linux-arm64.snap" {
		t.Errorf("names %v", n)
	}
}

func TestSnapOptions(t *testing.T) {
	withOSRelease(t, "ID=debian\nVERSION_ID=13\n")
	c := newCtx(t, "linux", `linux:
  snap:
    name: xuehua
    summary: Short
    base: core22
    confinement: devmode
    plugs: []
    extensions: [gnome]
    stage_packages: [libsecret-1-0]
    apps: {helper: tools/helper}
    extra_args: [--verbosity, verbose]
`, "")
	y := parseSnap(t, SnapcraftYAML(c))
	if y.Name != "xuehua" || y.Base != "core22" || y.Grade != "devel" || y.Confinement != "devmode" || y.Summary != "Short" {
		t.Errorf("header: %+v", y)
	}
	if a := y.Apps["xuehua"]; len(a.Plugs) != 0 {
		t.Errorf("plugs: [] must give no plugs, got %v", a.Plugs)
	}
	if a := y.Apps["helper"]; a.Command != "tools/helper" || len(a.Extensions) != 1 {
		t.Errorf("helper app: %+v", a)
	}
	if p := y.Parts["xuehua"]; !slices.Equal(p.StagePackages, []string{"libsecret-1-0"}) {
		t.Errorf("stage-packages: %v", p.StagePackages)
	}
	if m := c.SnapBuildMode(); m != SnapLXD {
		t.Errorf("auto on Debian = %s", m)
	}
	if cs := cmds(plan(t, &Snap{}, c)); !strings.HasPrefix(cs[0], "snapcraft pack --use-lxd --output ") || !strings.HasSuffix(cs[0], " --verbosity verbose") {
		t.Errorf("cmds %v", cs)
	}
	// lxd missing is fatal with the install hint.
	fatals, _ := preflightOf(t, &Snap{}, c)
	if !slices.ContainsFunc(fatals, func(i Issue) bool {
		return strings.Contains(i.Msg, "snapcraft") && strings.Contains(i.Fix, "snap install snapcraft --classic")
	}) ||
		!slices.ContainsFunc(fatals, func(i Issue) bool { return strings.Contains(i.Msg, "lxd") }) {
		t.Errorf("preflight: %+v", fatals)
	}
}

func TestSnapClassicAndBuildModes(t *testing.T) {
	withOSRelease(t, "ID=debian\n")
	c := newCtx(t, "linux", "linux:\n  snap:\n    confinement: classic\n    build_mode: destructive\n", "")
	y := parseSnap(t, SnapcraftYAML(c))
	if a := y.Apps["xue-hua-im"]; len(a.Plugs)+len(a.Extensions) != 0 {
		t.Errorf("classic must not plug or use extensions: %+v", a)
	}
	c.Tools.(*fakeTools).bins["snapcraft"] = "/snap/bin/snapcraft"
	fatals, _ := preflightOf(t, &Snap{}, c)
	if len(fatals) != 1 || !strings.Contains(fatals[0].Msg, "24.04") || !strings.Contains(fatals[0].Fix, "lxd") {
		t.Errorf("destructive on Debian: %+v", fatals)
	}
	c.Config.Linux.Snap.BuildMode = SnapMultipass
	args, env := SnapArgs(c, c.SnapBuildMode(), "out.snap")
	if strings.Join(args, " ") != "pack --output out.snap" || !slices.Equal(env, []string{"SNAPCRAFT_BUILD_ENVIRONMENT=multipass"}) {
		t.Errorf("multipass: %v %v", args, env)
	}
}

func TestSnapDestructiveUsesSudoWhenNotRoot(t *testing.T) {
	withOSRelease(t, noble)
	isRoot = func() bool { return false }
	c := newCtx(t, "linux", "", "")
	cs := cmds(plan(t, &Snap{}, c))
	if len(cs) != 2 || !strings.HasPrefix(cs[0], "sudo -n snapcraft pack --destructive-mode --output ") || !strings.HasPrefix(cs[1], "sudo -n chown -R ") {
		t.Errorf("cmds %v", cs)
	}
	c.DryRun = false
	c.Tools.(*fakeTools).bins["snapcraft"] = "/snap/bin/snapcraft"
	c.Tools.(*fakeTools).fail["sudo -n true"] = true
	fatals, _ := preflightOf(t, &Snap{}, c)
	if len(fatals) != 1 || !strings.Contains(fatals[0].Fix, "sudo -v") {
		t.Errorf("sudo with password: %+v", fatals)
	}
}

func TestSnapVersionAndSummary(t *testing.T) {
	c := newCtx(t, "linux", "app:\n  description: "+strings.Repeat("x", 100)+"\n", "")
	c.BuildName, c.BuildNumber = "1.2.3_beta 4", ""
	if v := snapVersion(c); v != "1.2.3-beta-4" {
		t.Errorf("version %q", v)
	}
	c.BuildName = strings.Repeat("9", 40)
	if v := snapVersion(c); len(v) != 32 {
		t.Errorf("version %q not cut to 32", v)
	}
	if s := []rune(c.snapSummary()); len(s) != 78 {
		t.Errorf("summary length %d", len(s))
	}
}

// TestSnapStageAndFakeSnapcraft runs the packaging plan with a fake
// snapcraft that checks the staged project and writes the snap.
func TestSnapStageAndFakeSnapcraft(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake")
	}
	withOSRelease(t, noble)
	c := newCtx(t, "linux", "linux:\n  snap:\n    template: snap.tmpl\n", "")
	must(t, os.WriteFile(c.Project.Abs("snap.tmpl"), []byte("name: {name}\nversion: {version}\nconfinement: {confinement}\napps:\n  a:\n    command: {command}\n"), 0o644))
	in, err := (&Snap{}).Locate(c, true, c.Started)
	if err != nil {
		t.Fatal(err)
	}
	must(t, os.MkdirAll(in["bundle"], 0o755))
	must(t, os.WriteFile(filepath.Join(in["bundle"], "xue_hua_im"), []byte("#!/bin/sh\n"), 0o755))
	fake := filepath.Join(t.TempDir(), "snapcraft")
	must(t, os.WriteFile(fake, []byte(`#!/bin/sh
set -e
test "$1" = pack
test -x bundle/xue_hua_im
test -f snap/gui/xue-hua-im.desktop
test -f snap/gui/xue-hua-im.png
grep -q '^name: "xue-hua-im"$' snap/snapcraft.yaml
grep -q '^    command: "xue_hua_im"$' snap/snapcraft.yaml
while [ "$1" != --output ]; do shift; done
echo snap > "$2"
`), 0o755))
	p, err := (&Snap{}).Package(c, in)
	if err != nil {
		t.Fatal(err)
	}
	must(t, os.MkdirAll(c.OutDir, 0o755))
	for _, op := range p.Ops {
		switch {
		case op.Fn != nil:
			must(t, op.Fn())
		case op.Cmd != nil:
			cmd := exec.Command(fake, op.Cmd.Args...)
			cmd.Dir = op.Cmd.Dir
			if out, cerr := cmd.CombinedOutput(); cerr != nil {
				t.Fatalf("fake snapcraft: %v\n%s", cerr, out)
			}
		}
	}
	if !exists(p.Artifacts[0].Path) {
		t.Fatal("snap not moved to the output directory")
	}
	desktop, err := os.ReadFile(filepath.Join(c.Stage("snap"), "snap", "gui", "xue-hua-im.desktop"))
	must(t, err)
	if !strings.Contains(string(desktop), "Exec=xue-hua-im %U") || !strings.Contains(string(desktop), "Icon=${SNAP}/meta/gui/xue-hua-im.png") {
		t.Errorf("desktop:\n%s", desktop)
	}
}
