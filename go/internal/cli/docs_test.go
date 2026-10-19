package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/build"
	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/targets"
)

// The tests in this file keep the hand-written documentation honest: every
// variable, key, flag and target it mentions must exist, the generated
// exit-code tables must match build.ExitCodeDocs, and every public FPACK_*
// variable the code reads must be documented.

const repoRoot = "../../.."

// docFiles are the documents users read (the changelog is history and is
// not checked).
func docFiles(t *testing.T) []string {
	t.Helper()
	files := []string{"README.md", "README.ZH.md", "doc/configuration.md", "skills/fpack-cli/SKILL.md", "example/README.md", "example/fpack.yaml"}
	err := filepath.WalkDir(filepath.Join(repoRoot, "website", "content"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") && !strings.HasPrefix(d.Name(), "changelog") {
			rel, _ := filepath.Rel(repoRoot, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Skip("no website/content")
	}
	return files
}

func readDoc(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func exitCodesMarkdown(lang string) string {
	var b strings.Builder
	if lang == "zh" {
		b.WriteString("| 退出码 | 含义 |\n| --- | --- |\n")
	} else {
		b.WriteString("| Code | Meaning |\n| --- | --- |\n")
	}
	for _, c := range build.ExitCodeDocs {
		text := c.EN
		if lang == "zh" {
			text = c.ZH
		}
		fmt.Fprintf(&b, "| %d | %s |\n", c.Code, text)
	}
	return b.String()
}

// TestDocsExitCodes regenerates the exit-code tables (FPACK_UPDATE=1).
func TestDocsExitCodes(t *testing.T) {
	pages := map[string]string{
		"doc/configuration.md":          "zh",
		"website/content/exit-codes.md": "en", "website/content/exit-codes.zh.md": "zh",
	}
	for rel, lang := range pages {
		orig := readDoc(t, rel)
		doc, ok := config.ReplaceGenerated(orig, "EXIT", exitCodesMarkdown(lang))
		if !ok {
			t.Fatalf("markers for EXIT missing in %s", rel)
		}
		if os.Getenv("FPACK_UPDATE") == "1" {
			if err := os.WriteFile(filepath.Join(repoRoot, filepath.FromSlash(rel)), []byte(doc), 0o644); err != nil {
				t.Fatal(err)
			}
		} else if doc != orig {
			t.Errorf("%s: exit codes out of date: run FPACK_UPDATE=1 go test ./internal/cli -run TestDocs", rel)
		}
	}
	documented := map[int]bool{}
	for _, c := range build.ExitCodeDocs {
		documented[c.Code] = true
	}
	for _, code := range []int{build.ExitFailed, build.ExitUsage, build.ExitPrereq, build.ExitInterrupted} {
		if !documented[code] {
			t.Errorf("exit code %d is not in build.ExitCodeDocs", code)
		}
	}
}

var fpackVar = regexp.MustCompile(`\bFPACK_[A-Z0-9_]*[A-Z0-9]`)

// internalVars are passed between fpack's own processes or used by its
// tests only; they are deliberately not documented for users.
var internalVars = map[string]bool{"FPACK_KS_PASS": true, "FPACK_KEY_PASS": true, "FPACK_WRAPPER_VERSION": true, "FPACK_UPDATE": true}

// codeVars returns every FPACK_* name in the Go and Dart sources; public
// excludes tests and internalVars.
func codeVars(t *testing.T) (all, public map[string]bool) {
	t.Helper()
	all, public = map[string]bool{}, map[string]bool{}
	for _, dir := range []string{"go/cmd", "go/internal", "lib", "bin"} {
		err := filepath.WalkDir(filepath.Join(repoRoot, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || (!strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".dart")) {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			test := strings.HasSuffix(p, "_test.go")
			for _, m := range fpackVar.FindAllStringIndex(string(b), -1) {
				v := string(b[m[0]:m[1]])
				if m[1] < len(b) && b[m[1]] == '_' {
					continue // a prefix such as FPACK_MACOS_
				}
				all[v] = true
				if !test && !internalVars[v] {
					public[v] = true
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return all, public
}

// docVars returns the FPACK_* names in s, ignoring prefixes like FPACK_MACOS_*.
func docVars(s string) []string {
	var out []string
	for _, m := range fpackVar.FindAllStringIndex(s, -1) {
		rest := s[m[1]:]
		if strings.HasPrefix(rest, "_*") || strings.HasPrefix(rest, "*") || strings.HasPrefix(rest, "_…") {
			continue
		}
		out = append(out, s[m[0]:m[1]])
	}
	return out
}

func TestDocsEnvironmentVariables(t *testing.T) {
	all, public := codeVars(t)
	for _, rel := range docFiles(t) {
		for _, v := range docVars(readDoc(t, rel)) {
			if !all[v] {
				t.Errorf("%s mentions %s, which fpack does not read", rel, v)
			}
		}
	}
	// Every public variable is documented in each language's reference.
	refs := map[string][]string{
		"doc/configuration.md": {"doc/configuration.md"},
		"website (en)":         {"website/content/environment.md", "website/content/configuration.md"},
		"website (zh)":         {"website/content/environment.zh.md", "website/content/configuration.zh.md"},
	}
	for name, files := range refs {
		var text strings.Builder
		for _, f := range files {
			text.WriteString(readDoc(t, f))
		}
		have := map[string]bool{}
		for _, v := range docVars(text.String()) {
			have[v] = true
		}
		var missing []string
		for v := range public {
			if !have[v] {
				missing = append(missing, v)
			}
		}
		slices.Sort(missing)
		if len(missing) > 0 {
			t.Errorf("%s does not document %s", name, strings.Join(missing, ", "))
		}
	}
}

var keyCountClaim = regexp.MustCompile(`(\d+) (?:validated )?` + "`?" + `fpack\.yaml` + "`?" + ` keys|(\d+) 个(?:经过校验的)? ?` + "`?" + `fpack\.yaml` + "`?" + ` ?键`)

func TestDocsKeyCount(t *testing.T) {
	want := fmt.Sprint(len(config.Keys))
	for _, rel := range docFiles(t) {
		for _, m := range keyCountClaim.FindAllStringSubmatch(readDoc(t, rel), -1) {
			if got := m[1] + m[2]; got != want {
				t.Errorf("%s claims %s fpack.yaml keys, the registry has %s", rel, got, want)
			}
		}
	}
}

var dottedKey = regexp.MustCompile("`([a-z][a-z0-9_]*(?:\\.[a-z0-9_]+)+)(?:[:=][^`]*)?`")

func TestDocsConfigKeys(t *testing.T) {
	sections := map[string]bool{}
	for _, k := range config.Keys {
		sections[strings.Split(k.Path, ".")[0]] = true
	}
	known := func(p string) bool {
		if _, ok := config.KeyByPath(p); ok {
			return true
		}
		for _, k := range config.Keys {
			if strings.HasPrefix(k.Path, p+".") {
				return true
			}
		}
		return false
	}
	for _, rel := range docFiles(t) {
		for _, m := range dottedKey.FindAllStringSubmatch(readDoc(t, rel), -1) {
			p := m[1]
			if strings.HasSuffix(p, ".gradle") || strings.HasSuffix(p, ".kts") {
				continue // file names
			}
			if !sections[strings.Split(p, ".")[0]] || known(p) {
				continue
			}
			// map keys below a map-typed key (output.names.exe, hooks.post_package.dmg)
			parent, ok := p[:strings.LastIndex(p, ".")], false
			if k, found := config.KeyByPath(parent); found && strings.Contains(string(k.Kind), "map") {
				ok = true
			}
			if !ok {
				t.Errorf("%s mentions unknown key %s", rel, p)
			}
		}
	}
}

var yamlBlock = regexp.MustCompile("(?s)```ya?ml\\n(.*?)```")
var topKey = regexp.MustCompile(`(?m)^([a-z_]+):`)
var envRef = regexp.MustCompile(`\$\{[^}]*\}`)

// TestDocsYAMLExamples parses every fpack.yaml snippet with the strict
// config parser (unknown keys and wrong types fail).
func TestDocsYAMLExamples(t *testing.T) {
	sections := map[string]bool{}
	for _, k := range config.Keys {
		sections[strings.Split(k.Path, ".")[0]] = true
	}
	n := 0
	for _, rel := range docFiles(t) {
		text := readDoc(t, rel)
		blocks := [][]string{}
		if strings.HasSuffix(rel, ".yaml") {
			blocks = append(blocks, []string{"", text})
		} else {
			blocks = yamlBlock.FindAllStringSubmatch(text, -1)
		}
		for _, b := range blocks {
			tops := topKey.FindAllStringSubmatch(b[1], -1)
			if len(tops) == 0 || slices.ContainsFunc(tops, func(m []string) bool { return !sections[m[1]] }) {
				continue
			}
			n++
			src := envRef.ReplaceAllString(b[1], "x")
			if err := config.Parse([]byte(src), &config.Config{}); err != nil {
				t.Errorf("%s: fpack.yaml example does not parse: %v\n%s", rel, err, b[1])
			}
		}
	}
	if n < 10 {
		t.Errorf("only %d fpack.yaml examples found; is the block detection broken?", n)
	}
}

// shellBlockLines returns the lines of fenced blocks that hold commands.
func shellBlockLines(text string) []string {
	var out []string
	in, keep := false, false
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			if in {
				in = false
				continue
			}
			in = true
			lang := strings.TrimPrefix(t, "```")
			keep = lang == "" || lang == "bash" || lang == "sh" || lang == "shell" || lang == "console" || lang == "text" || lang == "powershell"
			continue
		}
		if in && keep {
			out = append(out, l)
		}
	}
	return out
}

var inlineCmd = regexp.MustCompile("`(fpack [^`]+)`")

// TestDocsCommandLines checks every documented `fpack …` invocation: the
// command exists, every flag before `--` belongs to it (or is global) and
// build targets are real targets or aliases.
func TestDocsCommandLines(t *testing.T) {
	global := map[string]flagSpec{}
	for _, f := range globalFlags {
		for _, n := range f.names {
			global[n] = f
		}
	}
	checked := 0
	for _, rel := range docFiles(t) {
		text := readDoc(t, rel)
		var lines []string
		lines = append(lines, shellBlockLines(text)...)
		for _, m := range inlineCmd.FindAllStringSubmatch(text, -1) {
			lines = append(lines, m[1])
		}
		for _, line := range lines {
			if msg := checkCommandLine(line, global); msg != "" {
				t.Errorf("%s: %s\n    %s", rel, msg, strings.TrimSpace(line))
			} else if strings.Contains(line, "fpack ") {
				checked++
			}
		}
	}
	if checked < 50 {
		t.Errorf("only %d fpack command lines checked; is the detection broken?", checked)
	}
}

func checkCommandLine(line string, global map[string]flagSpec) string {
	if i := strings.Index(line, " #"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	if i := strings.Index(line, "  "); i >= 0 {
		line = line[:i] // usage line: command, then its description
	}
	fields := strings.Fields(line)
	// skip leading VAR=value assignments, `$`, `run:` and `&&` chains
	start := -1
	for i, f := range fields {
		if f == "fpack" {
			start = i
			break
		}
		if !strings.Contains(f, "=") && f != "$" && f != "-" && f != "run:" {
			return ""
		}
	}
	if start < 0 {
		return ""
	}
	args := fields[start+1:]
	for i, a := range args {
		if a == "&&" || a == "|" || a == ">" || a == "||" || a == ";" {
			args = args[:i]
			break
		}
	}
	placeholder := func(s string) bool { return strings.ContainsAny(s, "<…[]|{}$") || s == "..." }
	var c *command
	var flags map[string]flagSpec
	pos := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" || strings.HasPrefix(a, "[") {
			break
		}
		if placeholder(a) {
			continue
		}
		if strings.HasPrefix(a, "-") {
			name := a
			if j := strings.Index(a, "="); j >= 0 {
				name = a[:j]
			}
			if c == nil && (name == "--version" || name == "-V" || name == "--wrapper-info") {
				continue
			}
			spec, ok := global[name]
			if !ok && flags != nil {
				spec, ok = flags[name]
			}
			if !ok {
				if c == nil {
					return "unknown global option " + name
				}
				return "unknown option " + name + " for fpack " + c.name
			}
			if (spec.kind == kString || spec.kind == kList) && !strings.Contains(a, "=") {
				i++
			}
			continue
		}
		if c == nil {
			if a == "help" {
				return ""
			}
			c = findCommand(a)
			if c == nil {
				return "unknown command " + a
			}
			flags = map[string]flagSpec{}
			for _, f := range c.flags {
				for _, n := range f.names {
					flags[n] = f
				}
			}
			continue
		}
		pos++
		if c.name == "build" {
			if _, ok := targets.Get(a); !ok {
				return "unknown target " + a
			}
		}
	}
	return ""
}
