package targets

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/runner"
)

// Authenticode signing with signtool (windows.sign). One configuration signs
// the app binaries before they are zipped or packaged, the Inno Setup
// installer (and its uninstaller) and the MSIX.

// DefaultTimestampURL is the RFC 3161 server used when none is configured.
const DefaultTimestampURL = "http://timestamp.digicert.com"

// defaultSignFiles are the files of the Release folder signed by default.
var defaultSignFiles = []string{"*.exe", "*.dll"}

// winSigning reports whether windows.sign selects a certificate.
func winSigning(c *Context) bool {
	s := c.Config.Windows.Sign
	return s.Certificate != "" || s.Thumbprint != "" || s.Subject != "" || s.Dlib != ""
}

// WindowsSigning reports whether windows.sign is configured and, if so, the
// signtool fpack would use ("" when missing). Used by fpack doctor.
func WindowsSigning(c *Context) (configured bool, signtoolPath string) {
	if !winSigning(c) {
		return false, ""
	}
	return true, signtool(c)
}

// signtool is windows.sign.signtool, else signtool on PATH or in the newest
// Windows SDK.
func signtool(c *Context) string {
	if p := c.Config.Windows.Sign.Signtool; p != "" {
		return c.Project.Abs(p)
	}
	return c.Tools.Find("signtool")
}

// signtoolArgs are the arguments after "signtool sign" (files excluded):
// digest, timestamp, certificate selection and description. quote wraps
// paths and texts (Inno Setup needs $q).
func signtoolArgs(c *Context, quote func(string) string) (args, secrets []string) {
	s := c.Config.Windows.Sign
	args = []string{"/fd", "sha256", "/tr", orDefault(s.TimestampURL, DefaultTimestampURL), "/td", "sha256"}
	switch {
	case s.Dlib != "":
		args = append(args, "/dlib", quote(c.Project.Abs(s.Dlib)), "/dmdf", quote(c.Project.Abs(s.DlibMetadata)))
	case s.Certificate != "":
		args = append(args, "/f", quote(c.Project.Abs(s.Certificate)))
		if s.Password != "" {
			args = append(args, "/p", s.Password)
			secrets = append(secrets, s.Password)
		}
	default:
		if s.Store != "" {
			args = append(args, "/s", quote(s.Store))
		}
		if boolOr(s.MachineStore, false) {
			args = append(args, "/sm")
		}
		if s.Thumbprint != "" {
			args = append(args, "/sha1", strings.ReplaceAll(s.Thumbprint, " ", ""))
		} else {
			args = append(args, "/n", quote(s.Subject))
		}
	}
	args = append(args, "/d", quote(orDefault(s.Description, c.DisplayName())))
	if s.URL != "" {
		args = append(args, "/du", quote(s.URL))
	}
	return args, secrets
}

func signCmd(c *Context, files []string) *runner.Cmd {
	args, secrets := signtoolArgs(c, func(s string) string { return s })
	return &runner.Cmd{Name: orName(signtool(c), "signtool.exe"), Args: append(append([]string{"sign"}, args...), files...), Secret: secrets}
}

func verifyCmd(c *Context, files []string) *runner.Cmd {
	return &runner.Cmd{Name: orName(signtool(c), "signtool.exe"), Args: append([]string{"verify", "/pa"}, files...)}
}

var signHint = i18n.S("check windows.sign (certificate path/password, thumbprint or subject) and the timestamp server",
	"请检查 windows.sign（证书路径/密码、指纹或主题名称）以及时间戳服务器")

// signOp signs files that exist when the op is planned (an installer, an MSIX).
func signOp(c *Context, file string) Op {
	return Op{Desc: i18n.F("sign %s (signtool)", "签名 %s（signtool）", filepath.Base(file)), Cmd: signCmd(c, []string{file}), Hint: signHint}
}

// verifyOp checks a signature; a failure (e.g. a self-signed test
// certificate whose root is not trusted) is a warning, not an error.
func verifyOp(c *Context, files []string, what string) Op {
	if !boolOr(c.Config.Windows.Sign.Verify, true) {
		return Op{}
	}
	return Op{Desc: i18n.F("verify signature of %s (signtool verify /pa)", "校验 %s 的签名（signtool verify /pa）", what), Cmd: verifyCmd(c, files), Optional: true}
}

