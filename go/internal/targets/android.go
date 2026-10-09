package targets

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/runner"
)

// ---------------------------------------------------------------- signing --

// AndroidSigning is the resolved release signing configuration. fpack never
// edits Gradle files: it injects the Android Gradle Plugin's standard
// `android.injected.signing.*` properties through ORG_GRADLE_PROJECT_*
// environment variables, so secrets never appear on a command line.
type AndroidSigning struct {
	Enabled       bool
	StoreFile     string // absolute
	StorePassword string
	KeyAlias      string
	KeyPassword   string
	base64        string // keystore content from FPACK_ANDROID_KEYSTORE_BASE64
	written       bool
}

// ResolveAndroidSigning combines config (which already includes env and
// flag overrides) with FPACK_ANDROID_KEYSTORE_BASE64.
func ResolveAndroidSigning(cfg *config.Config, root, workDir string, getenv func(string) string) (AndroidSigning, error) {
	s := cfg.Android.Signing
	out := AndroidSigning{StorePassword: s.StorePassword, KeyAlias: s.KeyAlias, KeyPassword: s.KeyPassword}
	if b := strings.TrimSpace(getenv("FPACK_ANDROID_KEYSTORE_BASE64")); b != "" && s.StoreFile == "" {
		out.base64 = b
		out.StoreFile = filepath.Join(workDir, "secrets", "upload-keystore.jks")
	} else if s.StoreFile != "" {
		p := s.StoreFile
		if strings.HasPrefix(p, "~") {
			if h, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(h, p[1:])
			}
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		out.StoreFile = p
	}
	if out.StoreFile == "" {
		if s.StorePassword != "" || s.KeyAlias != "" {
			return out, fmt.Errorf("%s", i18n.S("android signing: a password/alias is set but no keystore (android.signing.store_file or FPACK_ANDROID_KEYSTORE)", "Android 签名：设置了密码/别名，但没有指定 keystore（android.signing.store_file 或 FPACK_ANDROID_KEYSTORE）"))
		}
		return out, nil
	}
	if out.KeyPassword == "" {
		out.KeyPassword = out.StorePassword
	}
	var missing []string
	if out.StorePassword == "" {
		missing = append(missing, "store_password / FPACK_ANDROID_KEYSTORE_PASSWORD")
	}
	if out.KeyAlias == "" {
		missing = append(missing, "key_alias / FPACK_ANDROID_KEY_ALIAS")
	}
	if len(missing) > 0 {
		return out, fmt.Errorf("%s%s", i18n.S("android signing is incomplete, missing: ", "Android 签名配置不完整，缺少："), strings.Join(missing, ", "))
	}
	out.Enabled = true
	return out, nil
}

// Materialize writes a base64 keystore to disk (mode 0600).
func (s *AndroidSigning) Materialize() error {
	if s.base64 == "" || s.written {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s.base64), ""))
	if err != nil {
		return fmt.Errorf("FPACK_ANDROID_KEYSTORE_BASE64 is not valid base64: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.StoreFile), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(s.StoreFile, data, 0o600); err != nil {
		return err
	}
	s.written = true
	return nil
}

// Cleanup removes a materialized keystore.
func (s *AndroidSigning) Cleanup() {
	if s.written {
		os.Remove(s.StoreFile)
		s.written = false
	}
}

// FromBase64 reports whether the keystore comes from an env variable.
func (s AndroidSigning) FromBase64() bool { return s.base64 != "" }

const injected = "ORG_GRADLE_PROJECT_android.injected.signing."

// GradleEnv returns env vars that make AGP sign release builds.
func (s AndroidSigning) GradleEnv() (env, secrets []string) {
	if !s.Enabled {
		return nil, nil
	}
	env = []string{
		injected + "store.file=" + s.StoreFile,
		injected + "store.password=" + s.StorePassword,
		injected + "key.alias=" + s.KeyAlias,
		injected + "key.password=" + s.KeyPassword,
	}
	return env, []string{s.StorePassword, s.KeyPassword}
}

