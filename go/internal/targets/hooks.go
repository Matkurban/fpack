package targets

import (
	"runtime"
	"strings"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/runner"
)

// ShellCmd runs a hook command through the platform shell in the project root.
func ShellCmd(c *Context, command string, env []string) *runner.Cmd {
	if runtime.GOOS == "windows" {
		return &runner.Cmd{Name: "cmd", Args: []string{"/C", command}, Dir: c.Project.Root, Env: env}
	}
	return &runner.Cmd{Name: "sh", Args: []string{"-c", command}, Dir: c.Project.Root, Env: env}
}

// HookEnv is the environment given to every hook.
func (c *Context) HookEnv() []string {
	return []string{
		"FPACK_PROJECT_ROOT=" + c.Project.Root, "FPACK_OUTPUT_DIR=" + c.OutDir,
		"FPACK_VERSION=" + c.BuildName, "FPACK_BUILD_NUMBER=" + c.BuildNumber,
		"FPACK_MODE=" + c.Mode(), "FPACK_FLAVOR=" + c.Flavor(),
	}
}

func hookOps(c *Context, what string, cmds config.List, env []string) []Op {
	var ops []Op
	for _, h := range cmds {
		ops = append(ops, Op{Desc: what + ": " + h, Cmd: ShellCmd(c, h, append(c.HookEnv(), env...)),
			Hint: i18n.S("the hook command failed; run it by hand in the project root to debug", "钩子命令失败；可在项目根目录手动运行它排查")})
	}
	return ops
}

// PackageFor plans a target's packaging with its per-target name template
// and pre_package/post_package hooks.
func PackageFor(c *Context, t Target, in Inputs) (*Plan, error) {
	c.CurrentTarget = t.Name()
	defer func() { c.CurrentTarget = "" }()
	p, err := t.Package(c, in)
	if err != nil {
		return nil, err
	}
	env := []string{"FPACK_TARGET=" + t.Name()}
	pre := hookOps(c, "hooks.pre_package."+t.Name(), c.Config.Hooks.PrePackage[t.Name()], env)
	var paths []string
	for _, a := range p.Artifacts {
		paths = append(paths, a.Path)
	}
	first := ""
	if len(paths) > 0 {
		first = paths[0]
	}
	post := hookOps(c, "hooks.post_package."+t.Name(), c.Config.Hooks.PostPackage[t.Name()],
		append(env, "FPACK_ARTIFACT="+first, "FPACK_ARTIFACTS="+strings.Join(paths, "\n")))
	p.Ops = append(append(pre, p.Ops...), post...)
	return p, nil
}

// BuildHookOps returns the pre_build or post_build hook operations.
func BuildHookOps(c *Context, post bool, artifacts []string, success bool) []Op {
	if !post {
		return hookOps(c, "hooks.pre_build", c.Config.Hooks.PreBuild, nil)
	}
	ok := "1"
	if !success {
		ok = "0"
	}
	return hookOps(c, "hooks.post_build", c.Config.Hooks.PostBuild,
		[]string{"FPACK_ARTIFACTS=" + strings.Join(artifacts, "\n"), "FPACK_SUCCESS=" + ok})
}

// ConfigIssues reports configured files that do not exist, for the keys that
// affect target t.
func ConfigIssues(c *Context, t string) []Issue {
	var out []Issue
	for _, k := range config.Keys {
		if k.Kind != config.KPath || !k.AppliesTo(t) {
			continue
		}
		v, _ := c.Config.Get(k.Path)
		s, _ := v.(string)
		if s == "" || strings.HasPrefix(k.Path, "linux.appimagetool") || k.Path == "flutter.path" {
			continue
		}
		if !exists(c.Project.Abs(s)) {
			out = append(out, fatal(i18n.F("%s: file not found: %s", "%s：找不到文件：%s", k.Path, s),
				i18n.S("paths are relative to the project root; fix the path in fpack.yaml", "路径相对于项目根目录；请修正 fpack.yaml 中的路径")))
		}
	}
	return out
}
