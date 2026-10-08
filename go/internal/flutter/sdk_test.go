package flutter

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fakeSDK(t *testing.T, dir, version string) {
	t.Helper()
	os.MkdirAll(filepath.Join(dir, "bin", "cache"), 0o755)
	os.MkdirAll(filepath.Join(dir, "packages", "flutter_tools"), 0o755)
	os.WriteFile(filepath.Join(dir, "bin", flutterExe()), []byte("#!/bin/sh\n"), 0o755)
	os.WriteFile(filepath.Join(dir, "bin", "cache", "flutter.version.json"), []byte(`{"frameworkVersion":"`+version+`","channel":"stable","dartSdkVersion":"3.13.5"}`), 0o644)
}

func TestLocateExplicitAndFVM(t *testing.T) {
	tmp := t.TempDir()
	sdk := filepath.Join(tmp, "sdk")
	fakeSDK(t, sdk, "3.47.6")
	s, err := Locate(tmp, []Candidate{{Path: filepath.Join(sdk, "bin"), Source: "--flutter"}})
	if err != nil || s.Version != "3.47.6" || s.Source != "--flutter" || s.DartVersion != "3.13.5" {
		t.Fatalf("explicit: %+v %v", s, err)
	}
	if _, err := Locate(tmp, []Candidate{{Path: filepath.Join(tmp, "nope"), Source: "FPACK_FLUTTER"}}); err == nil {
		t.Fatal("explicit invalid path must error")
	}
	if runtime.GOOS == "windows" {
		return
	}
	proj := filepath.Join(tmp, "proj")
	os.MkdirAll(filepath.Join(proj, ".fvm"), 0o755)
	fvmSDK := filepath.Join(tmp, "fvm324")
	fakeSDK(t, fvmSDK, "3.24.0")
	os.Symlink(fvmSDK, filepath.Join(proj, ".fvm", "flutter_sdk"))
	s, err = Locate(proj, nil)
	if err != nil || s.Version != "3.24.0" {
		t.Fatalf("fvm: %+v %v", s, err)
	}
}

func TestAtLeast(t *testing.T) {
	s := &SDK{Version: "3.47.6"}
	if !s.AtLeast(3, 22) || s.AtLeast(3, 50) || !s.AtLeast(3, 47) || (&SDK{}).AtLeast(9, 9) != true {
		t.Fatal("AtLeast")
	}
}
