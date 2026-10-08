package targets

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/project"
)

// fakeTools resolves only the listed executables and returns canned probe output.
type fakeTools struct {
	bins   map[string]string
	probes map[string]string // "name arg1 arg2" -> output
	fail   map[string]bool
}

func (f *fakeTools) Find(n string) string { return f.bins[n] }
func (f *fakeTools) Probe(n string, a ...string) (string, bool) {
	k := strings.TrimSpace(n + " " + strings.Join(a, " "))
	return f.probes[k], !f.fail[k]
}
func (f *fakeTools) ProbeEnv(_ []string, n string, a ...string) (string, bool) {
	return f.Probe(n, a...)
}

const devID = "Developer ID Application: XueHua Tech (ABCDE12345)"

func identities() string {
	return `  1) 1111111111111111111111111111111111111111 "Apple Development: dev@x.com (ZZZ)"
  2) 2222222222222222222222222222222222222222 "` + devID + `"
  3) 3333333333333333333333333333333333333333 "Apple Distribution: XueHua Tech (ABCDE12345)"
     3 valid identities found`
}

func newCtx(t *testing.T, goos, cfgYAML, pubspecExtra string) *Context {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	write("pubspec.yaml", "name: xue_hua_im\nversion: 1.0.0+1\ndependencies:\n  flutter:\n    sdk: flutter\n"+pubspecExtra)
	write("android/app/build.gradle.kts", `android { buildTypes { release { signingConfig = signingConfigs.getByName("debug") } } }`)
	write("ios/Podfile", "")
	write("ios/Runner.xcodeproj/project.pbxproj", "DEVELOPMENT_TEAM = ABCDE12345;\n")
	write("macos/Podfile", "")
	write("macos/Runner/Configs/AppInfo.xcconfig", "PRODUCT_NAME = XueHua\nPRODUCT_BUNDLE_IDENTIFIER = com.xuehua.im\n")
	write("macos/Runner/Release.entitlements", "<plist/>")
	write("linux/CMakeLists.txt", `set(BINARY_NAME "xue_hua_im")`)
	write("web/icons/Icon-512.png", "png")
	os.MkdirAll(filepath.Join(root, "windows"), 0o755)
	p, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	if err := config.Parse([]byte(cfgYAML), cfg); err != nil {
		t.Fatal(err)
	}
	tools := &fakeTools{
		bins:   map[string]string{"xcodebuild": "/usr/bin/xcodebuild", "pod": "/opt/homebrew/bin/pod", "dpkg-deb": "/usr/bin/dpkg-deb", "clang++": "/usr/bin/clang++", "cmake": "/usr/bin/cmake", "ninja": "/usr/bin/ninja", "pkg-config": "/usr/bin/pkg-config"},
		probes: map[string]string{"security find-identity -v -p codesigning": identities()},
		fail:   map[string]bool{},
	}
	c := &Context{
		Project: p, Config: cfg, SDK: &flutter.SDK{Root: "/sdk", Flutter: "/sdk/bin/flutter", Dart: "/sdk/bin/dart"},
		Host: host.Host{OS: goos, Arch: "arm64"}, Tools: tools,
		AppName: "xue_hua_im", BuildName: "1.0.0", BuildNumber: "1",
		OutDir: filepath.Join(root, "dist", "1.0.0+1"), WorkDir: filepath.Join(root, "build", "fpack"), DryRun: true,
	}
	c.SetAndroidEnv(&AndroidEnv{SDK: "/android", Licenses: true, Java: "/jdk/bin/java", JavaVersion: 21})
	c.Mac, err = ResolveMacSigning(p, cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.Signing, err = ResolveAndroidSigning(cfg, root, c.WorkDir, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func steps(t *testing.T, tg Target, c *Context) []FlutterStep {
	t.Helper()
	s, err := tg.Steps(c)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func plan(t *testing.T, tg Target, c *Context) *Plan {
	t.Helper()
	in, err := tg.Locate(c, true, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := tg.Package(c, in)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func cmds(p *Plan) []string {
	var out []string
	for _, o := range p.Ops {
		for _, c := range o.Commands() {
			out = append(out, c.String())
		}
	}
	return out
}

func names(p *Plan) []string {
	var out []string
	for _, a := range p.Artifacts {
		out = append(out, filepath.Base(a.Path))
	}
	return out
}

// ------------------------------------------------------------------ android

func TestAPKUniversalDefault(t *testing.T) {
	c := newCtx(t, "linux", "", "")
	s := steps(t, &APK{}, c)
	if len(s) != 1 || strings.Join(s[0].Args, " ") != "build apk --release" || s[0].Env != nil {
		t.Fatalf("steps: %+v", s)
	}
	p := plan(t, &APK{}, c)
	if !reflect.DeepEqual(names(p), []string{"xue_hua_im-1.0.0+1-android-universal.apk"}) {
		t.Fatal(names(p))
	}
}

func TestAPKBothWithFlavorAndOptions(t *testing.T) {
	c := newCtx(t, "darwin", `
build:
  flavor: paidPro
  target: lib/main_prod.dart
  build_name: 2.0.0
  build_number: 37
  obfuscate: true
  dart_define: {API: https://x}
  dart_define_from_file: env/prod.json
android:
  split_per_abi: both
  abis: [arm64-v8a, armeabi-v7a]
`, "")
	c.BuildName, c.BuildNumber = "2.0.0", "37"
	c.PassArgs = []string{"--no-tree-shake-icons"}
	s := steps(t, &APK{}, c)
	if len(s) != 2 || s[0].Key != "apk" || s[1].Key != "apk-split" {
		t.Fatalf("keys: %+v", s)
	}
	want := "build apk --release --flavor paidPro --target lib/main_prod.dart --build-name 2.0.0 --build-number 37 --obfuscate --split-debug-info=" + filepath.FromSlash("dist/1.0.0+1/debug-info/android") + " --dart-define=API=https://x --dart-define-from-file=env/prod.json --target-platform android-arm64,android-arm"
	if got := strings.Join(s[0].Args, " "); got != want+" --no-tree-shake-icons" {
		t.Fatalf("universal args:\n got %s\nwant %s", got, want)
	}
	if got := strings.Join(s[1].Args, " "); got != want+" --split-per-abi --no-tree-shake-icons" {
		t.Fatalf("split args: %s", got)
	}
	in, _ := (&APK{}).Locate(c, true, time.Time{})
	if filepath.Base(in["arm64-v8a"]) != "app-arm64-v8a-paidpro-release.apk" || filepath.Base(in["universal"]) != "app-paidpro-release.apk" {
		t.Fatalf("flutter output names: %v", in)
	}
	got := names(plan(t, &APK{}, c))
	wantNames := []string{"xue_hua_im-paidPro-2.0.0+37-android-universal.apk", "xue_hua_im-paidPro-2.0.0+37-android-arm64-v8a.apk", "xue_hua_im-paidPro-2.0.0+37-android-armeabi-v7a.apk"}
	if !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("artifacts: %v", got)
	}
}

func TestAPKSplitDefaultABIs(t *testing.T) {
	c := newCtx(t, "linux", "android:\n  split_per_abi: true\n", "")
	s := steps(t, &APK{}, c)
	if len(s) != 1 || !strings.HasSuffix(strings.Join(s[0].Args, " "), "--split-per-abi") || strings.Contains(strings.Join(s[0].Args, " "), "--target-platform") {
		t.Fatalf("%v", s[0].Args)
	}
	got := names(plan(t, &APK{}, c))
	if len(got) != 3 || got[0] != "xue_hua_im-1.0.0+1-android-armeabi-v7a.apk" || got[2] != "xue_hua_im-1.0.0+1-android-x86_64.apk" {
		t.Fatal(got)
	}
}

func TestAPKFileName(t *testing.T) {
	cases := map[[3]string]string{
		{"", "", "release"}:              "app-release.apk",
		{"", "Prod", "release"}:          "app-prod-release.apk",
		{"arm64-v8a", "", "release"}:     "app-arm64-v8a-release.apk",
		{"x86_64", "paidPro", "profile"}: "app-x86_64-paidpro-profile.apk",
	}
	for in, want := range cases {
		if got := APKFileName(in[0], in[1], in[2]); got != want {
			t.Errorf("%v: %s", in, got)
		}
	}
	if AABPath("paidPro", "release") != filepath.Join("build/app/outputs/bundle/paidProRelease/app-paidPro-release.aab") || AABPath("", "release") != filepath.Join("build/app/outputs/bundle/release/app-release.aab") {
		t.Fatal("aab path")
	}
}

func TestAndroidSigningInjectionAndRedaction(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "linux", "android:\n  signing:\n    store_file: keys/upload.jks\n    store_password: s3cr3t-pw\n    key_alias: upload\n", "")
	if !c.Signing.Enabled || c.Signing.KeyPassword != "s3cr3t-pw" || !filepath.IsAbs(c.Signing.StoreFile) {
		t.Fatalf("%+v", c.Signing)
	}
	s := steps(t, &AAB{}, c)
	if strings.Join(s[0].Args, " ") != "build appbundle --release" {
		t.Fatal(s[0].Args)
	}
	env := strings.Join(s[0].Env, "\n")
	for _, k := range []string{"store.file=", "store.password=s3cr3t-pw", "key.alias=upload", "key.password=s3cr3t-pw"} {
		if !strings.Contains(env, "ORG_GRADLE_PROJECT_android.injected.signing."+k) {
			t.Errorf("missing %s in %s", k, env)
		}
	}
	if !reflect.DeepEqual(s[0].Secret, []string{"s3cr3t-pw", "s3cr3t-pw"}) {
		t.Fatal("secrets")
	}
	// keystore file does not exist -> fatal preflight
	issues := (&AAB{}).Preflight(c)
	if len(issues) == 0 || !issues[0].Fatal || !strings.Contains(issues[0].Msg, "upload.jks") {
		t.Fatalf("%+v", issues)
	}
	if got := names(plan(t, &AAB{}, c)); got[0] != "xue_hua_im-1.0.0+1-android.aab" {
		t.Fatal(got)
	}
}

func TestAndroidSigningIncomplete(t *testing.T) {
	cfg := &config.Config{}
	config.Parse([]byte("android:\n  signing:\n    store_file: a.jks\n"), cfg)
	if _, err := ResolveAndroidSigning(cfg, "/p", "/p/build/fpack", func(string) string { return "" }); err == nil || !strings.Contains(err.Error(), "FPACK_ANDROID_KEYSTORE_PASSWORD") {
		t.Fatalf("%v", err)
	}
	cfg = &config.Config{}
	s, err := ResolveAndroidSigning(cfg, "/p", "/w", func(k string) string {
		return map[string]string{"FPACK_ANDROID_KEYSTORE_BASE64": "aGVsbG8="}[k]
	})
	if err == nil || s.Enabled || !s.FromBase64() {
		t.Fatalf("base64 keystore without password must be reported as incomplete: %+v %v", s, err)
	}
}

func TestAndroidDebugSigningWarning(t *testing.T) {
	c := newCtx(t, "linux", "", "")
	issues := (&APK{}).Preflight(c)
	if len(issues) != 1 || issues[0].Fatal || !strings.Contains(issues[0].Msg, "DEBUG") {
		t.Fatalf("%+v", issues)
	}
}

func TestSignerCheck(t *testing.T) {
	debug := runnerResult("Signer #1 certificate DN: C=US, O=Android, CN=Android Debug\n")
	c := newCtx(t, "linux", "", "")
	// Project signs release with the debug key and fpack injects nothing:
	// preflight already warned, so the post-build check is only a note.
	chk := signerCheck(c, `(?m)certificate DN: (.+)$`)
	if note, _ := chk(debug); strings.HasPrefix(note, "WARN:") || !strings.Contains(note, "debug") {
		t.Fatal(note)
	}
	if note, _ := chk(runnerResult("Signer #1 certificate DN: CN=XueHua, O=XueHua\n")); !strings.Contains(note, "CN=XueHua") {
		t.Fatal(note)
	}
	// Unexpected debug signature -> warning.
	c.Project.AndroidReleaseDebugSigned = false
	if note, _ := signerCheck(c, `(?m)certificate DN: (.+)$`)(debug); !strings.HasPrefix(note, "WARN:") {
		t.Fatal(note)
	}
	// Injected signing but still debug -> specific warning.
	c.Signing.Enabled = true
	if note, _ := signerCheck(c, `(?m)certificate DN: (.+)$`)(debug); !strings.HasPrefix(note, "WARN:") || !strings.Contains(note, "signingConfigs") {
		t.Fatal(note)
	}
}

// ---------------------------------------------------------------------- iOS

func TestIPADefault(t *testing.T) {
	c := newCtx(t, "darwin", "", "")
	s := steps(t, &IPA{}, c)
	if strings.Join(s[0].Args, " ") != "build ipa --release" {
		t.Fatal(s[0].Args)
	}
	p := plan(t, &IPA{}, c)
	if names(p)[0] != "xue_hua_im-1.0.0+1-ios-arm64.ipa" || p.Artifacts[0].Kind != "IPA (app-store)" {
		t.Fatalf("%v %v", names(p), p.Artifacts)
	}
	if issues := (&IPA{}).Preflight(c); len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
}

func TestIPAExportMethodAndPlist(t *testing.T) {
	c := newCtx(t, "darwin", "ios:\n  export_method: ad-hoc\nbuild:\n  flavor: prod\n", "")
	if got := strings.Join(steps(t, &IPA{}, c)[0].Args, " "); got != "build ipa --release --flavor prod --export-method ad-hoc" {
		t.Fatal(got)
	}
	if names(plan(t, &IPA{}, c))[0] != "xue_hua_im-prod-1.0.0+1-ios-arm64.ipa" {
		t.Fatal("flavored name")
	}
	c = newCtx(t, "darwin", "ios:\n  export_method: ad-hoc\n  export_options_plist: ios/ExportOptions.plist\n", "")
	if got := strings.Join(steps(t, &IPA{}, c)[0].Args, " "); got != "build ipa --release --export-options-plist ios/ExportOptions.plist" {
		t.Fatal(got)
	}
	issues := (&IPA{}).Preflight(c)
	if len(issues) < 2 || !issues[0].Fatal || !strings.Contains(issues[0].Msg, "ExportOptions.plist") || issues[1].Fatal {
		t.Fatalf("%+v", issues)
	}
}

func TestIPAUnsigned(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "darwin", "ios:\n  codesign: false\n", "")
	if got := strings.Join(steps(t, &IPA{}, c)[0].Args, " "); got != "build ipa --release --no-codesign" {
		t.Fatal(got)
	}
	p := plan(t, &IPA{}, c)
	if names(p)[0] != "xue_hua_im-1.0.0+1-ios-arm64-unsigned.ipa" {
		t.Fatal(names(p))
	}
	cs := cmds(p)
	if len(cs) != 2 || !strings.Contains(cs[0], "Runner.xcarchive/Products/Applications/Runner.app") || !strings.Contains(cs[0], "/Payload/Runner.app") ||
		!strings.HasPrefix(cs[1], "ditto -c -k --sequesterRsrc --keepParent ") || !strings.Contains(cs[1], "Payload") {
		t.Fatalf("%q", cs)
	}
}

func TestIPAPreflightMissingTools(t *testing.T) {
	c := newCtx(t, "darwin", "", "")
	c.Tools.(*fakeTools).bins = map[string]string{}
	issues := (&IPA{}).Preflight(c)
	if len(issues) < 2 || !issues[0].Fatal || !strings.Contains(issues[0].Msg, "xcodebuild") || !strings.Contains(issues[1].Fix, "brew install cocoapods") {
		t.Fatalf("%+v", issues)
	}
}

func TestIPANoTeamWarning(t *testing.T) {
	c := newCtx(t, "darwin", "", "")
	c.Project.IOSTeam = ""
	issues := (&IPA{}).Preflight(c)
	if len(issues) != 1 || issues[0].Fatal || !strings.Contains(issues[0].Msg, "DEVELOPMENT_TEAM") {
		t.Fatalf("%+v", issues)
	}
}

// -------------------------------------------------------------------- macOS

func TestMacAppUnsigned(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "darwin", "", "")
	if c.Mac.Enabled {
		t.Fatal("signing must be off without config")
	}
	if got := strings.Join(steps(t, &MacApp{}, c)[0].Args, " "); got != "build macos --release" {
		t.Fatal(got)
	}
	p := plan(t, &MacApp{}, c)
	cs := cmds(p)
	if names(p)[0] != "xue_hua_im-1.0.0+1-macos-universal.zip" || len(cs) != 1 ||
		!strings.Contains(cs[0], "build/macos/Build/Products/Release/XueHua.app") || !strings.HasPrefix(cs[0], "ditto -c -k --sequesterRsrc --keepParent") {
		t.Fatalf("%v %q", names(p), cs)
	}
}

func TestMacSigningFromConfig(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "darwin", "macos:\n  sign:\n    identity: \""+devID+"\"\n    notary_profile: XueHua\n", "")
	m := c.Mac
	if !m.Enabled || !m.Notarize || m.Identity != devID || m.Profile != "XueHua" || !m.Configured || m.TurnedOff {
		t.Fatalf("%+v", m)
	}
	if issues := (&DMG{}).Preflight(c); len(issues) != 0 {
		t.Fatalf("%+v", issues)
	}
	c.Tools.(*fakeTools).probes["xcrun --find notarytool"] = "/usr/bin/notarytool"
	p := plan(t, &DMG{}, c)
	cs := cmds(p)
	ent := filepath.Join(c.Project.Root, "macos", "Runner", "Release.entitlements")
	staged := filepath.Join(c.WorkDir, "stage", "dmg", "root", "XueHua.app")
	tmp := filepath.Join(c.WorkDir, "stage", "dmg", "xue_hua_im-1.0.0+1-macos-universal.dmg")
	q := func(s string) string { return "'" + s + "'" }
	want := []string{
		"ditto " + filepath.Join(c.Project.Root, "build/macos/Build/Products/Release/XueHua.app") + " " + staged,
		"codesign --force --deep --options runtime --timestamp --sign " + q(devID) + " " + staged,
		"codesign --force --options runtime --timestamp --entitlements " + ent + " --sign " + q(devID) + " " + staged,
		"codesign --verify --deep --strict --verbose=2 " + staged,
		"hdiutil create -volname XueHua -srcfolder " + filepath.Dir(staged) + " -ov -fs HFS+ -format UDZO " + tmp,
		"codesign --force --timestamp --sign " + q(devID) + " " + tmp,
		"xcrun notarytool submit " + tmp + " --keychain-profile XueHua --output-format json",
		"xcrun notarytool wait '<submission-id>' --keychain-profile XueHua --output-format json",
		"xcrun stapler staple " + tmp,
		"spctl --assess --type open --context context:primary-signature --verbose=2 " + tmp,
	}
	if !reflect.DeepEqual(cs, want) {
		t.Fatalf("dmg commands:\n%s\nwant:\n%s", strings.Join(cs, "\n"), strings.Join(want, "\n"))
	}
	if p.Artifacts[0].Kind != "DMG (signed, notarized)" || names(p)[0] != "xue_hua_im-1.0.0+1-macos-universal.dmg" || len(p.Notes) != 0 {
		t.Fatalf("%+v %v", p.Artifacts, p.Notes)
	}
}

// fpack never reads the pubspec `dmg:` section (it belongs to the dmg package).
func TestPubspecDMGSectionIsIgnored(t *testing.T) {
	c := newCtx(t, "darwin", "", "dmg:\n  sign-certificate: \""+devID+"\"\n  notary-profile: XueHua\n  sign: true\n  notarization: true\n")
	if c.Mac.Enabled || c.Mac.Notarize || c.Mac.Configured || c.Mac.Identity != "" {
		t.Fatalf("%+v", c.Mac)
	}
	p := plan(t, &DMG{}, c)
	if strings.Contains(strings.Join(cmds(p), "\n"), "codesign") {
		t.Fatal(cmds(p))
	}
	if n := strings.Join(p.Notes, "\n"); !strings.Contains(n, "macos.sign.identity") || strings.Contains(n, "pubspec") {
		t.Fatal(n)
	}
}
func TestMacZipSignedAndNotarized(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "darwin", "macos:\n  sign:\n    identity: \""+devID+"\"\n    notary_profile: XueHua\n", "")
	p := plan(t, &MacApp{}, c)
	cs := cmds(p)
	staged := filepath.Join(c.WorkDir, "stage", "macos", "XueHua.app")
	zip := filepath.Join(c.WorkDir, "stage", "macos", "xue_hua_im-1.0.0+1-macos-universal.zip")
	want := []string{
		"ditto " + filepath.Join(c.Project.Root, "build/macos/Build/Products/Release/XueHua.app") + " " + staged,
		"codesign --force --deep --options runtime --timestamp --sign '" + devID + "' " + staged,
		"codesign --force --options runtime --timestamp --entitlements " + filepath.Join(c.Project.Root, "macos/Runner/Release.entitlements") + " --sign '" + devID + "' " + staged,
		"codesign --verify --deep --strict --verbose=2 " + staged,
		"ditto -c -k --sequesterRsrc --keepParent " + staged + " " + zip,
		"xcrun notarytool submit " + zip + " --keychain-profile XueHua --output-format json",
		"xcrun notarytool wait '<submission-id>' --keychain-profile XueHua --output-format json",
		"xcrun stapler staple " + staged,
		"ditto -c -k --sequesterRsrc --keepParent " + staged + " " + zip,
	}
	if !reflect.DeepEqual(cs, want) {
		t.Fatalf("zip commands:\n%s\nwant:\n%s", strings.Join(cs, "\n"), strings.Join(want, "\n"))
	}
	if p.Artifacts[0].Kind != "macOS app (zip, signed, notarized)" {
		t.Fatal(p.Artifacts[0].Kind)
	}
	// --no-sign wins over everything
	c = newCtx(t, "darwin", "macos:\n  sign:\n    enabled: false\n    identity: \""+devID+"\"\n    notary_profile: XueHua\n", "")
	if c.Mac.Enabled || c.Mac.Notarize || !c.Mac.TurnedOff {
		t.Fatalf("%+v", c.Mac)
	}
}
func TestMacSigningOverrides(t *testing.T) {
	id := "    identity: \"" + devID + "\"\n    notary_profile: XueHua\n"
	// notarization off, signing stays on (--no-notarize)
	c := newCtx(t, "darwin", "macos:\n  sign:\n    notarize: false\n"+id, "")
	if !c.Mac.Enabled || c.Mac.Notarize {
		t.Fatalf("%+v", c.Mac)
	}
	for _, s := range cmds(plan(t, &DMG{}, c)) {
		if strings.Contains(s, "notarytool") {
			t.Fatal("notarize should be off")
		}
	}
	// disabling signing disables notarization and says so
	c = newCtx(t, "darwin", "macos:\n  sign:\n    enabled: false\n"+id, "")
	if c.Mac.Enabled || c.Mac.Notarize {
		t.Fatalf("%+v", c.Mac)
	}
	if n := strings.Join(plan(t, &DMG{}, c).Notes, "\n"); !strings.Contains(n, "(--no-sign)") {
		t.Fatal(n)
	}
	// nothing configured: unsigned, with a pointer to macos.sign
	if n := strings.Join(newCtxPlanNotes(t), "\n"); strings.Contains(n, "--no-sign") || !strings.Contains(n, "macos.sign.identity") {
		t.Fatal(n)
	}
	// notary profile alone means nothing to notarize without signing
	c = newCtx(t, "darwin", "macos:\n  sign:\n    notary_profile: XueHua\n", "")
	if c.Mac.Enabled || c.Mac.Notarize {
		t.Fatalf("%+v", c.Mac)
	}
	// conflicting explicit config is an error
	p, _ := project.Load(c.Project.Root)
	cfg := &config.Config{}
	config.Parse([]byte("macos:\n  sign:\n    enabled: false\n    notarize: true\n"), cfg)
	if _, err := ResolveMacSigning(p, cfg); err == nil {
		t.Fatal("expected conflict error")
	}
	// identity in fpack.yaml implies signing
	c = newCtx(t, "darwin", "macos:\n  sign:\n    identity: \"Developer ID Application: Other\"\n", "")
	if !c.Mac.Enabled || c.Mac.Notarize {
		t.Fatalf("%+v", c.Mac)
	}
	issues := (&DMG{}).Preflight(c)
	if len(issues) != 1 || !issues[0].Fatal || !strings.Contains(issues[0].Msg, "Other") || !strings.Contains(issues[0].Msg, devID) {
		t.Fatalf("identity not found must list available: %+v", issues)
	}
}
func TestMacAutoIdentity(t *testing.T) {
	c := newCtx(t, "darwin", "macos:\n  sign:\n    enabled: true\n", "")
	if c.Mac.Identity != "" {
		t.Fatal("identity should be auto")
	}
	if issues := (&MacApp{}).Preflight(c); len(issues) != 0 {
		t.Fatalf("%+v", issues)
	}
	if c.Mac.Identity != devID {
		t.Fatalf("auto-selected %q", c.Mac.Identity)
	}
	c = newCtx(t, "darwin", "macos:\n  sign:\n    enabled: true\n", "")
	c.Tools.(*fakeTools).probes["security find-identity -v -p codesigning"] = "0 valid identities found"
	issues := (&MacApp{}).Preflight(c)
	if len(issues) != 1 || !issues[0].Fatal {
		t.Fatalf("%+v", issues)
	}
}

