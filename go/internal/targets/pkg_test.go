package targets

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const devInstaller = "Developer ID Installer: XueHua Tech (ABCDE12345)"

func installerIdentities() string {
	return `  1) 1111111111111111111111111111111111111111 "Apple Development: dev@x.com (ZZZ)"
  2) 4444444444444444444444444444444444444444 "` + devInstaller + `"
  3) 5555555555555555555555555555555555555555 "3rd Party Mac Developer Installer: XueHua Tech (ABCDE12345)"
  4) 6666666666666666666666666666666666666666 "Developer ID Installer: Old Cert (ABCDE12345)" (CSSMERR_TP_CERT_REVOKED)
     3 valid identities found`
}

func pkgCtx(t *testing.T, cfg string) *Context {
	c := newCtx(t, "darwin", cfg, "")
	ft := c.Tools.(*fakeTools)
	ft.bins["pkgbuild"] = "/usr/bin/pkgbuild"
	ft.bins["productbuild"] = "/usr/bin/productbuild"
	ft.probes["security find-identity -v -p basic"] = installerIdentities()
	ft.probes["xcrun --find notarytool"] = "/usr/bin/notarytool"
	return c
}

func TestPkgUnsignedPlan(t *testing.T) {
	posixPaths(t)
	c := pkgCtx(t, "")
	if is := (&Pkg{}).Preflight(c); len(is) != 0 {
		t.Fatalf("%+v", is)
	}
	p := plan(t, &Pkg{}, c)
	stage := filepath.Join(c.WorkDir, "stage", "pkg")
	tmp := filepath.Join(stage, "xue_hua_im-1.0.0+1-macos.pkg")
	want := []string{
		"ditto " + filepath.Join(c.Project.Root, "build/macos/Build/Products/Release/XueHua.app") + " " + filepath.Join(stage, "root", "XueHua.app"),
		"pkgbuild --root " + filepath.Join(stage, "root") + " --component-plist " + filepath.Join(stage, "component.plist") + " --identifier com.xuehua.im --version 1.0.0 --install-location /Applications " + filepath.Join(stage, "packages", "app.pkg"),
		"productbuild --distribution " + filepath.Join(stage, "distribution.xml") + " --package-path " + filepath.Join(stage, "packages") + " " + tmp,
	}
	if cs := cmds(p); !reflect.DeepEqual(cs, want) {
		t.Fatalf("pkg commands:\n%s\nwant:\n%s", strings.Join(cs, "\n"), strings.Join(want, "\n"))
	}
	if names(p)[0] != "xue_hua_im-1.0.0+1-macos.pkg" || p.Artifacts[0].Kind != "macOS installer (pkg)" {
		t.Fatalf("%v %+v", names(p), p.Artifacts)
	}
	if n := strings.Join(p.Notes, "\n"); !strings.Contains(n, "macos.sign.installer_identity") || !strings.Contains(n, "Developer ID Installer") {
		t.Fatal(n)
	}
}

func TestPkgSignedNotarizedWithResources(t *testing.T) {
	posixPaths(t)
	c := pkgCtx(t, "macos:\n  sign:\n    identity: \""+devID+"\"\n    installer_identity: \"Developer ID Installer: XueHua\"\n    notary_profile: XueHua\n  pkg:\n    identifier: com.xuehua.im.pkg\n    install_location: /Applications/XueHua\n    title: 雪花 IM & Co\n    license: LICENSE.rtf\n    background: assets/bg.png\n")
	for _, f := range []string{"LICENSE.rtf", "assets/bg.png"} {
		os.MkdirAll(filepath.Dir(filepath.Join(c.Project.Root, f)), 0o755)
		os.WriteFile(filepath.Join(c.Project.Root, f), []byte("x"), 0o644)
	}
	if is := (&Pkg{}).Preflight(c); len(is) != 0 {
		t.Fatalf("%+v", is)
	}
	p := plan(t, &Pkg{}, c)
	cs := strings.Join(cmds(p), "\n")
	stage := filepath.Join(c.WorkDir, "stage", "pkg")
	tmp := filepath.Join(stage, "xue_hua_im-1.0.0+1-macos.pkg")
	for _, w := range []string{
		"codesign --force --options runtime --timestamp --entitlements",
		"--identifier com.xuehua.im.pkg --version 1.0.0 --install-location /Applications/XueHua",
		"--resources " + filepath.Join(stage, "resources") + " --sign 'Developer ID Installer: XueHua' --timestamp " + tmp,
		"pkgutil --check-signature " + tmp,
		"xcrun notarytool submit " + tmp + " --keychain-profile XueHua --wait",
		"xcrun stapler staple " + tmp,
		"spctl --assess --type install --verbose=2 " + tmp,
	} {
		if !strings.Contains(cs, w) {
			t.Errorf("missing %q in:\n%s", w, cs)
		}
	}
	if p.Artifacts[0].Kind != "macOS installer (pkg, signed, notarized)" || len(p.Notes) != 0 {
		t.Fatalf("%+v %v", p.Artifacts, p.Notes)
	}
	// run the internal file op: distribution.xml and resources
	os.MkdirAll(stage, 0o755)
	for _, op := range p.Ops {
		if op.Fn != nil && strings.Contains(op.Desc, "distribution.xml") {
			if err := op.Fn(); err != nil {
				t.Fatal(err)
			}
		}
	}
	dist, _ := os.ReadFile(filepath.Join(stage, "distribution.xml"))
	for _, w := range []string{"<title>雪花 IM &amp; Co</title>", `<license file="LICENSE.rtf"/>`, `<background file="bg.png"`, `hostArchitectures="arm64,x86_64"`, `<pkg-ref id="com.xuehua.im.pkg" version="1.0.0" onConclusion="none">app.pkg</pkg-ref>`} {
		if !strings.Contains(string(dist), w) {
			t.Errorf("distribution.xml lacks %q:\n%s", w, dist)
		}
	}
	comp, _ := os.ReadFile(filepath.Join(stage, "component.plist"))
	if !strings.Contains(string(comp), "<key>BundleIsRelocatable</key>\n\t\t<false/>") || !strings.Contains(string(comp), "<string>XueHua.app</string>") {
		t.Fatal(string(comp))
	}
	if _, err := os.Stat(filepath.Join(stage, "resources", "bg.png")); err != nil {
		t.Fatal(err)
	}
}

