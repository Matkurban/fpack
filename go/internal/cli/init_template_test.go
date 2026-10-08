package cli

import (
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
)

func allPlatforms() map[host.Platform]bool {
	m := map[host.Platform]bool{}
	for _, p := range host.AllPlatforms {
		m[p] = true
	}
	return m
}

func TestInitMacSigningIsCommentedPlaceholders(t *testing.T) {
	i18n.Set(i18n.EN)
	proj := &project.Project{Name: "demo", Platforms: map[host.Platform]bool{host.MacOS: true},
		// a pubspec `dmg:` section (dmg package) must be ignored completely
		Pubspec: map[string]any{"dmg": map[string]any{"sign-certificate": "Developer ID Application: FromPubspec (AAAAAAAAAA)", "notary-profile": "PubProfile"}}}
	for _, ids := range [][]string{nil, {"Developer ID Application: Keychain Person (BBBBBBBBBB)"}} {
		out := renderInitYAML(initValues{Targets: []string{"macos", "dmg"}, Display: "Demo", Split: "false", OutDir: config.DefaultOutputDir, DevIDs: ids, Proj: proj})
		if strings.Contains(out, "FromPubspec") || strings.Contains(out, "PubProfile") || strings.Contains(out, "pubspec.yaml dmg") {
			t.Fatalf("pubspec dmg: leaked into fpack.yaml:\n%s", out)
		}
		for _, want := range []string{"\nmacos:\n", "\n  sign:\n", "    # notary_profile: NotaryProfile\n", "    # installer_identity: \"Developer ID Installer: Your Name (TEAMID)\"\n", "\n  pkg:\n", "    # install_location: /Applications\n"} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q:\n%s", want, out)
			}
		}
		if ids != nil && !strings.Contains(out, "    # identity: \"Developer ID Application: Keychain Person (BBBBBBBBBB)\"") {
			t.Fatalf("keychain identity not offered:\n%s", out)
		}
		var c config.Config
		if err := config.Parse([]byte(out), &c); err != nil {
			t.Fatal(err)
		}
		if c.MacOS.Sign.Identity != "" || c.MacOS.Sign.Enabled != nil || c.MacOS.Sign.NotaryProfile != "" || c.MacOS.Sign.InstallerIdentity != "" {
			t.Fatalf("signing must stay off until the user uncomments it: %+v", c.MacOS.Sign)
		}
	}
}

// Every key appears in the template (commented or not), with a comment, and
// uncommenting every key line still yields a valid config.
func TestInitListsEveryKeyAndUncommentsCleanly(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.EN, i18n.ZH} {
		i18n.Set(lang)
		proj := &project.Project{Name: "xue_hua_im", Version: "1.2.0", BuildNumber: "7", Description: "IM app", AndroidApplicationID: "com.xuehua.im",
			AndroidFlavors: []string{"dev", "prod"}, Platforms: allPlatforms()}
		out := renderInitYAML(initValues{Targets: []string{"apk", "web"}, Display: "雪花IM", Split: "both", OutDir: config.DefaultOutputDir, Proj: proj})
		if !strings.HasPrefix(out, "# yaml-language-server: $schema="+config.SchemaURL) {
			t.Fatal("schema header missing")
		}
		lines := strings.Split(out, "\n")
		for _, k := range config.Keys {
			name := k.Path[strings.LastIndex(k.Path, ".")+1:]
			found := false
			for i, l := range lines {
				tl := strings.TrimSpace(l)
				if strings.HasPrefix(tl, name+":") || strings.HasPrefix(tl, "# "+name+":") {
					if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), "#") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("%s: key %s missing or without comment", lang, k.Path)
			}
		}
		if lang == i18n.ZH && !strings.Contains(out, "默认：") {
			t.Error("zh template lacks Chinese comments")
		}
		var c config.Config
		if err := config.Parse([]byte(out), &c); err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if c.App.DisplayName != "雪花IM" || c.Android.SplitPerABI != config.ABIBoth || len(c.Build.Targets) != 2 {
			t.Fatalf("active values: %+v", c)
		}
		// Uncomment every key line: the file must still parse.
		var un []string
		for _, l := range lines {
			tl := strings.TrimLeft(l, " ")
			ind := l[:len(l)-len(tl)]
			if strings.HasPrefix(tl, "# ") {
				rest := tl[2:]
				if i := strings.Index(rest, ":"); i > 0 && !strings.ContainsAny(rest[:i], " （(") && config.IsKeyName(rest[:i]) {
					un = append(un, ind+rest)
					continue
				}
			}
			un = append(un, l)
		}
		var all config.Config
		if err := config.Parse([]byte(strings.Join(un, "\n")), &all); err != nil {
			t.Fatalf("%s: uncommented template invalid: %v", lang, err)
		}
	}
	i18n.Set(i18n.EN)
}