func TestDMGCreateDMGAndFlavorProfile(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "darwin", "build:\n  flavor: prod\n  mode: profile\nmacos:\n  dmg:\n    tool: create-dmg\n    volume_name: XueHua Installer\n", "")
	c.Tools.(*fakeTools).bins["create-dmg"] = "/opt/homebrew/bin/create-dmg"
	if got := strings.Join(steps(t, &DMG{}, c)[0].Args, " "); got != "build macos --profile --flavor prod" {
		t.Fatal(got)
	}
	p := plan(t, &DMG{}, c)
	cs := cmds(p)
	if !strings.Contains(cs[0], "Products/Profile-prod/XueHua.app") {
		t.Fatalf("%q", cs[0])
	}
	last := cs[len(cs)-1]
	if !strings.HasPrefix(last, "create-dmg --volname 'XueHua Installer' --window-size 660 400") || !strings.Contains(last, "--app-drop-link 480 190 --skip-jenkins") {
		t.Fatalf("%s", last)
	}
	if names(p)[0] != "xue_hua_im-prod-1.0.0+1-macos-universal-profile.dmg" {
		t.Fatal(names(p))
	}
	if len(p.Notes) == 0 {
		t.Fatal("expected unsigned note")
	}
}

func TestParseNotary(t *testing.T) {
	r, ok := ParseNotary("Conducting pre-submission checks...\n{\"id\":\"abc-123\",\"status\":\"Invalid\",\"message\":\"Processing complete\"}\n")
	if !ok || r.Status != "Invalid" || r.ID != "abc-123" {
		t.Fatalf("%+v", r)
	}
}

