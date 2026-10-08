package project

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Matkurban/fpack/go/internal/host"
)

const kts = `
android {
    flavorDimensions += "env"
    productFlavors {
        create("dev") {
            dimension = "env"
            applicationIdSuffix = ".dev"
        }
        create("prod") { dimension = "env" }
    }
    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("debug")
        }
    }
}`

const groovy = `
android {
    defaultConfig { applicationId "com.acme.app" }
    flavorDimensions "tier"
    productFlavors {
        free {
            dimension "tier"
        }
        paid { dimension "tier" }
    }
    signingConfigs { release { storeFile file(keystoreProperties['storeFile']) } }
    buildTypes {
        release {
            signingConfig signingConfigs.release
        }
    }
}`

func TestAndroidFlavors(t *testing.T) {
	if got := AndroidFlavors(kts); !reflect.DeepEqual(got, []string{"dev", "prod"}) {
		t.Errorf("kts: %v", got)
	}
	if got := AndroidFlavors(groovy); !reflect.DeepEqual(got, []string{"free", "paid"}) {
		t.Errorf("groovy: %v", got)
	}
	if got := AndroidFlavors("android { }"); got != nil {
		t.Errorf("none: %v", got)
	}
}

func TestReleaseDebugSigning(t *testing.T) {
	if !ReleaseUsesDebugSigning(kts) {
		t.Error("kts template uses debug signing")
	}
	if ReleaseUsesDebugSigning(groovy) {
		t.Error("groovy uses release signing")
	}
	if !ReleaseUsesDebugSigning("buildTypes { release { signingConfig signingConfigs.debug } }") {
		t.Error("groovy debug")
	}
}

func TestSplitVersion(t *testing.T) {
	for in, want := range map[string][2]string{"1.2.3+45": {"1.2.3", "45"}, "2.0.0": {"2.0.0", "1"}, "": {"1.0.0", "1"}} {
		v, b := SplitVersion(in)
		if v != want[0] || b != want[1] {
			t.Errorf("%q -> %s %s", in, v, b)
		}
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndFind(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "apps", "im")
	write(t, filepath.Join(app, "pubspec.yaml"), `name: xue_hua_im
version: 1.0.0+7
dependencies:
  flutter:
    sdk: flutter
  xue_hua_sdk:
    path: ../../packages/xue_hua_sdk
dev_dependencies:
  msix: ^3.16.0
dmg:
  sign-certificate: "Developer ID Application: Acme (TEAM123)"
  notary-profile: XueHua
`)
	write(t, filepath.Join(app, "android", "app", "build.gradle.kts"), kts+"\n// applicationId = \"com.xuehua.im\"\n")
	write(t, filepath.Join(app, "macos", "Runner", "Configs", "AppInfo.xcconfig"), "PRODUCT_NAME = 雪花IM\nPRODUCT_BUNDLE_IDENTIFIER = com.x\n")
	write(t, filepath.Join(app, "linux", "CMakeLists.txt"), `set(BINARY_NAME "xue_hua_im")`+"\n"+`set(APPLICATION_ID "com.xuehua.im")`)
	write(t, filepath.Join(app, "ios", "Runner.xcodeproj", "project.pbxproj"), "DEVELOPMENT_TEAM = ABCDE12345;\nPRODUCT_BUNDLE_IDENTIFIER = com.xuehua.im;\n")
	write(t, filepath.Join(app, "ios", "Runner.xcodeproj", "xcshareddata", "xcschemes", "Runner.xcscheme"), "")
	write(t, filepath.Join(app, "ios", "Runner.xcodeproj", "xcshareddata", "xcschemes", "prod.xcscheme"), "")
	write(t, filepath.Join(app, "lib", "main.dart"), "")
	write(t, filepath.Join(app, "flutter_launcher_icons.yaml"), "flutter_launcher_icons:\n  image_path: assets/icon.png\n")
	write(t, filepath.Join(app, "assets", "icon.png"), "png")
	os.MkdirAll(filepath.Join(app, "web"), 0o755)

	p, err := Find(filepath.Join(app, "lib"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Root != app || p.Name != "xue_hua_im" || p.Version != "1.0.0" || p.BuildNumber != "7" {
		t.Fatalf("basics: %+v", p)
	}
	if !p.Platforms[host.Android] || !p.Platforms[host.MacOS] || !p.Platforms[host.Web] || p.Platforms[host.Windows] {
		t.Fatalf("platforms: %v", p.Platforms)
	}
	if !reflect.DeepEqual(p.AndroidFlavors, []string{"dev", "prod"}) || !reflect.DeepEqual(p.IOSSchemes, []string{"prod"}) {
		t.Fatalf("flavors: %v %v", p.AndroidFlavors, p.IOSSchemes)
	}
	if !p.AndroidReleaseDebugSigned || p.IOSTeam != "ABCDE12345" || p.MacProductName != "雪花IM" || p.LinuxBinary != "xue_hua_im" {
		t.Fatalf("details: %+v", p)
	}
	if p.Identifier() != "com.xuehua.im" || p.LauncherIcon != "assets/icon.png" || !p.HasDep("msix") {
		t.Fatalf("id/icon/deps: %s %s", p.Identifier(), p.LauncherIcon)
	}
	if miss := p.MissingPathDeps(); len(miss) != 1 || miss[0].Name != "xue_hua_sdk" {
		t.Fatalf("missing path deps: %v", miss)
	}
	if p.Section("dmg")["notary-profile"] != "XueHua" {
		t.Fatal("dmg section")
	}

	// Monorepo root: not a project, but candidates are listed.
	_, err = Find(root)
	var nf *ErrNotFound
	if !errors.As(err, &nf) || !reflect.DeepEqual(nf.Candidates, []string{filepath.Join("apps", "im")}) {
		t.Fatalf("monorepo: %v %+v", err, nf)
	}
}