func TestPkgInstallerIdentityChecks(t *testing.T) {
	// unknown identity lists the usable installer identities (not revoked ones)
	c := pkgCtx(t, "macos:\n  sign:\n    installer_identity: \"Developer ID Installer: Nobody\"\n")
	is := (&Pkg{}).Preflight(c)
	if len(is) != 1 || !is[0].Fatal || !strings.Contains(is[0].Msg, devInstaller) || strings.Contains(is[0].Msg, "Old Cert") || !strings.Contains(is[0].Fix, "Developer ID Application") {
		t.Fatalf("%+v", is)
	}
	// Mac App Store installer certificate → warning
	c = pkgCtx(t, "macos:\n  sign:\n    installer_identity: \"3rd Party Mac Developer Installer\"\n")
	is = (&Pkg{}).Preflight(c)
	if len(is) != 1 || is[0].Fatal || !strings.Contains(is[0].Msg, "App Store") {
		t.Fatalf("%+v", is)
	}
	// signed pkg, app not Developer ID signed → note; no notarization
	c = pkgCtx(t, "macos:\n  sign:\n    installer_identity: \""+devInstaller+"\"\n")
	p := plan(t, &Pkg{}, c)
	if p.Artifacts[0].Kind != "macOS installer (pkg, signed)" || len(p.Notes) != 1 || !strings.Contains(p.Notes[0], "not Developer ID signed") {
		t.Fatalf("%+v %v", p.Artifacts, p.Notes)
	}
	// --no-sign also leaves the pkg unsigned
	c = pkgCtx(t, "macos:\n  sign:\n    enabled: false\n    installer_identity: \""+devInstaller+"\"\n")
	p = plan(t, &Pkg{}, c)
	if strings.Contains(strings.Join(cmds(p), "\n"), "--sign") || !strings.Contains(p.Notes[0], "--no-sign") {
		t.Fatal(cmds(p), p.Notes)
	}
	// notarization on but no installer identity: pkg unsigned, not notarized, says why
	c = pkgCtx(t, "macos:\n  sign:\n    identity: \""+devID+"\"\n    notary_profile: XueHua\n")
	p = plan(t, &Pkg{}, c)
	if strings.Contains(strings.Join(cmds(p), "\n"), "notarytool") || len(p.Notes) != 2 || !strings.Contains(p.Notes[1], "notarizes signed packages") {
		t.Fatal(cmds(p), p.Notes)
	}
	// missing tools / resource files
	c = pkgCtx(t, "macos:\n  pkg:\n    welcome: docs/welcome.md\n    readme: nope.html\n")
	delete(c.Tools.(*fakeTools).bins, "productbuild")
	os.MkdirAll(filepath.Join(c.Project.Root, "docs"), 0o755)
	os.WriteFile(filepath.Join(c.Project.Root, "docs", "welcome.md"), []byte("x"), 0o644)
	is = (&Pkg{}).Preflight(c)
	msgs := ""
	for _, i := range is {
		msgs += i.Msg + "\n"
	}
	if len(is) != 3 || !strings.Contains(msgs, "pkgbuild/productbuild") || !strings.Contains(msgs, "unsupported file type \".md\"") || !strings.Contains(msgs, "file not found: nope.html") {
		t.Fatal(msgs)
	}
}

func TestPkgSharesFlutterBuildWithMacosAndDMG(t *testing.T) {
	c := pkgCtx(t, "")
	a, b, d := steps(t, &MacApp{}, c), steps(t, &DMG{}, c), steps(t, &Pkg{}, c)
	if a[0].Key != "macos" || b[0].Key != a[0].Key || d[0].Key != a[0].Key || !reflect.DeepEqual(a[0].Args, d[0].Args) {
		t.Fatal(a, b, d)
	}
	if tg, ok := Get("pkg"); !ok || tg.Name() != "pkg" {
		t.Fatal("pkg not registered")
	}
	found := false
	for _, tg := range ForPlatform("macos") {
		found = found || tg.Name() == "pkg"
	}
	if !found {
		t.Fatal("pkg not a macOS target")
	}
}
