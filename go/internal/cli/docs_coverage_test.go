package cli

import (
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/config"
)

// The website is organised as a learning path (see doc/docs-design.md):
// tutorials and guides teach, the reference lists. These tests make sure no
// feature exists only in generated reference tables: every command, flag,
// target and key feature must be explained by hand-written text, in English
// and in Chinese.

var docLangs = map[string]string{"en": ".md", "zh": ".zh.md"}

// sitePage returns a website page without its generated blocks.
func sitePage(t *testing.T, page, lang string) string {
	t.Helper()
	text := readDoc(t, "website/content/"+page+docLangs[lang])
	for {
		i := strings.Index(text, "<!-- BEGIN GENERATED")
		if i < 0 {
			return text
		}
		j := strings.Index(text[i:], "<!-- END GENERATED")
		if j < 0 {
			return text[:i]
		}
		k := strings.Index(text[i+j:], "-->")
		text = text[:i] + text[i+j+k+3:]
	}
}

func sitePages(t *testing.T, lang string, pages ...string) string {
	var b strings.Builder
	for _, p := range pages {
		b.WriteString(sitePage(t, p, lang))
		b.WriteString("\n")
	}
	return b.String()
}

var teachingPages = []string{
	"getting-started", "concepts", "installation",
	"platforms/android", "platforms/ios", "platforms/macos", "platforms/macos-credentials", "platforms/macos-profiles",
	"platforms/windows", "platforms/linux", "platforms/web",
	"guides/project-and-output", "guides/build-variants", "guides/secrets", "guides/hooks", "guides/many-targets",
}

// TestDocsCoverCommandsAndFlags: every command, alias and option is described
// in the hand-written part of the command reference, and every build option
// is taught in a tutorial or guide.
func TestDocsCoverCommandsAndFlags(t *testing.T) {
	for lang := range docLangs {
		ref := sitePage(t, "commands", lang)
		teach := sitePages(t, lang, teachingPages...)
		for _, c := range commands() {
			if !strings.Contains(ref, "fpack "+c.name) {
				t.Errorf("commands (%s): command %q is not described", lang, c.name)
			}
			for _, a := range c.aliases {
				if !strings.Contains(ref, "`"+a+"`") {
					t.Errorf("commands (%s): alias %q of %s is not described", lang, a, c.name)
				}
			}
			for _, f := range c.flags {
				for _, n := range f.names {
					if !strings.Contains(ref, "`"+n) {
						t.Errorf("commands (%s): option %s of fpack %s is not described", lang, n, c.name)
					}
				}
				if c.name == "build" && !strings.Contains(teach, f.names[0]) {
					t.Errorf("%s: build option %s is not taught in any tutorial or guide", lang, f.names[0])
				}
			}
		}
		for _, f := range globalFlags {
			for _, n := range f.names {
				if !strings.Contains(ref, "`"+n) {
					t.Errorf("commands (%s): global option %s is not described", lang, n)
				}
			}
		}
		for _, n := range []string{"--version", "-V", "--wrapper-info", "fpack notarize status", "fpack notarize finish", "fpack help"} {
			if !strings.Contains(ref, n) {
				t.Errorf("commands (%s): %s is not described", lang, n)
			}
		}
	}
}

// TestDocsCoverTargets: every target has a tutorial and every alias is listed.
func TestDocsCoverTargets(t *testing.T) {
	for lang := range docLangs {
		tutorials := sitePages(t, lang, "platforms/android", "platforms/ios", "platforms/macos", "platforms/windows", "platforms/linux", "platforms/web")
		for _, name := range config.TargetNames {
			if !strings.Contains(tutorials, "fpack build "+name) && !strings.Contains(tutorials, " "+name+" ") && !strings.Contains(tutorials, " "+name+"`") && !strings.Contains(tutorials, " "+name+"\n") {
				t.Errorf("%s: target %s has no tutorial", lang, name)
			}
		}
		lists := sitePages(t, lang, "targets", "guides/many-targets")
		for _, a := range config.TargetAliases {
			if !strings.Contains(lists, "`"+a+"`") {
				t.Errorf("%s: target alias %q is not listed", lang, a)
			}
		}
	}
}

