package pack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/config"
)

func TestRenderDefaultTemplate(t *testing.T) {
	tm := config.DefaultNameTemplate
	cases := []struct {
		f    Fields
		want string
	}{
		{Fields{App: "xue_hua_im", Version: "1.0.0", Build: "1", Platform: "android", Arch: "universal", Mode: "release"}, "xue_hua_im-1.0.0+1-android-universal"},
		{Fields{App: "xue_hua_im", Version: "1.0.0", Build: "1", Platform: "android", Arch: "arm64-v8a", Mode: "release"}, "xue_hua_im-1.0.0+1-android-arm64-v8a"},
		{Fields{App: "app", Version: "2.1.0", Build: "37", Platform: "ios", Arch: "arm64", Mode: "release", Flavor: "prod"}, "app-prod-2.1.0+37-ios-arm64"},
		{Fields{App: "app", Version: "1.0.0", Build: "1", Platform: "ios", Arch: "arm64", Variant: "unsigned", Mode: "release"}, "app-1.0.0+1-ios-arm64-unsigned"},
		{Fields{App: "app", Version: "1.0.0", Build: "1", Platform: "windows", Arch: "x64", Variant: "setup", Mode: "profile"}, "app-1.0.0+1-windows-x64-setup-profile"},
		{Fields{App: "app", Version: "1.0.0", Build: "1", Platform: "web", Mode: "release"}, "app-1.0.0+1-web"},
		{Fields{App: "app", Version: "1.0.0", Platform: "macos", Arch: "universal", Mode: "release"}, "app-1.0.0-macos-universal"},
	}
	for _, c := range cases {
		got, err := Render(tm, c.f)
		if err != nil || got != c.want {
			t.Errorf("got %q (%v) want %q", got, err, c.want)
		}
	}
}

func TestRenderCustomAndErrors(t *testing.T) {
	got, _ := Render("{app}_{version}_{platform}{_arch}", Fields{App: "My App", Version: "1.0", Platform: "linux", Arch: "x64"})
	if got != "My_App_1.0_linux_x64" {
		t.Fatal(got)
	}
	if _, err := Render("{app}-{verison}", Fields{}); err == nil || !strings.Contains(err.Error(), "{verison}") {
		t.Fatal("expected unknown placeholder error")
	}
	if SanitizeName("a/b:c") != "a-b-c" {
		t.Fatal("sanitize")
	}
	d, _ := RenderDir(config.DefaultOutputDir, Fields{Version: "1.0.0", Build: "3"})
	if d != "dist/1.0.0+3" {
		t.Fatal(d)
	}
}

func TestArchivesAndChecksums(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "lib"), 0o755)
	os.WriteFile(filepath.Join(src, "app"), []byte("bin"), 0o755)
	os.WriteFile(filepath.Join(src, "lib", "libapp.so"), []byte("so"), 0o644)
	os.Symlink("app", filepath.Join(src, "link"))
	out := t.TempDir()

	z := filepath.Join(out, "a.zip")
	if err := Zip(src, z, "bundle"); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(z)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]os.FileMode{}
	for _, f := range zr.File {
		names[f.Name] = f.Mode()
	}
	zr.Close()
	// Windows has no executable bit to carry over.
	if (runtime.GOOS != "windows" && names["bundle/app"]&0o100 == 0) || names["bundle/link"]&os.ModeSymlink == 0 {
		t.Fatalf("zip entries: %v", names)
	}

	tg := filepath.Join(out, "a.tar.gz")
	if err := TarGz(src, tg, "demo"); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(tg)
	gz, _ := gzip.NewReader(f)
	tr := tar.NewReader(gz)
	found := map[string]bool{}
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		found[h.Name] = true
	}
	f.Close()
	if !found["demo/app"] || !found["demo/lib/libapp.so"] || !found["demo/link"] {
		t.Fatalf("tar entries: %v", found)
	}

	p, sums, err := WriteChecksums(out)
	if err != nil || len(sums) != 2 {
		t.Fatalf("%v %v", sums, err)
	}
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "  a.tar.gz\n") || !strings.Contains(string(data), "  a.zip\n") {
		t.Fatal(string(data))
	}
}