func androidPreflight(c *Context) []Issue {
	var out []Issue
	s := &c.Signing
	switch {
	case s.Enabled && !s.FromBase64() && !exists(s.StoreFile):
		out = append(out, fatal(i18n.F("keystore not found: %s", "找不到 keystore：%s", s.StoreFile),
			i18n.S("fix android.signing.store_file / FPACK_ANDROID_KEYSTORE, or create one:\n", "请修正 android.signing.store_file / FPACK_ANDROID_KEYSTORE，或新建一个：\n")+
				"keytool -genkey -v -keystore ~/upload-keystore.jks -keyalg RSA -keysize 2048 -validity 10000 -alias upload"))
	case s.Enabled && !s.FromBase64() && !c.DryRun:
		if kt := c.Android().Keytool(c.Tools); kt != "" {
			out = append(out, verifyKeystore(c, kt)...)
		}
	case debugSigningExpected(c):
		out = append(out, warn(i18n.F("release builds are signed with the DEBUG key (%s). Fine for testing; Google Play will reject it.", "release 包将使用 DEBUG 密钥签名（%s）。测试可以，但 Google Play 会拒绝。", c.Project.AndroidGradleFile),
			i18n.S("set android.signing in fpack.yaml or FPACK_ANDROID_KEYSTORE / _PASSWORD / _KEY_ALIAS (see README.md → Android signing)", "在 fpack.yaml 设置 android.signing，或设置 FPACK_ANDROID_KEYSTORE / _PASSWORD / _KEY_ALIAS 环境变量（见 README.ZH.md → Android 签名）")))
	}
	if is, ok := flavorCheck(c, host.Android); !ok {
		out = append(out, is)
	}
	env := c.Android()
	if env.SDK == "" {
		out = append(out, warn(i18n.S("Android SDK not detected (Flutter may still find it)", "未检测到 Android SDK（Flutter 也许仍能找到）"),
			"flutter config --android-sdk <path>   # "+i18n.S("or install Android Studio", "或安装 Android Studio")))
	} else if !env.Licenses {
		out = append(out, warn(i18n.S("Android SDK licenses not accepted", "尚未接受 Android SDK 许可"), "flutter doctor --android-licenses"))
	}
	if env.Java == "" {
		is := warn(i18n.S("no Java (JDK 17+) found", "未找到 Java（需要 JDK 17+）"), i18n.S("install Android Studio (bundles a JDK) or a JDK 17/21, then: flutter config --jdk-dir <path>", "安装 Android Studio（自带 JDK）或 JDK 17/21，然后：flutter config --jdk-dir <路径>"))
		is.NotReady = true
		out = append(out, is)
	} else if env.JavaVersion > 0 && env.JavaVersion < 17 {
		is := warn(i18n.F("Java %d is too old for current Android Gradle Plugin (needs 17+)", "Java %d 版本过低，当前 Android Gradle 插件需要 17+", env.JavaVersion), "flutter config --jdk-dir <JDK 17/21>")
		is.NotReady = true
		out = append(out, is)
	}
	return out
}

// debugSigningExpected reports whether the project's release build type uses
// the debug signing config and fpack injects nothing.
func debugSigningExpected(c *Context) bool {
	return !c.Signing.Enabled && c.Project.AndroidReleaseDebugSigned && !c.Project.AndroidKeyProperties && c.Mode() == "release"
}

// keytoolEnglish forces keytool's messages to English: the JVM localizes
// them from the system locale (e.g. "所有者:" instead of "Owner:" on a
// Chinese macOS), which would break output parsing.
func keytoolEnglish() []string {
	return []string{"-J-Duser.language=en", "-J-Duser.country=US"}
}

