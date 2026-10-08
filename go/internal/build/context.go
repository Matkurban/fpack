// Package build plans and executes `fpack build`: it resolves targets,
// runs preflight checks, shares Flutter builds between targets, packages
// artifacts, writes checksums and renders the summary.
package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/targets"
)

// Options configure context creation.
type Options struct {
	ProjectDir  string
	ConfigPath  string
	FlutterPath string // --flutter
	// Override applies command-line flags on top of config+env.
	Override    func(*config.Config) error
	PassArgs    []string
	DryRun      bool
	Interactive bool
	RequireSDK  bool
	Getenv      func(string) string
	Tools       targets.Tools
	Host        *host.Host
}

// ConfigError wraps config problems (exit code 2).
type ConfigError struct{ Err error }

func (e *ConfigError) Error() string { return e.Err.Error() }
func (e *ConfigError) Unwrap() error { return e.Err }

// NewContext detects the project, loads config (flags > env > fpack.yaml >
// defaults), locates Flutter and resolves signing.
func NewContext(o Options) (*targets.Context, error) {
	if o.Getenv == nil {
		o.Getenv = os.Getenv
	}
	if o.Tools == nil {
		o.Tools = targets.SystemTools{}
	}
	dir := o.ProjectDir
	if dir == "" {
		dir = "."
	}
	proj, err := project.Find(dir)
	if err != nil {
		return nil, err
	}
	explicit := o.ConfigPath
	if explicit == "" {
		explicit = o.Getenv("FPACK_CONFIG")
	}
	cfgPath, err := config.Find(proj.Root, explicit)
	if err != nil {
		return nil, &ConfigError{err}
	}
	cfg, err := config.Load(cfgPath, o.Getenv)
	if err != nil {
		return nil, &ConfigError{err}
	}
	cfgSDK := cfg.Flutter.SDK
	if err := config.ApplyEnv(cfg, o.Getenv); err != nil {
		return nil, &ConfigError{err}
	}
	envSDK := o.Getenv("FPACK_FLUTTER")
	if o.Override != nil {
		if err := o.Override(cfg); err != nil {
			return nil, &ConfigError{err}
		}
	}
	if probs := cfg.Validate(); len(probs) > 0 {
		return nil, &ConfigError{errors.New(joinLines(probs))}
	}

	h := host.Current()
	if o.Host != nil {
		h = *o.Host
	}
	c := &targets.Context{Project: proj, Config: cfg, Host: h, Tools: o.Tools, DryRun: o.DryRun, Interactive: o.Interactive, PassArgs: o.PassArgs}

	sdk, sdkErr := flutter.Locate(proj.Root, []flutter.Candidate{
		{Path: o.FlutterPath, Source: "--flutter"},
		{Path: envSDK, Source: "FPACK_FLUTTER"},
		{Path: cfgSDK, Source: "fpack.yaml flutter.sdk"},
	})
	if sdkErr != nil && o.RequireSDK {
		return nil, sdkErr
	}
	c.SDK = sdk
	if c.SDK == nil {
		c.SDK = &flutter.SDK{Flutter: "flutter", Dart: "dart"}
	}

	c.AppName = pack.SanitizeName(firstNonEmpty(cfg.App.Name, proj.Name))
	c.BuildName = firstNonEmpty(string(cfg.Build.BuildName), proj.Version)
	c.BuildNumber = firstNonEmpty(string(cfg.Build.BuildNumber), proj.BuildNumber)
	outRel, err := pack.RenderDir(cfg.OutputDir(), pack.Fields{App: c.AppName, Version: c.BuildName, Build: c.BuildNumber, Mode: cfg.Mode(), Flavor: cfg.Build.Flavor})
	if err != nil {
		return nil, &ConfigError{err}
	}
	c.OutDir = proj.Abs(outRel)
	c.WorkDir = filepath.Join(proj.Root, "build", "fpack")

	if c.Mac, err = targets.ResolveMacSigning(proj, cfg); err != nil {
		return nil, &ConfigError{err}
	}
	if c.Signing, err = targets.ResolveAndroidSigning(cfg, proj.Root, c.WorkDir, o.Getenv); err != nil {
		return nil, &ConfigError{err}
	}
	if len(cfg.UnsetEnv) > 0 {
		// Not fatal by itself (the value may be optional) but worth knowing.
		fmt.Fprintln(os.Stderr, i18n.F("fpack: warning: %s references unset environment variables: %v", "fpack：警告：%s 引用了未设置的环境变量：%v", filepath.Base(cfg.File), cfg.UnsetEnv))
	}
	return c, nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func joinLines(l []string) string {
	out := ""
	for i, s := range l {
		if i > 0 {
			out += "\n"
		}
		out += s
	}
	return out
}
