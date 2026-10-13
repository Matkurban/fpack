package targets

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
)

// WebZip zips build/web.
type WebZip struct{}

// Name implements [Target].
func (*WebZip) Name() string { return "web" }

// Platform implements [Target].
func (*WebZip) Platform() host.Platform { return host.Web }

// Formats implements [Target].
func (*WebZip) Formats() []string { return []string{".zip"} }

// Optional implements [Target].
func (*WebZip) Optional() bool { return false }

// Description implements [Target].
func (*WebZip) Description() string {
	return i18n.S("Web build (zip of build/web, ready to deploy)", "Web 构建（build/web 的 zip，可直接部署）")
}

// Preflight implements [Target].
func (*WebZip) Preflight(*Context) []Issue { return nil }

// Steps implements [Target].
func (*WebZip) Steps(c *Context) ([]FlutterStep, error) {
	args, w := CommonArgs(c, host.Web, "web")
	if b := c.Config.Web.BaseHref; b != "" {
		args = append(args, "--base-href", b)
	}
	w2 := c.Config.Web
	if boolOr(w2.Wasm, false) {
		args = append(args, "--wasm")
	}
	if w2.SourceMaps != nil {
		args = append(args, map[bool]string{true: "--source-maps", false: "--no-source-maps"}[*w2.SourceMaps])
	}
	if boolOr(w2.CSP, false) {
		args = append(args, "--csp")
	}
	if o := w2.OptimizationLevel; o != nil {
		args = append(args, fmt.Sprintf("-O%d", *o))
	}
	if u := w2.StaticAssetsURL; u != "" {
		args = append(args, "--static-assets-url", u)
	}
	if w2.WebResourcesCDN != nil {
		args = append(args, map[bool]string{true: "--web-resources-cdn", false: "--no-web-resources-cdn"}[*w2.WebResourcesCDN])
	}
	for _, k := range sortedKeys(w2.WebDefine) {
		args = append(args, "--web-define="+k+"="+w2.WebDefine[k])
	}
	args = append(args, tailArgs(c, c.Config.Web.ExtraArgs)...)
	return []FlutterStep{{Key: "web", Platform: host.Web, Args: args, Warnings: w}}, nil
}

// Locate implements [Target].
func (*WebZip) Locate(c *Context, predicted bool, _ time.Time) (Inputs, error) {
	dir := filepath.Join(c.Project.Root, "build", "web")
	if predicted || exists(filepath.Join(dir, "index.html")) {
		return Inputs{"dir": dir}, nil
	}
	return nil, notFound("index.html", c.Rel(dir))
}

// Package implements [Target].
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
