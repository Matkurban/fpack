package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseKinds(t *testing.T) {
	specs := buildFlags
	p, err := parse([]string{"apk", "-m", "profile", "--dart-define", "A=1", "--dart-define=B=2", "--split-per-abi=both", "-f", "aab", "--", "--no-tree-shake-icons"}, specs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.pos, []string{"apk", "aab"}) {
		t.Fatal(p.pos)
	}
	if p.s("mode") != "profile" || !p.b("force") {
		t.Fatalf("%+v", p)
	}
	if !reflect.DeepEqual(p.l("dart-define"), []string{"A=1", "B=2"}) {
		t.Fatal(p.l("dart-define"))
	}
	if p.s("split-per-abi") != "both" {
		t.Fatal(p.s("split-per-abi"))
	}
	if !reflect.DeepEqual(p.pass, []string{"--no-tree-shake-icons"}) {
		t.Fatal(p.pass)
	}
}

func TestParseOptionalValueDefaults(t *testing.T) {
	p, err := parse([]string{"--split-per-abi", "apk"}, buildFlags)
	if err != nil {
		t.Fatal(err)
	}
	if !p.has("split-per-abi") || p.s("split-per-abi") != "true" || !reflect.DeepEqual(p.pos, []string{"apk"}) {
		t.Fatalf("%q %v", p.s("split-per-abi"), p.pos)
	}
}

func TestParseErrors(t *testing.T) {
	specs := buildFlags
	for args, want := range map[string]string{
		"--mode":                "needs a value",
		"--flavr x":             "--flavor",
		"--force=maybe":         "expected true or false",
		"--no-tree-shake-icons": "--",
	} {
		_, err := parse(strings.Fields(args), specs)
		if err == nil {
			t.Errorf("%q: expected error", args)
			continue
		}
		if want != "" && !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v (want %q)", args, err, want)
		}
	}
}