// signFilesOps sign (and verify) the windows.sign.files of the Flutter
// Release folder dir, right after flutter build windows, so the portable
// zip, the installer and the MSIX all contain signed binaries.
func signFilesOps(c *Context, dir string) []Op {
	patterns := []string(c.Config.Windows.Sign.Files)
	if len(patterns) == 0 {
		patterns = defaultSignFiles
	}
	preview := []string{filepath.Join(dir, strings.Join(patterns, "|"))}
	what := strings.Join(patterns, ", ")
	files := func() ([]string, error) {
		list, err := matchFiles(dir, patterns)
		if err == nil && len(list) == 0 {
			err = errors.New(i18n.F("no file in %s matches windows.sign.files (%s)", "%s 中没有文件匹配 windows.sign.files（%s）", c.Rel(dir), what))
		}
		return list, err
	}
	ops := []Op{{
		Desc: i18n.F("sign %s (signtool)", "签名 %s（signtool）", what), Hint: signHint,
		Preview: []*runner.Cmd{signCmd(c, preview)},
		Run: func(_ context.Context, env OpEnv) (string, error) {
			list, err := files()
			if err != nil {
				return "", err
			}
			_, err = env.Run(*signCmd(c, list))
			return "", err
		},
	}}
	if v := verifyOp(c, preview, what); v.Cmd != nil {
		ops = append(ops, Op{Desc: v.Desc, Optional: true, Preview: []*runner.Cmd{v.Cmd},
			Run: func(_ context.Context, env OpEnv) (string, error) {
				list, err := files()
				if err != nil {
					return "", err
				}
				_, err = env.Run(*verifyCmd(c, list))
				return "", err
			}})
	}
	return ops
}

// matchFiles lists the files below dir whose slash-separated relative path
// matches one of patterns; a pattern without "/" matches the file name in
// any folder (path.Match syntax).
func matchFiles(dir string, patterns []string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		for _, pat := range patterns {
			pat = strings.TrimPrefix(filepath.ToSlash(pat), "./")
			name := rel
			if !strings.Contains(pat, "/") {
				name = path.Base(rel)
			}
			if ok, _ := path.Match(strings.ToLower(pat), strings.ToLower(name)); ok {
				out = append(out, p)
				break
			}
		}
		return nil
	})
	return out, err
}

// signPreflight checks the signing configuration before anything is built.
func signPreflight(c *Context) []Issue {
	if !winSigning(c) || c.Host.OS != "windows" {
		return nil
	}
	s := c.Config.Windows.Sign
	var out []Issue
	if signtool(c) == "" {
		out = append(out, fatal(i18n.S("windows.sign is set but signtool.exe was not found", "设置了 windows.sign，但找不到 signtool.exe"),
			i18n.S("install the Windows SDK: winget install Microsoft.WindowsSDK.10.0.26100 (or Visual Studio Installer → Individual components → Windows SDK), or set windows.sign.signtool",
				"安装 Windows SDK：winget install Microsoft.WindowsSDK.10.0.26100（或 Visual Studio Installer → 单个组件 → Windows SDK），或设置 windows.sign.signtool")))
	}
	if s.Certificate != "" && s.Dlib == "" {
		if _, err := os.Stat(c.Project.Abs(s.Certificate)); err != nil {
			out = append(out, fatal(i18n.F("windows.sign.certificate %s not found", "找不到 windows.sign.certificate %s", s.Certificate),
				i18n.S("set the path of the .pfx file (FPACK_WINDOWS_CERTIFICATE)", "设置 .pfx 文件路径（FPACK_WINDOWS_CERTIFICATE）")))
		}
	}
	if s.Dlib != "" && s.DlibMetadata == "" {
		out = append(out, fatal(i18n.S("windows.sign.dlib needs windows.sign.dlib_metadata (the signing service's metadata.json)", "windows.sign.dlib 需要同时设置 windows.sign.dlib_metadata（签名服务的 metadata.json）"), ""))
	}
	return out
}

// quoteArg quotes a value containing spaces for a command line parsed with
// the usual rules (msix:create's --signtool-options).
func quoteArg(s string) string {
	if strings.ContainsAny(s, " \t\"") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
