package targets

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Matkurban/fpack/go/internal/runner"
)

const devIDTeam = "ABCDE12345"

func profileXML(kind, appID, aps string, expires time.Time) string {
	devices := ""
	all := "<key>ProvisionsAllDevices</key><true/>"
	if kind == "development" {
		devices = "<key>ProvisionedDevices</key><array><string>0000-1111</string></array>"
		all = ""
	}
	apsKey := ""
	if aps != "" {
		apsKey = "<key>com.apple.developer.aps-environment</key><string>" + aps + "</string>"
	}
	return `SecPolicySetValue: warning before the plist
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>Name</key><string>XueHua ` + kind + `</string>
<key>TeamIdentifier</key><array><string>` + devIDTeam + `</string></array>
<key>DeveloperCertificates</key><array><data>
Y2VydC1h
</data></array>
<key>ExpirationDate</key><date>` + expires.UTC().Format(time.RFC3339) + `</date>
` + all + devices + `
<key>Entitlements</key><dict>
<key>com.apple.application-identifier</key><string>` + appID + `</string>
<key>keychain-access-groups</key><array><string>` + devIDTeam + `.*</string></array>
` + apsKey + `
<key>com.apple.developer.team-identifier</key><string>` + devIDTeam + `</string>
</dict></dict></plist>`
}

