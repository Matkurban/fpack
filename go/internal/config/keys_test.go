package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestRegistryCoversEveryField(t *testing.T) {
	reg := map[string]bool{}
	for _, k := range Keys {
		if reg[k.Path] {
			t.Errorf("duplicate key %s", k.Path)
		}
		reg[k.Path] = true
		if k.Doc.EN == "" || k.Doc.ZH == "" || k.Targets == "" || k.Example == "" {
			t.Errorf("%s: doc/targets/example missing", k.Path)
		}
		if k.Kind == KEnum && len(k.Enum) == 0 {
			t.Errorf("%s: enum without values", k.Path)
		}
	}
	leaves := map[string]bool{}
	for _, p := range LeafPaths() {
		leaves[p] = true
		if !reg[p] {
			t.Errorf("field %s has no registry entry", p)
		}
	}
	for p := range reg {
		if !leaves[p] {
			t.Errorf("registry entry %s has no Config field", p)
		}
	}
	// every section of a key has a heading
	secs := map[string]bool{}
	for _, s := range Sections {
		secs[s.Path] = true
	}
	for _, k := range Keys {
		if i := strings.LastIndex(k.Path, "."); i > 0 && !secs[k.Path[:i]] {
			t.Errorf("%s: no Sections entry for %s", k.Path, k.Path[:i])
		}
	}
}

func TestEnvVarsAreUniqueAndSettable(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Keys {
		if k.Env == "" {
			continue
		}
		if seen[k.Env] {
			t.Errorf("duplicate env %s", k.Env)
		}
		seen[k.Env] = true
		c := &Config{}
		v := k.Example
		switch k.Kind {
		case KBool:
			v = "true"
		case KList:
			v = "a,b"
		case KSplit:
			v = "both"
		}
		if err := c.SetString(k.Path, strings.Trim(v, "[]{}\"")); err != nil {
			t.Errorf("%s (%s): %v", k.Env, k.Path, err)
		}
	}
	names := []string{}
	for _, e := range EnvVars() {
		names = append(names, e[0])
	}
	if !sort.StringsAreSorted(names) && len(names) < 20 {
		t.Log(names)
	}
}

func TestExamplesParse(t *testing.T) {
	// Every key's example must be a valid value for that key.
	for _, k := range Keys {
		doc := yamlFor(k.Path, k.Example)
		var c Config
		if err := Parse([]byte(doc), &c); err != nil {
			t.Errorf("%s: example %q: %v", k.Path, k.Example, err)
			continue
		}
		if !c.IsSet(k.Path) {
			t.Errorf("%s: example %q did not set the field", k.Path, k.Example)
		}
	}
}

// yamlFor nests "a.b.c: v" into YAML.
func yamlFor(path, value string) string {
	parts := strings.Split(path, ".")
	var b strings.Builder
	for i, p := range parts {
		b.WriteString(strings.Repeat("  ", i) + p + ":")
		if i == len(parts)-1 {
			b.WriteString(" " + value)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func TestFriendlyTypeErrors(t *testing.T) {
	cases := map[string]string{
		"android:\n  signing:\n    v1: maybe\n":        `line 3: android.signing.v1: expected true or false, got "maybe"`,
		"macos:\n  dmg:\n    icon_size: big\n":         `line 3: macos.dmg.icon_size: expected a number, got "big"`,
		"output:\n  names: [a, b]\n":                   `line 2: output.names: expected a map (key: value), got a list`,
		"linux:\n  pakage_name: x\n":                   `unknown key "pakage_name" in "linux" (did you mean "package_name"?)`,
		"macos:\n  dmg:\n    window_size: [1, 2, 3]\n": `line 3: expected two numbers like [600, 400]`,
	}
	for in, want := range cases {
		var c Config
		err := Parse([]byte(in), &c)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q:\n got  %v\n want %s", in, err, want)
		}
	}
}

// The committed schema must match the registry (run with FPACK_UPDATE=1 to
// regenerate it).
func TestSchemaUpToDate(t *testing.T) {
	got, err := Schema()
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("..", "..", "..", "schema", "fpack.schema.json")
	if os.Getenv("FPACK_UPDATE") == "1" {
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil || string(want) != string(got) {
		t.Fatalf("schema/fpack.schema.json is out of date: run FPACK_UPDATE=1 go test ./internal/config (%v)", err)
	}
	var v map[string]any
	if json.Unmarshal(got, &v) != nil {
		t.Fatal("invalid JSON")
	}
}

// doc/configuration.md embeds generated tables (FPACK_UPDATE=1 rewrites).
func TestDocsUpToDate(t *testing.T) {
	path := filepath.Join("..", "..", "..", "doc", "configuration.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	for name, content := range map[string]string{"KEYS": Markdown("zh"), "ENV": EnvMarkdown("zh")} {
		var ok bool
		doc, ok = ReplaceGenerated(doc, name, content)
		if !ok {
			t.Fatalf("markers for %s missing in doc/configuration.md", name)
		}
	}
	if os.Getenv("FPACK_UPDATE") == "1" {
		os.WriteFile(path, []byte(doc), 0o644)
	} else if doc != string(b) {
		t.Fatal("doc/configuration.md is out of date: run FPACK_UPDATE=1 go test ./internal/config")
	}
}
