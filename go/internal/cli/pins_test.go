package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/version"
)

// pinnedDocs contain CI snippets that pin the fpack version and GitHub Actions.
var pinnedDocs = []string{
	"README.md",
	"README.ZH.md",
	filepath.Join("website", "content", "ci.md"),
	filepath.Join("website", "content", "zh", "ci.md"),
}

var (
	// CI steps (`run: dart pub global activate fpack…`) and already pinned
	// mentions; plain install instructions keep installing the latest version.
	activatePin = regexp.MustCompile(`(run: dart pub global activate fpack|dart pub global activate fpack(?: \d+\.\d+\.\d+))(?: \d+\.\d+\.\d+)?`)
	usesPin     = regexp.MustCompile(`(uses: )([\w.-]+/[\w.-]+)@(v\d+)`)
)

// workflowActions maps every action used by this repository's own workflows to
// its major version, so the documented snippets follow the versions CI really runs.
func workflowActions(t *testing.T, root string) map[string]string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	m := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range usesPin.FindAllStringSubmatch(string(b), -1) {
			m[s[2]] = s[3]
		}
	}
	if len(m) == 0 {
		t.Skip("no workflows")
	}
	return m
}

// pinDocs rewrites CI snippets: `dart pub global activate fpack` pins the
// current version, and every action uses the major version of our own workflows.
func pinDocs(s string, actions map[string]string) string {
	s = activatePin.ReplaceAllStringFunc(s, func(m string) string {
		g := activatePin.FindStringSubmatch(m)
		return strings.TrimRight(strings.TrimSuffix(strings.TrimRight(g[1], "0123456789."), " "), " ") + " " + version.Version
	})
	return usesPin.ReplaceAllStringFunc(s, func(m string) string {
		g := usesPin.FindStringSubmatch(m)
		if v, ok := actions[g[2]]; ok {
			return g[1] + g[2] + "@" + v
		}
		return m
	})
}

// TestDocsPinsUpToDate keeps the fpack version and action versions in the CI
// examples current. FPACK_UPDATE=1 rewrites them.
func TestDocsPinsUpToDate(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	actions := workflowActions(t, root)
	for _, rel := range pinnedDocs {
		p := filepath.Join(root, rel)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		old := strings.ReplaceAll(string(b), "\r\n", "\n")
		want := pinDocs(old, actions)
		if !strings.Contains(want, "activate fpack "+version.Version) {
			t.Errorf("%s: no pinned `dart pub global activate fpack %s`", rel, version.Version)
		}
		if want == old {
			continue
		}
		if os.Getenv("FPACK_UPDATE") == "1" {
			if err := os.WriteFile(p, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		t.Errorf("%s: outdated fpack/action version pins; run FPACK_UPDATE=1 go test ./internal/cli -run TestDocsPinsUpToDate", rel)
	}
}

func TestPinDocs(t *testing.T) {
	a := map[string]string{"actions/checkout": "v7"}
	in := "- uses: actions/checkout@v4\n- uses: other/x@v1\n- run: dart pub global activate fpack && echo\n`dart pub global activate fpack 1.0.0` pins\n"
	want := "- uses: actions/checkout@v7\n- uses: other/x@v1\n- run: dart pub global activate fpack " + version.Version + " && echo\n`dart pub global activate fpack " + version.Version + "` pins\n"
	if got := pinDocs(in, a); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