// ------------------------------------------------------------ other targets

func TestDesktopAndWebPlans(t *testing.T) {
	posixPaths(t)
	c := newCtx(t, "linux", "", "")
	if got := strings.Join(steps(t, &WebZip{}, c)[0].Args, " "); got != "build web --release" {
		t.Fatal(got)
	}
	if names(plan(t, &WebZip{}, c))[0] != "xue_hua_im-1.0.0+1-web.zip" {
		t.Fatal("web name")
	}
	if names(plan(t, &LinuxTar{}, c))[0] != "xue_hua_im-1.0.0+1-linux-arm64.tar.gz" {
		t.Fatal(names(plan(t, &LinuxTar{}, c)))
	}
	if cs := cmds(plan(t, &Deb{}, c)); !strings.HasPrefix(cs[0], "dpkg-deb --build --root-owner-group") {
		t.Fatal(cs)
	}
	c2 := newCtx(t, "windows", "build:\n  flavor: prod\n", "")
	c2.Host.Arch = "amd64"
	s := steps(t, &WinZip{}, c2)
	if strings.Join(s[0].Args, " ") != "build windows --release" || len(s[0].Warnings) != 1 {
		t.Fatalf("flavor must be dropped with a warning on windows: %+v", s[0])
	}
	if names(plan(t, &WinZip{}, c2))[0] != "xue_hua_im-1.0.0+1-windows-x64-portable.zip" { // no {flavor}: not a flavored build
		t.Fatal(names(plan(t, &WinZip{}, c2)))
	}
	if issues := (&Msix{}).Preflight(c2); len(issues) == 0 || !strings.Contains(issues[len(issues)-1].Fix, "flutter pub add --dev msix") {
		t.Fatalf("%+v", issues)
	}
}

