package targets

import (
	"path/filepath"
	"time"

	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
)

// WebZip zips build/web.
type WebZip struct{}

func (*WebZip) Name() string            { return "web" }
func (*WebZip) Platform() host.Platform { return host.Web }
func (*WebZip) Formats() []string       { return []string{".zip"} }
func (*WebZip) Optional() bool          { return false }
func (*WebZip) Description() string {
	return i18n.S("Web build (zip of build/web, ready to deploy)", "Web 构建（build/web 的 zip，可直接部署）")
}
func (*WebZip) Preflight(c *Context) []Issue { return nil }
func (*WebZip) Steps(c *Context) ([]FlutterStep, error) {
	args, w := CommonArgs(c, host.Web, "web")
	if b := c.Config.Web.BaseHref; b != "" {
		args = append(args, "--base-href", b)
	}
	if c.Config.Web.Wasm != nil && *c.Config.Web.Wasm {
		args = append(args, "--wasm")
	}
	args = append(args, tailArgs(c, c.Config.Web.ExtraArgs)...)
	return []FlutterStep{{Key: "web", Platform: host.Web, Args: args, Warnings: w}}, nil
}
func (*WebZip) Locate(c *Context, predicted bool, _ time.Time) (Inputs, error) {
	dir := filepath.Join(c.Project.Root, "build", "web")
	if predicted || exists(filepath.Join(dir, "index.html")) {
		return Inputs{"dir": dir}, nil
	}
	return nil, notFound("index.html", c.Rel(dir))
}
func (*WebZip) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Web, "", "", ".zip")
	if err != nil {
		return nil, err
	}
	src := in["dir"]
	stage := c.Stage("web")
	tmp := filepath.Join(stage, filepath.Base(dst))
	return &Plan{Ops: []Op{
		resetDirOp(c, stage),
		{Desc: i18n.F("zip %s", "压缩 %s", c.Rel(src)), Fn: func() error { return pack.Zip(src, tmp, "") }},
		moveOp(c, tmp, dst),
	}, Artifacts: []Artifact{{Path: dst, Kind: "Web (zip)"}}}, nil
}