func verifyKeystore(c *Context, keytool string) []Issue {
	s := c.Signing
	out, ok := c.Tools.ProbeEnv([]string{"FPACK_KS_PASS=" + s.StorePassword}, keytool, append(keytoolEnglish(), "-list", "-keystore", s.StoreFile, "-storepass:env", "FPACK_KS_PASS", "-alias", s.KeyAlias)...)
	out = runner.Redact(out, []string{s.StorePassword, s.KeyPassword})
	if ok {
		return nil
	}
	low := strings.ToLower(out)
	switch {
	case strings.Contains(low, "password was incorrect") || strings.Contains(low, "password verification failed"):
		return []Issue{fatal(i18n.S("keystore password is incorrect", "keystore 密码错误"), i18n.S("check FPACK_ANDROID_KEYSTORE_PASSWORD / android.signing.store_password", "请检查 FPACK_ANDROID_KEYSTORE_PASSWORD / android.signing.store_password"))}
	case strings.Contains(low, "does not exist"):
		return []Issue{fatal(i18n.F("key alias %q not found in the keystore", "keystore 中不存在别名 %q", s.KeyAlias), "keytool -list -keystore "+runner.Quote(s.StoreFile))}
	}
	return []Issue{warn(i18n.S("could not verify the keystore with keytool: ", "无法用 keytool 校验 keystore：")+firstLine(out), "")}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// --------------------------------------------------------------------- apk --

// APK builds Android APKs: universal, per-ABI, or both.
type APK struct{}

func (*APK) Name() string            { return "apk" }
func (*APK) Platform() host.Platform { return host.Android }
func (*APK) Formats() []string       { return []string{".apk"} }
func (*APK) Optional() bool          { return false }
func (*APK) Description() string {
	return i18n.S("Android APK (universal and/or per-ABI)", "Android APK（通用包和/或按 ABI 拆分）")
}
func (*APK) Preflight(c *Context) []Issue {
	out := androidPreflight(c)
	if apkSchemesSet(c) {
		switch {
		case !c.Signing.Enabled:
			out = append(out, warn(i18n.S("android.signing.v1-v4 are ignored: no keystore is configured (android.signing.store_file)", "android.signing.v1-v4 被忽略：未配置 keystore（android.signing.store_file）"), ""))
		case c.Android().Apksigner() == "":
			out = append(out, fatal(i18n.S("android.signing.v1-v4 need apksigner (Android SDK build-tools)", "android.signing.v1-v4 需要 apksigner（Android SDK build-tools）"),
				"sdkmanager \"build-tools;36.0.0\""))
		}
	}
	return out
}

// apkSchemesSet reports whether any signature scheme is configured.
func apkSchemesSet(c *Context) bool {
	s := c.Config.Android.Signing
	return s.V1 != nil || s.V2 != nil || s.V3 != nil || s.V4 != nil
}

// resignOp re-signs an APK with apksigner and the configured schemes.
func resignOp(c *Context, signer, src, dst string) Op {
	s := c.Signing
	sc := c.Config.Android.Signing
	args := []string{"sign", "--ks", s.StoreFile, "--ks-pass", "env:FPACK_KS_PASS", "--ks-key-alias", s.KeyAlias, "--key-pass", "env:FPACK_KEY_PASS"}
	for i, b := range []*bool{sc.V1, sc.V2, sc.V3, sc.V4} {
		if b != nil {
			args = append(args, fmt.Sprintf("--v%d-signing-enabled", i+1), fmt.Sprint(*b))
		}
	}
	args = append(args, "--out", dst, src)
	return Op{Desc: i18n.F("sign %s with apksigner (%s)", "用 apksigner 签名 %s（%s）", filepath.Base(dst), schemeLabel(sc)),
		Cmd:  &runner.Cmd{Name: signer, Args: args, Env: []string{"FPACK_KS_PASS=" + s.StorePassword, "FPACK_KEY_PASS=" + s.KeyPassword}, Secret: []string{s.StorePassword, s.KeyPassword}},
		Hint: i18n.S("check android.signing (keystore, alias, passwords)", "请检查 android.signing（keystore、别名、密码）")}
}

func schemeLabel(sc config.AndroidSigning) string {
	var on []string
	for i, b := range []*bool{sc.V1, sc.V2, sc.V3, sc.V4} {
		if b != nil && *b {
			on = append(on, fmt.Sprintf("v%d", i+1))
		} else if b != nil {
			on = append(on, fmt.Sprintf("-v%d", i+1))
		}
	}
	return strings.Join(on, " ")
}

var abiToPlatform = map[string]string{"armeabi-v7a": "android-arm", "arm64-v8a": "android-arm64", "x86_64": "android-x64"}

func androidArgs(c *Context, sub string) ([]string, []string) {
	args, warns := CommonArgs(c, host.Android, sub)
	if len(c.Config.Android.ABIs) > 0 {
		var plats []string
		for _, a := range c.Config.Android.ABIs {
			plats = append(plats, abiToPlatform[a])
		}
		args = append(args, "--target-platform", strings.Join(plats, ","))
	}
	for _, k := range sortedKeys(c.Config.Android.ProjectArgs) {
		args = append(args, "-P", k+"="+c.Config.Android.ProjectArgs[k])
	}
	return args, warns
}

func (*APK) Steps(c *Context) ([]FlutterStep, error) {
	env, secrets := c.Signing.GradleEnv()
	mode := c.Config.SplitPerABI()
	var steps []FlutterStep
	if mode == config.ABIUniversal || mode == config.ABIBoth {
		args, w := androidArgs(c, "apk")
		args = append(args, tailArgs(c, c.Config.Android.ExtraArgs)...)
		steps = append(steps, FlutterStep{Key: "apk", Platform: host.Android, Args: args, Env: env, Secret: secrets, Warnings: w})
	}
	if mode == config.ABISplit || mode == config.ABIBoth {
		args, w := androidArgs(c, "apk")
		args = append(args, "--split-per-abi")
		args = append(args, tailArgs(c, c.Config.Android.ExtraArgs)...)
		steps = append(steps, FlutterStep{Key: "apk-split", Platform: host.Android, Args: args, Env: env, Secret: secrets, Warnings: w})
	}
	return steps, nil
}

// APKFileName returns Flutter's output file name, e.g. app-release.apk,
// app-prod-release.apk, app-arm64-v8a-prod-release.apk (flavor lowercased).
func APKFileName(abi, flavor, mode string) string {
	parts := []string{"app"}
	if abi != "" {
		parts = append(parts, abi)
	}
	if flavor != "" {
		parts = append(parts, strings.ToLower(flavor))
	}
	parts = append(parts, mode)
	return strings.Join(parts, "-") + ".apk"
}

func (*APK) Locate(c *Context, predicted bool, since time.Time) (Inputs, error) {
	dir := filepath.Join(c.Project.Root, "build", "app", "outputs", "flutter-apk")
	in := Inputs{}
	mode := c.Config.SplitPerABI()
	var abis []string
	if mode == config.ABIUniversal || mode == config.ABIBoth {
		abis = append(abis, "")
	}
	if mode == config.ABISplit || mode == config.ABIBoth {
		abis = append(abis, c.Config.ABIs()...)
	}
	for _, abi := range abis {
		key := abi
		if key == "" {
			key = "universal"
		}
		want := filepath.Join(dir, APKFileName(abi, c.Flavor(), c.Mode()))
		if predicted {
			in[key] = want
			continue
		}
		// The exact path is deterministic for this flavor/mode/ABI. It may
		// be older than this run when Gradle found the task up to date.
		if exists(want) {
			in[key] = want
			continue
		}
		// Fallback for naming changes across Flutter versions.
		fl := strings.ToLower(c.Flavor())
		p := findNewest(filepath.Join(dir, "*.apk"), since, func(n string) bool {
			n = strings.ToLower(n)
			if abi == "" {
				for _, a := range config.KnownABIs {
					if strings.Contains(n, a) {
						return false
					}
				}
			} else if !strings.Contains(n, abi) {
				return false
			}
			return strings.Contains(n, c.Mode()) && (fl == "" || strings.Contains(n, fl))
		})
		if p == "" {
			return nil, notFound(filepath.Base(want), c.Rel(dir))
		}
		in[key] = p
	}
	return in, nil
}

func (t *APK) Package(c *Context, in Inputs) (*Plan, error) {
	pl := &Plan{}
	order := append([]string{"universal"}, c.Config.ABIs()...)
	first := ""
	for _, key := range order {
		src, ok := in[key]
		if !ok {
			continue
		}
		dst, err := c.ArtifactPath(host.Android, key, "", ".apk")
		if err != nil {
			return nil, err
		}
		kind := "APK (" + key + ")"
		if signer := c.Android().Apksigner(); apkSchemesSet(c) && c.Signing.Enabled && signer != "" {
			// Sign into the staging directory, then move: a failed signature
			// never leaves a partial file in the output directory.
			stage := c.Stage("apk")
			tmp := filepath.Join(stage, filepath.Base(dst))
			if len(pl.Ops) == 0 {
				pl.Ops = append(pl.Ops, resetDirOp(c, stage))
			}
			pl.Ops = append(pl.Ops, resignOp(c, signer, src, tmp), moveOp(c, tmp, dst))
			pl.Artifacts = append(pl.Artifacts, Artifact{Path: dst, Kind: kind, Arch: key})
			if v4 := c.Config.Android.Signing.V4; v4 != nil && *v4 {
				pl.Ops = append(pl.Ops, moveOp(c, tmp+".idsig", dst+".idsig"))
				pl.Artifacts = append(pl.Artifacts, Artifact{Path: dst + ".idsig", Kind: "APK v4 signature (" + key + ")", Arch: key})
			}
		} else {
			pl.Ops = append(pl.Ops, copyOp(c, src, dst))
			pl.Artifacts = append(pl.Artifacts, Artifact{Path: dst, Kind: kind, Arch: key})
		}
		if first == "" {
			first = dst
		}
	}
	if first != "" {
		if signer := c.Android().Apksigner(); signer != "" {
			args := []string{"verify", "--verbose", "--print-certs"}
			if resigned := apkSchemesSet(c) && c.Signing.Enabled; resigned {
				sc := c.Config.Android.Signing
				// apksigner skips v1 for minSdk >= 24 and only checks v4
				// with the .idsig file: verify what was configured.
				if sc.V1 != nil && *sc.V1 {
					args = append(args, "--min-sdk-version", "21") // Flutter's minimum
				}
				if sc.V4 != nil && *sc.V4 {
					args = append(args, "--v4-signature-file", first+".idsig")
				}
			}
			args = append(args, first)
			pl.Ops = append(pl.Ops, Op{Desc: i18n.S("verify APK signature", "校验 APK 签名"), Cmd: &runner.Cmd{Name: signer, Args: args, Capture: true},
				Optional: true, Check: signerCheck(c, `(?m)certificate DN: (.+)$`)})
		}
	}
	return pl, nil
}

// signerCheck turns apksigner/keytool output into a note, flagging debug keys.
func signerCheck(c *Context, pattern string) func(runner.Result) (string, error) {
	injected := c.Signing.Enabled
	expected := debugSigningExpected(c) || c.Mode() != "release"
	re := regexp.MustCompile(pattern)
	return func(r runner.Result) (string, error) {
		m := re.FindStringSubmatch(r.Output)
		if m == nil {
			return "", nil
		}
		dn := strings.TrimSpace(m[1])
		schemes := ""
		for _, v := range regexp.MustCompile(`(?m)^Verified using (v[0-9.]+) scheme[^:]*: true`).FindAllStringSubmatch(r.Output, -1) {
			schemes += " " + v[1]
		}
		if strings.Contains(dn, "CN=Android Debug") {
			if injected {
				return "WARN:" + i18n.S("signing was configured but the output is signed with the DEBUG key – the Gradle signingConfigs may override injected signing", "已配置签名，但产物仍使用 DEBUG 密钥签名 —— Gradle 的 signingConfigs 可能覆盖了注入的签名"), nil
			}
			if expected {
				return i18n.S("signed with the Android debug key", "使用 Android debug 密钥签名"), nil
			}
			return "WARN:" + i18n.S("signed with the Android DEBUG key – not accepted by Google Play", "使用 Android DEBUG 密钥签名 —— Google Play 不接受"), nil
		}
		if schemes != "" {
			return i18n.F("signed by %s (schemes:%s)", "签名者：%s（签名方案：%s）", dn, schemes), nil
		}
		return i18n.F("signed by %s", "签名者：%s", dn), nil
	}
}

// --------------------------------------------------------------------- aab --

// AAB builds an Android App Bundle for Google Play.
type AAB struct{}

func (*AAB) Name() string            { return "aab" }
func (*AAB) Platform() host.Platform { return host.Android }
func (*AAB) Formats() []string       { return []string{".aab"} }
func (*AAB) Optional() bool          { return false }
func (*AAB) Description() string {
	return i18n.S("Android App Bundle for Google Play", "Android App Bundle（用于 Google Play）")
}
func (*AAB) Preflight(c *Context) []Issue { return androidPreflight(c) }

func (*AAB) Steps(c *Context) ([]FlutterStep, error) {
	env, secrets := c.Signing.GradleEnv()
	args, w := androidArgs(c, "appbundle")
	args = append(args, tailArgs(c, c.Config.Android.ExtraArgs)...)
	return []FlutterStep{{Key: "aab", Platform: host.Android, Args: args, Env: env, Secret: secrets, Warnings: w}}, nil
}

// AABPath returns Flutter's bundle path relative to the project, e.g.
// build/app/outputs/bundle/prodRelease/app-prod-release.aab.
func AABPath(flavor, mode string) string {
	dir := mode
	file := "app-" + mode + ".aab"
	if flavor != "" {
		dir = flavor + ModeCap(mode)
		file = "app-" + flavor + "-" + mode + ".aab"
	}
	return filepath.Join("build", "app", "outputs", "bundle", dir, file)
}

func (*AAB) Locate(c *Context, predicted bool, since time.Time) (Inputs, error) {
	want := filepath.Join(c.Project.Root, AABPath(c.Flavor(), c.Mode()))
	if predicted {
		return Inputs{"aab": want}, nil
	}
	// Gradle leaves an up-to-date bundle untouched, so the exact path is
	// accepted regardless of its age (flutter just succeeded).
	if exists(want) {
		return Inputs{"aab": want}, nil
	}
	fl := strings.ToLower(c.Flavor())
	p := findNewest(filepath.Join(c.Project.Root, "build", "app", "outputs", "bundle", "*", "*.aab"), since, func(n string) bool {
		n = strings.ToLower(n)
		return strings.Contains(n, c.Mode()) && (fl == "" || strings.Contains(n, fl))
	})
	if p == "" {
		return nil, notFound(filepath.Base(want), "build/app/outputs/bundle")
	}
	return Inputs{"aab": p}, nil
}

func (*AAB) Package(c *Context, in Inputs) (*Plan, error) {
	dst, err := c.ArtifactPath(host.Android, "", "", ".aab")
	if err != nil {
		return nil, err
	}
	pl := &Plan{Ops: []Op{copyOp(c, in["aab"], dst)}, Artifacts: []Artifact{{Path: dst, Kind: "AAB"}}}
	if kt := c.Android().Keytool(c.Tools); kt != "" {
		pl.Ops = append(pl.Ops, Op{Desc: i18n.S("verify AAB signature", "校验 AAB 签名"), Cmd: &runner.Cmd{Name: kt, Args: append(keytoolEnglish(), "-printcert", "-jarfile", dst), Capture: true},
			Optional: true, Check: signerCheck(c, `(?m)^(?:Owner|所有者): (.+)$`)})
	}
	return pl, nil
}