func TestWebIgnoresObfuscate(t *testing.T) {
	c := newCtx(t, "linux", "build:\n  obfuscate: true\nweb:\n  base_href: /app/\n  wasm: true\n", "")
	s := steps(t, &WebZip{}, c)
	if strings.Join(s[0].Args, " ") != "build web --release --base-href /app/ --wasm" || len(s[0].Warnings) != 1 {
		t.Fatalf("%+v", s[0])
	}
}

func TestHelpers(t *testing.T) {
	if MsixVersion("1.2.3") != "1.2.3.0" || DebArch("x64") != "amd64" || RpmArch("arm64") != "aarch64" {
		t.Fatal("helpers")
	}
	if g := StableGUID("com.x"); g != StableGUID("com.x") || len(g) != 36 || g == StableGUID("com.y") {
		t.Fatal(g)
	}
	if ParseJavaMajor(`openjdk version "21.0.12" 2026-08-18`) != 21 || ParseJavaMajor(`java version "1.8.0_202"`) != 8 {
		t.Fatal("java")
	}
	if tg, ok := Get("appbundle"); !ok || tg.Name() != "aab" {
		t.Fatal("alias")
	}
	if _, ok := Get("nope"); ok {
		t.Fatal("unknown")
	}
}

func TestPlatformRulesForEveryTarget(t *testing.T) {
	for _, tg := range All() {
		for _, goos := range []string{"darwin", "linux", "windows"} {
			ok, _ := host.Host{OS: goos}.CanBuild(tg.Platform())
			want := map[host.Platform][]string{
				host.Android: {"darwin", "linux", "windows"}, host.Web: {"darwin", "linux", "windows"},
				host.IOS: {"darwin"}, host.MacOS: {"darwin"}, host.Windows: {"windows"}, host.Linux: {"linux"},
			}[tg.Platform()]
			exp := false
			for _, w := range want {
				exp = exp || w == goos
			}
			if ok != exp {
				t.Errorf("%s on %s: %v", tg.Name(), goos, ok)
			}
		}
	}
}

