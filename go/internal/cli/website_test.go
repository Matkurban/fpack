package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/config"
)

// websiteCommands are documented on the website's commands page.
var websiteCommands = []string{"build", "doctor", "list", "init", "schema", "notarize", "clean", "version"}

// helpMarkdown renders `fpack help <command>` for every command.
func helpMarkdown(t *testing.T, lang string) string {
	t.Helper()
	var b strings.Builder
	for i, c := range websiteCommands {
		r := run(t, map[string]string{"FPACK_LANG": lang}, "help", c)
		if r.code != 0 || strings.TrimSpace(r.stdout) == "" {
			t.Fatalf("help %s: exit %d\n%s", c, r.code, r.all())
		}
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("### `fpack " + c + "`\n\n```text\n" + strings.TrimRight(r.stdout, "\n") + "\n```\n")
	}
	return b.String()
}

// changelogMarkdown is CHANGELOG.md without its title.
func changelogMarkdown(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	s = strings.TrimPrefix(s, "# Changelog\n")
	return strings.TrimSpace(s) + "\n"
}

// TestWebsiteUpToDate keeps the generated parts of the documentation
// website (website/content) in sync with the CLI, the key registry and the
// changelog. FPACK_UPDATE=1 rewrites them.
func TestWebsiteUpToDate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("generated on Linux/macOS (help output uses POSIX paths)")
	}
	content := filepath.Join("..", "..", "..", "website", "content")
	if _, err := os.Stat(content); err != nil {
		t.Skip("no website/content")
	}
	changelog := changelogMarkdown(t)
	for _, lang := range []string{"en", "zh"} {
		dir := content
		if lang == "zh" {
			dir = filepath.Join(content, "zh")
		}
		pages := map[string]map[string]string{
			"configuration.md": {"KEYS": config.SiteMarkdown(lang)},
			"environment.md":   {"ENV": config.EnvMarkdown(lang)},
			"commands.md":      {"HELP": helpMarkdown(t, lang)},
			"changelog.md":     {"CHANGELOG": changelog},
		}
		for page, blocks := range pages {
			path := filepath.Join(dir, page)
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			orig := strings.ReplaceAll(string(b), "\r\n", "\n")
			doc := orig
			for name, text := range blocks {
				var ok bool
				if doc, ok = config.ReplaceGenerated(doc, name, text); !ok {
					t.Fatalf("markers for %s missing in %s", name, path)
				}
			}
			if os.Getenv("FPACK_UPDATE") == "1" {
				if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
					t.Fatal(err)
				}
			} else if doc != orig {
				t.Errorf("%s is out of date: run FPACK_UPDATE=1 go test ./internal/cli -run TestWebsiteUpToDate", path)
			}
		}
	}
}
