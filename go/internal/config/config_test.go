package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/pack"
)

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

const sample = `
app:
  name: xue_hua_im
  display_name: "雪花 IM"
build:
  targets: [apk, aab, ipa]
  mode: release
  flavor: prod
  dart_define:
    API_URL: https://api.example.com
    DEBUG: false
  dart_define_from_file: env/prod.json
  build_number: 42
output:
  dir: out/{version}
  overwrite: true
android:
  split_per_abi: both
  abis: [arm64-v8a, armeabi-v7a]
  signing:
    store_file: ${KEYSTORE_PATH}
    store_password: ${KS_PASS}
    key_alias: ${KEY_ALIAS:-upload}
ios:
  export_method: ad-hoc
macos:
  sign:
    notarize: false
  dmg:
    tool: hdiutil
`

func TestParseSample(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fpack.yaml")
	os.WriteFile(p, []byte(sample), 0o644)
	c, err := Load(p, envOf(map[string]string{"KEYSTORE_PATH": "/k/upload.jks", "KS_PASS": "pw"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.App.DisplayName != "雪花 IM" || c.Build.Flavor != "prod" || string(c.Build.BuildNumber) != "42" {
		t.Fatalf("basic fields: %+v", c.Build)
	}
	if !reflect.DeepEqual([]string(c.Build.DartDefine), []string{"API_URL=https://api.example.com", "DEBUG=false"}) {
		t.Fatalf("defines: %v", c.Build.DartDefine)
	}
	if !reflect.DeepEqual([]string(c.Build.DartDefineFromFile), []string{"env/prod.json"}) {
		t.Fatal("scalar list")
	}
	if c.SplitPerABI() != ABIBoth || len(c.ABIs()) != 2 || !c.Overwrite() {
		t.Fatal("android/output")
	}
	if c.Android.Signing.StoreFile != "/k/upload.jks" || c.Android.Signing.StorePassword != "pw" || c.Android.Signing.KeyAlias != "upload" {
		t.Fatalf("interpolation: %+v", c.Android.Signing)
	}
	if c.MacOS.Sign.Notarize == nil || *c.MacOS.Sign.Notarize {
		t.Fatal("notarize false")
	}
	if probs := c.Validate(); len(probs) != 0 {
		t.Fatal(probs)
	}
}

func TestUnsetEnvReported(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fpack.yaml")
	os.WriteFile(p, []byte("android:\n  signing:\n    store_password: ${NOPE}\n"), 0o644)
	c, err := Load(p, envOf(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.UnsetEnv) != 1 || c.UnsetEnv[0] != "NOPE" {
		t.Fatal(c.UnsetEnv)
	}
}

func TestUnknownKeySuggestion(t *testing.T) {
	var c Config
	err := Parse([]byte("build:\n  flavour: prod\n"), &c)
	if err == nil || !strings.Contains(err.Error(), `unknown key "flavour" in "build"`) || !strings.Contains(err.Error(), `did you mean "flavor"`) {
		t.Fatalf("got %v", err)
	}
	err = Parse([]byte("macos:\n  sign:\n    identiy: x\n"), &c)
	if err == nil || !strings.Contains(err.Error(), `"macos.sign"`) || !strings.Contains(err.Error(), `"identity"`) {
		t.Fatalf("got %v", err)
	}
}

func TestEmptyAndCommentOnly(t *testing.T) {
	var c Config
	if err := Parse([]byte(""), &c); err != nil {
		t.Fatal(err)
	}
	if err := Parse([]byte("# only comments\n"), &c); err != nil {
		t.Fatal(err)
	}
}

func TestDefaults(t *testing.T) {
	c := &Config{}
	if c.Mode() != "release" || c.OutputDir() != DefaultOutputDir || c.NameTemplate() != DefaultNameTemplate || c.Overwrite() || !c.Checksums() || !c.IOSCodesign() || c.SplitPerABI() != ABIUniversal {
		t.Fatal("defaults")
	}
}

func TestEnvOverridesConfig(t *testing.T) {
	c := &Config{}
	Parse([]byte("build:\n  mode: profile\n  flavor: dev\nmacos:\n  sign:\n    enabled: true\n"), c)
	err := ApplyEnv(c, envOf(map[string]string{"FPACK_MODE": "debug", "FPACK_MACOS_SIGN": "no", "FPACK_SPLIT_PER_ABI": "true", "FPACK_BUILD_NUMBER": "7"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode() != "debug" || c.Build.Flavor != "dev" || *c.MacOS.Sign.Enabled || c.SplitPerABI() != ABISplit || c.Build.BuildNumber != "7" {
		t.Fatalf("env precedence: %+v", c.Build)
	}
	if err := ApplyEnv(c, envOf(map[string]string{"FPACK_OVERWRITE": "maybe"})); err == nil {
		t.Fatal("expected bad bool error")
	}
}

func TestValidate(t *testing.T) {
	c := &Config{}
	Parse([]byte("build:\n  mode: fast\nandroid:\n  abis: [x86]\n  signing:\n    store_file: a.jks\nios:\n  export_method: store\noutput:\n  name: \"{app}-{vers}\"\n  names: {exe: \"{app}{-nope}\"}\n"), c)
	probs := strings.Join(c.Validate(), "\n")
	for _, want := range []string{"build.mode", "unknown ABI \"x86\"", "ios.export_method", "store_password", "key_alias", "output.name: unknown placeholder {vers}", "output.names.exe: unknown placeholder {-nope}"} {
		if !strings.Contains(probs, want) {
			t.Errorf("missing %q in %s", want, probs)
		}
	}
}

func TestBadTypes(t *testing.T) {
	var c Config
	if err := Parse([]byte("android:\n  split_per_abi: sometimes\n"), &c); err == nil {
		t.Fatal("expected error")
	}
	if err := Parse([]byte("build:\n  dart_define: [NOEQUALS]\n"), &c); err == nil {
		t.Fatal("expected error")
	}
}

func TestNestedEnvDefaultsAndDollarValues(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fpack.yaml")
	os.WriteFile(p, []byte("android:\n  signing:\n    store_password: ${KS:-x}\n    key_password: ${KEY_PW:-${KS}}\n    key_alias: ${A:-${B:-upload}}\n"), 0o644)
	// the password contains "${" and must be taken literally
	c, err := Load(p, envOf(map[string]string{"KS": "pa$s${B}w"}))
	if err != nil {
		t.Fatal(err)
	}
	s := c.Android.Signing
	if s.StorePassword != "pa$s${B}w" || s.KeyPassword != "pa$s${B}w" || s.KeyAlias != "upload" || len(c.UnsetEnv) != 0 {
		t.Fatalf("%+v unset=%v", s, c.UnsetEnv)
	}
}

func TestNamePlaceholdersMatchPack(t *testing.T) {
	if !reflect.DeepEqual(NamePlaceholders, pack.Placeholders) {
		t.Fatalf("config %v != pack %v", NamePlaceholders, pack.Placeholders)
	}
}

func TestEnvRefsInTypedKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fpack.yaml")
	os.WriteFile(p, []byte("build:\n  obfuscate: \"${OBF:-false}\"\n  tree_shake_icons: ${TREE}\nandroid:\n  split_per_abi: ${SPLIT}\n  signing:\n    store_password: ${PW}\nweb:\n  optimization_level: ${OPT:-2}\n"), 0o644)
	env := map[string]string{"SPLIT": "both", "TREE": "true", "PW": "a: b #c ${X}"}
	c, err := Load(p, func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if c.Build.Obfuscate == nil || *c.Build.Obfuscate || c.Build.TreeShakeIcons == nil || !*c.Build.TreeShakeIcons {
		t.Fatalf("bools: %+v", c.Build)
	}
	if string(c.Android.SplitPerABI) != "both" || c.Android.Signing.StorePassword != "a: b #c ${X}" {
		t.Fatalf("android: %q %q", c.Android.SplitPerABI, c.Android.Signing.StorePassword)
	}
	if c.Web.OptimizationLevel == nil || *c.Web.OptimizationLevel != 2 {
		t.Fatalf("web: %v", c.Web.OptimizationLevel)
	}
	// A real type error still reports the original line.
	os.WriteFile(p, []byte("build:\n  obfuscate: ${OBF}\n  mode: [x]\n"), 0o644)
	if _, err := Load(p, func(string) string { return "" }); err == nil || !strings.Contains(err.Error(), "line") {
		t.Fatal(err)
	}
}