// Real output from a Mac with duplicate and revoked identities but no
// Developer ID certificate (SHA-1s shortened to fakes).
const macIdentitiesNoDevID = `  1) 1111111111111111111111111111111111111111 "Apple Development: Alex Example (AAAAAAAAA1)"
  2) 2222222222222222222222222222222222222222 "Apple Development: Jane Appleseed (BBBBBBBBB2)"
  3) 3333333333333333333333333333333333333333 "Apple Development: Other Developer (CCCCCCCCC3)" (CSSMERR_TP_CERT_REVOKED)
  4) 4444444444444444444444444444444444444444 "Apple Distribution: Other Developer (OTHERTEAM1)"
  5) 5555555555555555555555555555555555555555 "Apple Distribution: Other Developer (OTHERTEAM1)"
  6) 6666666666666666666666666666666666666666 "Apple Development: Former Developer (DDDDDDDDD4)" (CSSMERR_TP_CERT_REVOKED)
  7) 7777777777777777777777777777777777777777 "Apple Distribution: Former Developer (REVOKED001)" (CSSMERR_TP_CERT_REVOKED)
     7 valid identities found
`

func TestParseIdentitiesDedupesAndFlagsRevoked(t *testing.T) {
	ids := ParseIdentities(macIdentitiesNoDevID)
	if len(ids) != 6 {
		t.Fatalf("%+v", ids)
	}
	if ids[2].Name != "Apple Development: Other Developer (CCCCCCCCC3)" || ids[2].Problem != "CSSMERR_TP_CERT_REVOKED" {
		t.Fatalf("%+v", ids[2])
	}
	if ids[3].Name != "Apple Distribution: Other Developer (OTHERTEAM1)" || ids[3].Problem != "" {
		t.Fatalf("%+v", ids[3])
	}
	if got := identitySummary([]string{"Apple Development: a", "Apple Development: b", "Apple Distribution: c"}); got != "Apple Development ×2, Apple Distribution ×1" {
		t.Fatal(got)
	}
}

