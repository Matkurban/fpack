package cli

import (
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/project"
)

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
		if !strings.Contains(out, "\nmacos:\n") || !strings.Contains(out, "  # sign:\n") || !strings.Contains(out, "notary_profile: NotaryProfile") {
			t.Fatalf("missing commented macos.sign section:\n%s", out)
		}
		if !strings.Contains(out, "  #   installer_identity: \"Developer ID Installer: Your Name (TEAMID)\"") || !strings.Contains(out, "  # pkg:") || !strings.Contains(out, "  #   install_location: /Applications") {
			t.Fatalf("missing pkg placeholders:\n%s", out)
		}
		if ids != nil && !strings.Contains(out, "  #   identity: \"Developer ID Application: Keychain Person (BBBBBBBBBB)\"") {
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
