package i18n

import "testing"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func osLang(v string) func(string) string { return func(string) string { return v } }

func TestDetect(t *testing.T) {
	cases := []struct {
		name     string
		goos     string
		override string
		env      map[string]string
		os       string
		want     Lang
	}{
		{"override wins", "linux", "en", map[string]string{"LANG": "zh_CN.UTF-8"}, "", EN},
		{"override zh", "darwin", "zh", map[string]string{"LANG": "en_US.UTF-8"}, "en-US", ZH},
		{"FPACK_LANG", "linux", "", map[string]string{"FPACK_LANG": "zh", "LANG": "en_US.UTF-8"}, "", ZH},
		{"FPACK_LANG beats OS", "darwin", "", map[string]string{"FPACK_LANG": "en"}, "zh-Hans-CN", EN},

		{"linux LC_ALL beats LANG", "linux", "", map[string]string{"LC_ALL": "zh_CN.UTF-8", "LANG": "en_US.UTF-8"}, "", ZH},
		{"linux LANG zh_TW", "linux", "", map[string]string{"LANG": "zh_TW.UTF-8"}, "", ZH},
		{"linux LANG en", "linux", "", map[string]string{"LANG": "en_GB.UTF-8"}, "", EN},
		{"linux other language is English", "linux", "", map[string]string{"LANG": "de_DE.UTF-8"}, "", EN},
		{"linux LANGUAGE list", "linux", "", map[string]string{"LANGUAGE": "zh_CN:en", "LANG": "en_US.UTF-8"}, "", ZH},
		{"linux C skipped", "linux", "", map[string]string{"LC_ALL": "C.UTF-8", "LANG": "zh_CN.UTF-8"}, "", ZH},
		{"linux nothing", "linux", "", nil, "", EN},

		{"mac Chinese UI beats Terminal LANG", "darwin", "", map[string]string{"LANG": "en_US.UTF-8"}, "zh-Hans-CN", ZH},
		{"mac English UI beats zh LANG", "darwin", "", map[string]string{"LANG": "zh_CN.UTF-8"}, "en-CN", EN},
		{"mac explicit LC_ALL wins", "darwin", "", map[string]string{"LC_ALL": "en_US.UTF-8"}, "zh-Hans-CN", EN},
		{"mac LC_MESSAGES zh", "darwin", "", map[string]string{"LC_MESSAGES": "zh_CN.UTF-8"}, "en-US", ZH},
		{"mac zh-Hant", "darwin", "", nil, "zh-Hant-TW", ZH},
		{"mac lookup failed: LANG", "darwin", "", map[string]string{"LANG": "zh_CN.UTF-8"}, "", ZH},
		{"mac other language", "darwin", "", nil, "ja-JP", EN},

		{"windows Chinese UI beats Git Bash LANG", "windows", "", map[string]string{"LANG": "en_US.UTF-8"}, "zh-CN", ZH},
		{"windows English UI", "windows", "", nil, "en-US", EN},
		{"windows zh-HK", "windows", "", nil, "zh-HK", ZH},
		{"windows lookup failed", "windows", "", nil, "", EN},
	}
	for _, c := range cases {
		if got := detect(c.override, env(c.env), c.goos, osLang(c.os)); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestFirstAppleLanguage(t *testing.T) {
	if v := firstAppleLanguage("(\n    \"zh-Hans-CN\",\n    \"en-CN\"\n)"); v != "zh-Hans-CN" {
		t.Fatal(v)
	}
	if v := firstAppleLanguage("(\n    en,\n    \"zh-Hans\"\n)"); v != "en" {
		t.Fatal(v)
	}
	if firstAppleLanguage("") != "" {
		t.Fatal("empty")
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