func TestMacSigningNoDeveloperIDCertificate(t *testing.T) {
	c := newCtx(t, "darwin", "macos:\n  sign:\n    identity: \""+devID+"\"\n    notary_profile: XueHua\n", "")
	c.Tools.(*fakeTools).probes["security find-identity -v -p codesigning"] = macIdentitiesNoDevID
	issues := (&DMG{}).Preflight(c)
	if len(issues) != 1 || !issues[0].Fatal {
		t.Fatalf("%+v", issues)
	}
	msg := issues[0].Msg
	for _, want := range []string{"no \"Developer ID Application\" certificate", devID, "Apple Development ×2, Apple Distribution ×1", "fpack.yaml macos.sign"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "Former") || strings.Contains(msg, "pubspec") || !strings.Contains(issues[0].Fix, "--no-sign") || !strings.Contains(issues[0].Fix, ".p12") {
		t.Fatalf("%+v", issues[0])
	}
	// --no-sign: no signing preflight at all
	c = newCtx(t, "darwin", "macos:\n  sign:\n    enabled: false\n    identity: \""+devID+"\"\n", "")
	c.Tools.(*fakeTools).probes["security find-identity -v -p codesigning"] = macIdentitiesNoDevID
	if issues := macSigningPreflight(c); len(issues) != 0 {
		t.Fatalf("%+v", issues)
	}
}
func TestIPADistributionCertificateForOtherTeamOnly(t *testing.T) {
	c := newCtx(t, "darwin", "", "")
	c.Tools.(*fakeTools).probes["security find-identity -v -p codesigning"] = macIdentitiesNoDevID
	var w *Issue
	for _, is := range (&IPA{}).Preflight(c) {
		if strings.Contains(is.Msg, "for team") {
			is := is
			w = &is
		}
	}
	if w == nil || w.Fatal || !w.IfFails || !strings.Contains(w.Msg, "ABCDE12345") || !strings.Contains(w.Msg, "OTHERTEAM1") || strings.Contains(w.Msg, "REVOKED001") {
		t.Fatalf("%+v", w)
	}
	// development export doesn't need a distribution certificate
	c = newCtx(t, "darwin", "ios:\n  export_method: development\n", "")
	c.Tools.(*fakeTools).probes["security find-identity -v -p codesigning"] = macIdentitiesNoDevID
	for _, is := range (&IPA{}).Preflight(c) {
		if strings.Contains(is.Msg, "for team") {
			t.Fatalf("%+v", is)
		}
	}
	if got := distributionTeams([]string{"Apple Distribution: X (AAAAAAAAAA)", "iPhone Distribution: Y (BBBBBBBBBB)", "Apple Development: Z (CCCCCCCCCC)", "Apple Distribution: X (AAAAAAAAAA)"}); !reflect.DeepEqual(got, []string{"AAAAAAAAAA", "BBBBBBBBBB"}) {
		t.Fatal(got)
	}
}

