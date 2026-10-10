package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/targets"
)

var (
	cjk = regexp.MustCompile(`[\p{Han}\x{3000}-\x{303f}\x{ff01}-\x{ff60}]`)
	// Things that are the same in every language: code, commands, paths,
	// keys, flags, variables, URLs, identifiers.
	codeish = []*regexp.Regexp{
		regexp.MustCompile("`[^`]*`"),
		regexp.MustCompile(`https?://\S+`),
		regexp.MustCompile(`\$\{[^}]*\}|\$\w+`),
		regexp.MustCompile(`--?[\w-]+(=\S*)?`),
		regexp.MustCompile(`\b[A-Z0-9_]{2,}\b`),
		regexp.MustCompile(`"[^"]*"`),
		regexp.MustCompile(`\S*[/\\.<>:=\[\]{}()|@+*#'%]\S*`),
	}
	englishPhrase = regexp.MustCompile(`\b[A-Za-z]{2,}\b[ \t]+\b[A-Za-z]{2,}\b`)
	yamlKeyLine   = regexp.MustCompile(`^\s*#?\s*[\w.-]+:(\s|$)`)
	// Proper nouns and product names that stay English in Chinese text.
	allowed     = regexp.MustCompile(`(?i)\b(Developer ID( Application| Installer)?|Your Name|App Store( Connect)?|Google Play|Inno Setup|Hardened Runtime|Android Studio|Start menu|Xcode|dry run|Apple ID|Aurora Notes|XueHua IM|Flutter SDK|macOS|Windows|Linux|Android|iOS|Web)\b`)
	commandLine = regexp.MustCompile(`^\s*([$→·=]\s|fpack\s|flutter\s|sudo\s|brew\s|choco\s|winget\s|xcrun\s|dart\s|spctl\s|ditto\s|codesign\s|security\s)`)
)

// leftovers returns lines of zh output that still contain English prose,
// or lines of en output that contain Chinese.
func leftovers(lang, text string, yaml bool) []string {
	var bad []string
	for _, l := range strings.Split(text, "\n") {
		if lang == "en" {
			if cjk.MatchString(l) {
				bad = append(bad, l)
			}
			continue
		}
		if cjk.MatchString(l) || commandLine.MatchString(l) {
			continue
		}
		if yaml && (!strings.HasPrefix(strings.TrimSpace(l), "#") || yamlKeyLine.MatchString(l)) {
			continue // YAML values and commented-out keys are data, not prose
		}
		c := allowed.ReplaceAllString(l, "")
		for _, re := range codeish {
			c = re.ReplaceAllString(c, " ")
		}
		if englishPhrase.MatchString(c) {
			bad = append(bad, l)
		}
	}
	return bad
}

// Every user-facing output must be entirely in the selected language.
func TestNoMixedLanguageOutput(t *testing.T) {
	bad := "build:\n  flavr: x\n  targets: [wbe]\nandroid:\n  split_per_abi: maybe\n  abis: [mips]\n  signing:\n    store_file: missing.jks\nweb:\n  base_href: app\noutput:\n  name: \"{nope}\"\nhooks:\n  post_package:\n    foo: [x]\n"
	good := "build:\n  flavor: nope\n  dart_define:\n    A: b\nmacos:\n  sign:\n    identity: \"Developer ID Application: X (ABCDE12345)\"\n"
	for _, lang := range []string{"en", "zh"} {
		bproj, _ := fixture(t, bad)
		proj, sdk := fixture(t, good)
		env := map[string]string{"FPACK_LANG": lang, "FPACK_FLUTTER": sdk}
		var out strings.Builder
		cmds := [][]string{
			{"--help"}, {"--version"}, {"help"},
			{"build", "--help"}, {"doctor", "--help"}, {"list", "--help"}, {"init", "--help"},
			{"schema", "--help"}, {"notarize", "--help"}, {"clean", "--help"}, {"version", "--help"},
			{"-C", proj, "doctor"}, {"-C", proj, "list"},
			{"-C", proj, "build", "apk", "web", "--dry-run"},
			{"-C", proj, "build", "bogus"},
			{"-C", proj, "build", "--frobnicate"},
			{"-C", proj, "notarize", "status"},
			{"-C", proj, "clean", "--dist"},
			{"nosuchcommand"},
			{"-C", bproj, "build", "apk", "--dry-run"},
			{"-C", bproj, "doctor"},
			{"-C", proj, "build", "apk", "--dart-define", "nokv", "--dry-run"},
			{"-C", filepath.Join(proj, "missing"), "list"},
		}
		for _, a := range cmds {
			out.WriteString(run(t, env, a...).all())
		}
		if bad := leftovers(lang, out.String(), false); len(bad) > 0 {
			t.Errorf("%s output has %d mixed-language lines:\n%s", lang, len(bad), strings.Join(bad, "\n"))
		}

		os.Remove(filepath.Join(proj, "fpack.yaml"))
		r := run(t, env, "-C", proj, "init", "--yes")
		if bad := leftovers(lang, r.all(), false); len(bad) > 0 {
			t.Errorf("%s init output:\n%s", lang, strings.Join(bad, "\n"))
		}
		b, err := os.ReadFile(filepath.Join(proj, "fpack.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if bad := leftovers(lang, string(b), true); len(bad) > 0 {
			t.Errorf("%s fpack.yaml from init has %d mixed-language lines:\n%s", lang, len(bad), strings.Join(bad, "\n"))
		}
	}
}

// NOTARIZATION.md follows the language too.
func TestNotaryMarkdownLanguage(t *testing.T) {
	defer i18n.Set(i18n.EN)
	now := time.Now()
	rec := &targets.NotaryRecord{Schema: 1, Tool: "fpack", App: "Demo", Version: "1.0.0", Updated: now, Submissions: []targets.NotarySubmission{
		{Target: "dmg", Artifact: "/d/demo.dmg", Uploaded: "/d/demo.dmg", SHA256: "ab", ID: "123", SubmittedAt: now, UpdatedAt: now, Auth: "keychain profile X", AuthArgs: []string{"--keychain-profile", "X"}, Status: "Invalid", Message: "x", Log: "/d/log.json", Issues: []string{"x"}},
		{Target: "macos", Artifact: "/d/demo.zip", Uploaded: "/d/demo.zip", App: "Demo.app", SHA256: "cd", ID: "456", SubmittedAt: now, UpdatedAt: now, Auth: "keychain profile X", AuthArgs: []string{"--keychain-profile", "X"}, Status: "In Progress"},
	}}
	for _, l := range []i18n.Lang{i18n.EN, i18n.ZH} {
		i18n.Set(l)
		md := targets.NotaryMarkdown(rec, "/d")
		if bad := leftovers(string(l), md, false); len(bad) > 0 {
			t.Errorf("%s NOTARIZATION.md:\n%s", l, strings.Join(bad, "\n"))
		}
	}
}