const pushEntitlements = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>com.apple.security.app-sandbox</key><true/>
<key>com.apple.developer.aps-environment</key><string>production</string>
<key>keychain-access-groups</key><array><string>ABCDE12345.com.xuehua.im</string></array>
</dict></plist>`

func TestParsePlistAndProfileChecks(t *testing.T) {
	ft := &fakeTools{probes: map[string]string{}, fail: map[string]bool{}}
	future := time.Now().Add(240 * time.Hour)
	ft.probes["security cms -D -i good"] = profileXML("developer-id", devIDTeam+".com.xuehua.im", "production", future)
	ft.probes["security cms -D -i dev"] = profileXML("development", devIDTeam+".*", "development", future)
	ft.probes["security cms -D -i old"] = profileXML("developer-id", devIDTeam+".com.other", "", time.Now().Add(-time.Hour))
	ents, err := plistDict([]byte(pushEntitlements))
	if err != nil {
		t.Fatal(err)
	}
	good, err := LoadProfile(ft, "good")
	if err != nil {
		t.Fatal(err)
	}
	if good.TeamID != devIDTeam || !good.AllDevices || good.Development() || good.Expires.IsZero() {
		t.Fatalf("parsed %+v", good)
	}
	if p := CheckProfile(good, Signer{Team: devIDTeam}, "com.xuehua.im", ents, time.Now()); len(p) != 0 {
		t.Errorf("good profile: %v", p)
	}
	if p := strings.Join(CheckProfile(good, Signer{Team: "ZZZZZ99999"}, "com.xuehua.im", ents, time.Now()), "\n"); !strings.Contains(p, "team") {
		t.Errorf("team mismatch not reported: %s", p)
	}
	dev, _ := LoadProfile(ft, "dev")
	p := strings.Join(CheckProfile(dev, Signer{Team: devIDTeam}, "com.xuehua.im", ents, time.Now()), "\n")
	if !dev.Development() || !strings.Contains(p, "development/ad hoc") || !strings.Contains(p, `"production"`) {
		t.Errorf("development profile: %s", p)
	}
	old, _ := LoadProfile(ft, "old")
	p = strings.Join(CheckProfile(old, Signer{Team: devIDTeam}, "com.xuehua.im", ents, time.Now()), "\n")
	for _, want := range []string{"expired", "app id", "com.apple.developer.aps-environment is not authorized"} {
		if !strings.Contains(p, want) {
			t.Errorf("missing %q in:\n%s", want, p)
		}
	}
	if _, err := LoadProfile(ft, "missing"); err == nil {
		t.Error("unreadable profile must fail")
	}
	if identityTeam("Developer ID Application: XueHua Tech (ABCDE12345)") != devIDTeam {
		t.Error("identityTeam")
	}
	if !wildcardMatch("ABCDE12345.*", "ABCDE12345.com.x") || wildcardMatch("ABCDE12345.com.y", "ABCDE12345.com.x") {
		t.Error("wildcardMatch")
	}
}

func macSignedCtx(t *testing.T, yaml string) *Context {
	t.Helper()
	c := newCtx(t, "darwin", "macos:\n  sign:\n    identity: \"Developer ID Application: XueHua Tech (ABCDE12345)\"\n"+yaml, "")
	must(t, os.WriteFile(filepath.Join(c.Project.Root, "macos", "Runner", "Release.entitlements"), []byte(pushEntitlements), 0o644))
	c.Tools.(*fakeTools).probes["security find-identity -v -p codesigning"] = fmt.Sprintf("  1) %X \"Developer ID Application: XueHua Tech (ABCDE12345)\"", sha1.Sum([]byte("cert-a"))) //nolint:gosec // test fingerprint
	return c
}

func TestProfilePreflight(t *testing.T) {
	// restricted entitlements without a profile
	c := macSignedCtx(t, "")
	f, _ := preflightOf(t, &MacApp{}, c)
	if len(f) != 1 || !strings.Contains(f[0].Msg, "restricted entitlements") || !strings.Contains(f[0].Fix, "Developer ID") {
		t.Fatalf("no profile: %+v", f)
	}
	// profile_check: false
	c = macSignedCtx(t, "    profile_check: false\n")
	if f, _ := preflightOf(t, &MacApp{}, c); len(f) != 0 {
		t.Errorf("profile_check false: %+v", f)
	}
	// valid profile
	c = macSignedCtx(t, "    provisioning_profile: macos/XueHua.provisionprofile\n")
	prof := c.Project.Abs("macos/XueHua.provisionprofile")
	c.Tools.(*fakeTools).probes["security cms -D -i "+prof] = profileXML("developer-id", devIDTeam+".com.xuehua.im", "production", time.Now().Add(time.Hour))
	if f, _ := preflightOf(t, &MacApp{}, c); len(f) != 0 {
		t.Errorf("valid profile: %+v", f)
	}
	// development profile configured
	c.Tools.(*fakeTools).probes["security cms -D -i "+prof] = profileXML("development", devIDTeam+".com.xuehua.im", "development", time.Now().Add(time.Hour))
	if f, _ := preflightOf(t, &MacApp{}, c); len(f) < 2 {
		t.Errorf("development profile: %+v", f)
	}
}

type fakeEnv struct{ cmds []string }

func (f *fakeEnv) env() OpEnv {
	return OpEnv{Run: func(c runner.Cmd) (runner.Result, error) {
		f.cmds = append(f.cmds, strings.Join(append([]string{c.Name}, c.Args...), " "))
		return runner.Result{}, nil
	}, Print: func(string, string) {}, Status: func(string) {}}
}

func stagedApp(t *testing.T, c *Context) (app, ext string) {
	t.Helper()
	app = filepath.Join(c.Stage("macos"), "XueHua.app")
	ext = filepath.Join(app, "Contents", "PlugIns", "Share.appex")
	must(t, os.MkdirAll(filepath.Join(ext, "Contents"), 0o755))
	ft := c.Tools.(*fakeTools)
	ft.probes["plutil -extract CFBundleIdentifier raw -o - "+filepath.Join(app, "Contents", "Info.plist")] = "com.xuehua.im"
	ft.probes["plutil -extract CFBundleIdentifier raw -o - "+filepath.Join(ext, "Contents", "Info.plist")] = "com.xuehua.im.share"
	ft.probes["codesign -d --entitlements - --xml "+ext] = "Executable=…\n" + pushEntitlements
	return app, ext
}

func TestEmbedProfilesAndNestedSigning(t *testing.T) {
	c := macSignedCtx(t, "    provisioning_profile: macos/XueHua.provisionprofile\n    provisioning_profiles: {com.xuehua.im.share: macos/Share.provisionprofile}\n  smoke_test: true\n  smoke_test_seconds: 1\n")
	ft := c.Tools.(*fakeTools)
	for name, appID := range map[string]string{"macos/XueHua.provisionprofile": "com.xuehua.im", "macos/Share.provisionprofile": "com.xuehua.im.share"} {
		p := c.Project.Abs(name)
		must(t, os.WriteFile(p, []byte("profile "+appID), 0o644))
		ft.probes["security cms -D -i "+p] = profileXML("developer-id", devIDTeam+"."+appID, "production", time.Now().Add(time.Hour))
	}
	app, ext := stagedApp(t, c)
	p := plan(t, &MacApp{}, c)
	var descs []string
	for _, op := range p.Ops {
		descs = append(descs, op.Desc)
	}
	all := strings.Join(descs, "|")
	if !strings.Contains(all, "embed provisioning profiles") || !strings.Contains(all, "nested bundles") || !strings.Contains(all, "smoke test") {
		t.Fatalf("ops: %s", all)
	}
	var nested []nestedSignature
	fe := &fakeEnv{}
	must(t, embedProfiles(c, app, &nested, fe.env()))
	for _, b := range []string{app, ext} {
		if _, err := os.Stat(filepath.Join(b, "Contents", "embedded.provisionprofile")); err != nil {
			t.Errorf("profile not embedded in %s", b)
		}
	}
	if len(nested) != 1 || nested[0].Entitlements == "" {
		t.Fatalf("nested: %+v", nested)
	}
	if !strings.HasPrefix(nested[0].Entitlements, filepath.Join(c.WorkDir, "nested-entitlements")+string(filepath.Separator)) {
		t.Fatalf("entitlements written under payload dir: %s", nested[0].Entitlements)
	}
	if _, err := os.Stat(nested[0].Entitlements); err != nil {
		t.Fatalf("entitlements file missing before signing: %v", err)
	}
	if _, err := signNestedOp(c, &nested, []string{"--options", "runtime"}).Run(context.Background(), fe.env()); err != nil {
		t.Fatal(err)
	}
	if len(fe.cmds) != 1 || !strings.Contains(fe.cmds[0], "--entitlements "+nested[0].Entitlements) || !strings.HasSuffix(fe.cmds[0], "Share.appex") {
		t.Errorf("nested signing: %v", fe.cmds)
	}
	if _, err := os.Stat(nested[0].Entitlements); !os.IsNotExist(err) {
		t.Errorf("entitlements file not cleaned after signing: %v", err)
	}
}

// TestNestedEntitlementsStayOutOfPackageRoot ensures the temporary
// entitlements file used to re-sign a nested bundle is not written next to
// the .app. For dmg/pkg that directory is the package root (hdiutil
// -srcfolder / pkgbuild --root), so a leak would ship nested-0.entitlements
// beside the app — the bug reported for Elephant.app.
func TestNestedEntitlementsStayOutOfPackageRoot(t *testing.T) {
	c := macSignedCtx(t, "")
	off := false
	c.Config.MacOS.Sign.ProfileCheck = &off
	root := filepath.Join(c.Stage("dmg"), "root")
	app := filepath.Join(root, "Elephant.app")
	ext := filepath.Join(app, "Contents", "PlugIns", "Share.appex")
	must(t, os.MkdirAll(filepath.Join(ext, "Contents"), 0o755))
	ft := c.Tools.(*fakeTools)
	ft.probes["plutil -extract CFBundleIdentifier raw -o - "+filepath.Join(app, "Contents", "Info.plist")] = "com.kurban.elephantapp"
	ft.probes["plutil -extract CFBundleIdentifier raw -o - "+filepath.Join(ext, "Contents", "Info.plist")] = "com.kurban.elephantapp.ShareExtension"
	ft.probes["codesign -d --entitlements - --xml "+ext] = "Executable=…\n" + pushEntitlements

	var nested []nestedSignature
	must(t, embedProfiles(c, app, &nested, (&fakeEnv{}).env()))
	if len(nested) != 1 || nested[0].Entitlements == "" {
		t.Fatalf("nested: %+v", nested)
	}
	leaked := filepath.Join(root, "nested-0.entitlements")
	if _, err := os.Stat(leaked); err == nil {
		t.Fatalf("nested entitlements leaked into DMG/pkg root: %s", leaked)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".entitlements") {
			t.Fatalf("package root contains entitlements file %q", e.Name())
		}
		if e.Name() != "Elephant.app" {
			t.Fatalf("unexpected entry in package root: %s", e.Name())
		}
	}
	if !strings.Contains(nested[0].Entitlements, filepath.Join("nested-entitlements", "nested-0.entitlements")) {
		t.Fatalf("expected WorkDir/nested-entitlements/…, got %s", nested[0].Entitlements)
	}
}

func TestEmbedProfilesRefusesDevelopmentProfile(t *testing.T) {
	c := macSignedCtx(t, "")
	app, ext := stagedApp(t, c)
	ft := c.Tools.(*fakeTools)
	ft.probes["security cms -D -i "+filepath.Join(app, "Contents", "embedded.provisionprofile")] = profileXML("development", devIDTeam+".*", "development", time.Now().Add(time.Hour))
	ft.probes["security cms -D -i "+filepath.Join(ext, "Contents", "embedded.provisionprofile")] = profileXML("development", devIDTeam+".*", "development", time.Now().Add(time.Hour))
	var nested []nestedSignature
	err := embedProfiles(c, app, &nested, (&fakeEnv{}).env())
	if err == nil || !strings.Contains(err.Error(), "development profile") || !strings.Contains(err.Error(), "Share.appex") {
		t.Fatalf("err = %v", err)
	}
	f := false
	c.Config.MacOS.Sign.ProfileCheck = &f
	if err := embedProfiles(c, app, &nested, (&fakeEnv{}).env()); err != nil {
		t.Errorf("profile_check false: %v", err)
	}
}

func TestSmokeTest(t *testing.T) {
	c := macSignedCtx(t, "")
	app := filepath.Join(t.TempDir(), "X.app")
	ft := c.Tools.(*fakeTools)
	ft.probes["plutil -extract CFBundleExecutable raw -o - "+filepath.Join(app, "Contents", "Info.plist")] = "xue_hua"
	fe := &fakeEnv{}
	note, err := runSmokeTest(context.Background(), c, fe.env(), app, time.Millisecond)
	if err != nil || !strings.Contains(note, "passed") || len(fe.cmds) != 1 || fe.cmds[0] != "open -n "+app {
		t.Fatalf("passing smoke test: %q %v %v", note, err, fe.cmds)
	}
	ft.fail["pgrep -f "+filepath.Join(app, "Contents", "MacOS", "xue_hua")] = true
	if _, err := runSmokeTest(context.Background(), c, fe.env(), app, time.Millisecond); err == nil || !strings.Contains(err.Error(), "not running") {
		t.Errorf("dead app: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runSmokeTest(ctx, c, fe.env(), app, time.Hour); !errors.Is(err, runner.ErrInterrupted) {
		t.Errorf("cancelled: %v", err)
	}
}

func TestProfileCertificate(t *testing.T) {
	ft := &fakeTools{probes: map[string]string{}, fail: map[string]bool{}}
	ft.probes["security cms -D -i p"] = profileXML("developer-id", devIDTeam+".com.x", "", time.Now().Add(time.Hour))
	certA := fmt.Sprintf("%X", sha1.Sum([]byte("cert-a"))) //nolint:gosec // test fingerprint
	ft.probes["security find-identity -v -p codesigning"] = `  1) ` + certA + ` "Developer ID Application: A (` + devIDTeam + `)"
  2) 94B38B25487D5AC5C97016D41FCBD05FF9D466C4 "Developer ID Application: B (` + devIDTeam + `)"
     2 valid identities found`
	prof, err := LoadProfile(ft, "p")
	if err != nil || len(prof.Certs) != 1 || prof.Certs[0] != certA {
		t.Fatalf("certs %v %v", prof, err)
	}
	if s := resolveSigner(ft, "Developer ID Application: A ("+devIDTeam+")"); s.SHA1 != certA || s.Team != devIDTeam {
		t.Errorf("signer A: %+v", s)
	}
	if p := CheckProfile(prof, resolveSigner(ft, "Developer ID Application: A ("+devIDTeam+")"), "com.x", nil, time.Now()); len(p) != 0 {
		t.Errorf("matching cert: %v", p)
	}
	signerB := resolveSigner(ft, "94b38b25487d5ac5c97016d41fcbd05ff9d466c4")
	p := strings.Join(CheckProfile(prof, signerB, "com.x", nil, time.Now()), "")
	if signerB.Team != devIDTeam || !strings.Contains(p, "another Developer ID Application certificate") || !strings.Contains(p, "94B38B25…") {
		t.Errorf("other cert: %+v %s", signerB, p)
	}
}