func TestKeytoolOutputIsLocaleIndependent(t *testing.T) {
	c := newCtx(t, "darwin", "", "")
	c.Project.AndroidReleaseDebugSigned = false
	chk := signerCheck(c, `(?m)^(?:Owner|所有者): (.+)$`)
	// keytool on a Chinese macOS without -J-Duser.language=en
	zh := "签名者 #1:\n\nCertificate #1:\n所有者: CN=example-signer, O=example-signer, C=CN\n发布者: CN=example-signer\n"
	if note, _ := chk(runnerResult(zh)); note != "signed by CN=example-signer, O=example-signer, C=CN" {
		t.Fatalf("%q", note)
	}
	if note, _ := chk(runnerResult("Signer #1:\n\nCertificate #1:\nOwner: CN=X, C=CN\nIssuer: CN=X\n")); note != "signed by CN=X, C=CN" {
		t.Fatalf("%q", note)
	}
	if a := keytoolEnglish(); len(a) != 2 || a[0] != "-J-Duser.language=en" {
		t.Fatal(a)
	}
}

// newCtxPlanNotes: DMG notes for a project with no signing configured at all.
func newCtxPlanNotes(t *testing.T) []string {
	return plan(t, &DMG{}, newCtx(t, "darwin", "", "")).Notes
}