// featureCoverage maps features to the page that teaches them. The needles
// are code-like so the same list holds for both languages.
var featureCoverage = []struct {
	page    string
	needles []string
}{
	{"getting-started", []string{"fpack doctor", "fpack list", "--dry-run", "fpack init", "dist/"}},
	{"concepts", []string{"fpack.yaml", "dist/", "build/fpack"}},
	{"installation", []string{"dart pub global activate fpack", "--wrapper-info", "FPACK_DOWNLOAD_URL", "FPACK_NO_DOWNLOAD", "FPACK_REBUILD", "FPACK_HOME", "FPACK_CORE", "FPACK_GO"}},
	{"platforms/android", []string{"fpack build apk", "aab", "--split-per-abi", "--abis", "android.signing", "FPACK_ANDROID_KEYSTORE_BASE64", "keytool", "apksigner"}},
	{"platforms/ios", []string{"fpack build ipa", "ios.team_id", "--export-method", "--export-options-plist", "--no-codesign", "provisioning_profiles"}},
	{"platforms/macos", []string{"fpack build macos", "dmg", "pkg", "macos.sign", "--notarize", "--no-notarize", "--notarize-no-wait", "fpack notarize status", "fpack notarize finish", "--smoke-test", "--dmg-tool", "installer_identity", "NOTARIZATION.md", "--no-sign"}},
	{"platforms/macos-credentials", []string{"notarytool store-credentials", "FPACK_NOTARY_API_KEY", "FPACK_NOTARY_APPLE_ID", "FPACK_NOTARY_PASSWORD", "notary_profile"}},
	{"platforms/macos-profiles", []string{"provisioning_profile", "profile_check", "--provisioning-profile", "entitlements", "--smoke-test"}},
	{"platforms/windows", []string{"fpack build windows", "exe", "msix", "inno_setup", "windows.sign", "signtool", "FPACK_WINDOWS_CERTIFICATE"}},
	{"platforms/linux", []string{"fpack build linux", "deb", "rpm", "appimage", "snap", "lxd", "multipass", "destructive"}},
	{"platforms/web", []string{"fpack build web", "--base-href", "--wasm"}},
	{"guides/project-and-output", []string{"app.display_name", "--build-name", "--build-number", "output.dir", "output.name", "output.names", "--force", "checksum_algorithm", "SHA256SUMS"}},
	{"guides/build-variants", []string{"--flavor", "-t lib/main_prod.dart", "--dart-define", "--dart-define-from-file", "--profile", "--debug", "--obfuscate", "--split-debug-info", "extra_args", "--flutter", ".fvmrc"}},
	{"guides/secrets", []string{"${", ":-", "FPACK_", "***", "0600"}},
	{"guides/hooks", []string{"hooks.pre_build", "hooks.pre_package", "hooks.post_package", "hooks.post_build", "FPACK_ARTIFACT", "FPACK_SUCCESS"}},
	{"guides/many-targets", []string{"--all", "--dry-run", "--json", "-C", "--config", "fpack clean", "--lang", "--no-color", "130"}},
	{"exit-codes", []string{"--json", "targets[].status", "notarization", "hookError", "conflicts"}},
	{"schema", []string{"fpack schema", "yaml-language-server"}},
	{"ci", []string{"activate fpack", "FPACK_ANDROID_KEYSTORE_BASE64", "--json", "--notarize-no-wait"}},
	{"troubleshooting", []string{"fpack doctor", "--verbose", "build/fpack/logs"}},
}

func TestDocsCoverFeatures(t *testing.T) {
	for lang := range docLangs {
		for _, f := range featureCoverage {
			text := sitePage(t, f.page, lang)
			for _, n := range f.needles {
				if !strings.Contains(text, n) {
					t.Errorf("%s (%s) does not cover %q", f.page, lang, n)
				}
			}
		}
	}
}
