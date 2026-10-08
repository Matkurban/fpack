package host

import "testing"

func TestCanBuild(t *testing.T) {
	cases := []struct {
		os   string
		p    Platform
		want bool
	}{
		{"linux", Android, true}, {"linux", Web, true}, {"linux", Linux, true},
		{"linux", IOS, false}, {"linux", MacOS, false}, {"linux", Windows, false},
		{"darwin", IOS, true}, {"darwin", MacOS, true}, {"darwin", Android, true},
		{"darwin", Windows, false}, {"darwin", Linux, false},
		{"windows", Windows, true}, {"windows", Android, true},
		{"windows", MacOS, false}, {"windows", IOS, false}, {"windows", Linux, false},
	}
	for _, c := range cases {
		got, req := Host{OS: c.os}.CanBuild(c.p)
		if got != c.want {
			t.Errorf("%s/%s: got %v want %v", c.os, c.p, got, c.want)
		}
		if !got && req != RequiredOS(c.p) {
			t.Errorf("%s/%s: required os %q", c.os, c.p, req)
		}
	}
}

func TestInstallCommand(t *testing.T) {
	pk := map[string]string{"brew": "cocoapods", "apt": "rpm", "dnf": "rpm-build", "winget": "JRSoftware.InnoSetup"}
	if got := (Host{OS: "darwin"}).Install(pk); got != "brew install cocoapods" {
		t.Error(got)
	}
	if got := (Host{OS: "linux", Distro: "ubuntu"}).Install(pk); got != "sudo apt-get install -y rpm" {
		t.Error(got)
	}
	if got := (Host{OS: "linux", Distro: "rocky", Like: []string{"rhel", "fedora"}}).Install(pk); got != "sudo dnf install -y rpm-build" {
		t.Error(got)
	}
	if got := (Host{OS: "windows"}).Install(pk); got != "winget install --id JRSoftware.InnoSetup -e" {
		t.Error(got)
	}
}

func TestFlutterArch(t *testing.T) {
	if (Host{Arch: "arm64"}).FlutterArch() != "arm64" || (Host{Arch: "amd64"}).FlutterArch() != "x64" {
		t.Fatal("arch mapping")
	}
}