func TestMsixNeverPromptsForCertificate(t *testing.T) {
	c := newCtx(t, "windows", "", "")
	p := plan(t, &Msix{}, c)
	cs := strings.Join(cmds(p), "\n")
	if !strings.Contains(cs, "msix:create") || !strings.Contains(cs, "--install-certificate false") {
		t.Fatal(cs)
	}
	if len(p.Notes) != 1 || !strings.Contains(p.Notes[0], "test certificate") {
		t.Fatal(p.Notes)
	}
	// the project decides explicitly → fpack doesn't override it
	c = newCtx(t, "windows", "", "msix_config:\n  install_certificate: true\n  certificate_path: C:\\certs\\app.pfx\n")
	p = plan(t, &Msix{}, c)
	if cs := strings.Join(cmds(p), "\n"); strings.Contains(cs, "--install-certificate") || len(p.Notes) != 0 {
		t.Fatal(cs, p.Notes)
	}
}

func TestDebCopyright(t *testing.T) {
	for in, want := range map[string]string{"© 2026 Matkurban": "2026 Matkurban", "Copyright (c) 2026 A": "2026 A", "2026 B": "2026 B", "Copyright © 2026 C": "2026 C"} {
		if got := debCopyright(in); got != want {
			t.Errorf("debCopyright(%q) = %q, want %q", in, got, want)
		}
	}
}
