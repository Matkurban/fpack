package config

import (
	"strings"
	"testing"
)

func TestWindowsSignOneCertificateSource(t *testing.T) {
	c := &Config{}
	c.Windows.Sign.Certificate = "a.pfx"
	c.Windows.Sign.Subject = "XueHua"
	if p := strings.Join(c.Validate(), "\n"); !strings.Contains(p, "certificate + subject") {
		t.Fatal(p)
	}
	c.Windows.Sign.Subject = ""
	for _, p := range c.Validate() {
		if strings.Contains(p, "windows.sign") {
			t.Fatal(p)
		}
	}
}
