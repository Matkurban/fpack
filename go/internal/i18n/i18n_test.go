package i18n

import "testing"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDetect(t *testing.T) {
	cases := []struct {
		name     string
		override string
		env      map[string]string
		want     Lang
	}{
		{"override wins", "en", map[string]string{"LANG": "zh_CN.UTF-8"}, EN},
		{"override zh", "zh", map[string]string{"LANG": "en_US.UTF-8"}, ZH},
		{"FPACK_LANG", "", map[string]string{"FPACK_LANG": "zh", "LANG": "en_US.UTF-8"}, ZH},
		{"LC_ALL beats LANG", "", map[string]string{"LC_ALL": "zh_CN.UTF-8", "LANG": "en_US.UTF-8"}, ZH},
		{"LANG zh_TW", "", map[string]string{"LANG": "zh_TW.UTF-8"}, ZH},
		{"LANG en", "", map[string]string{"LANG": "en_GB.UTF-8"}, EN},
		{"real non-zh locale is respected", "", map[string]string{"LANG": "de_DE.UTF-8"}, EN},
	}
	for _, c := range cases {
		if got := Detect(c.override, env(c.env)); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestParse(t *testing.T) {
	for _, v := range []string{"zh", "ZH", "zh_CN", "zh-Hans", "cn", "chinese"} {
		if l, ok := Parse(v); !ok || l != ZH {
			t.Errorf("%q should parse as zh", v)
		}
	}
	if _, ok := Parse("fr"); ok {
		t.Error("fr should not parse")
	}
}

func TestS(t *testing.T) {
	Set(ZH)
	defer Set(EN)
	if S("a", "乙") != "乙" || S("a", "") != "a" || F("%d x", "%d 个", 2) != "2 个" {
		t.Fatal("S/F")
	}
}
