package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/i18n"
)

type kind int

const (
	kBool kind = iota
	kString
	kList
	kOptional // --flag or --flag=value (value defaults to "true")
)

type flagSpec struct {
	names   []string // "--all", "-a"
	kind    kind
	metavar string
	en, zh  string
}

func (f flagSpec) key() string { return strings.TrimLeft(f.names[0], "-") }

// UsageError is a bad invocation (exit code 2).
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func usagef(en, zh string, a ...any) error { return &UsageError{Msg: i18n.F(en, zh, a...)} }

type parsed struct {
	bools map[string]bool
	strs  map[string]string
	lists map[string][]string
	pos   []string
	pass  []string // after "--"
	set   map[string]bool
}

func (p *parsed) has(k string) bool   { return p.set[k] }
func (p *parsed) b(k string) bool     { return p.bools[k] }
func (p *parsed) s(k string) string   { return p.strs[k] }
func (p *parsed) l(k string) []string { return p.lists[k] }

var globalFlags = []flagSpec{
	{names: []string{"--help", "-h"}, kind: kBool, en: "show help", zh: "显示帮助"},
	{names: []string{"--project", "-C"}, kind: kString, metavar: "DIR", en: "Flutter project directory (default: current directory or a parent)", zh: "Flutter 项目目录（默认：当前目录或其上级）"},
	{names: []string{"--config"}, kind: kString, metavar: "FILE", en: "config file (default: fpack.yaml in the project)", zh: "配置文件（默认：项目中的 fpack.yaml）"},
	{names: []string{"--flutter"}, kind: kString, metavar: "SDK", en: "Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)", zh: "Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）"},
	{names: []string{"--lang"}, kind: kString, metavar: "zh|en", en: "output language (default: from locale)", zh: "输出语言（默认：根据系统语言）"},
	{names: []string{"--verbose", "-v"}, kind: kBool, en: "stream full tool output", zh: "输出完整的工具日志"},
	{names: []string{"--json"}, kind: kBool, en: "print a machine-readable JSON result to stdout", zh: "向 stdout 输出机器可读的 JSON 结果"},
	{names: []string{"--no-color"}, kind: kBool, en: "disable colors", zh: "关闭颜色"},
	{names: []string{"--yes", "-y"}, kind: kBool, en: "assume yes / use defaults for questions", zh: "所有提问使用默认值 / 自动确认"},
}

func parse(args []string, specs []flagSpec) (*parsed, error) {
	all := append(append([]flagSpec{}, globalFlags...), specs...)
	byName := map[string]flagSpec{}
	var names []string
	for _, s := range all {
		for _, n := range s.names {
			byName[n] = s
			names = append(names, n)
		}
	}
	p := &parsed{bools: map[string]bool{}, strs: map[string]string{}, lists: map[string][]string{}, set: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			p.pass = append(p.pass, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			p.pos = append(p.pos, a)
			continue
		}
		name, val, hasVal := strings.Cut(a, "=")
		spec, ok := byName[name]
		if !ok {
			msg := i18n.F("unknown option %s", "未知选项 %s", name)
			if sug := config.Suggest(name, names); sug != "" {
				msg += i18n.F(" (did you mean %s?)", "（你是不是想用 %s？）", sug)
			}
			if strings.HasPrefix(name, "--") {
				msg += i18n.S("\nTip: pass extra flutter arguments after `--`, e.g. fpack build apk -- --no-tree-shake-icons", "\n提示：额外的 flutter 参数请放在 `--` 之后，例如：fpack build apk -- --no-tree-shake-icons")
			}
			return nil, &UsageError{Msg: msg}
		}
		k := spec.key()
		p.set[k] = true
		switch spec.kind {
		case kBool:
			if hasVal {
				b, err := config.ParseBool(val)
				if err != nil {
					return nil, usagef("%s: %v", "%s：%v", name, err)
				}
				p.bools[k] = b
			} else {
				p.bools[k] = true
			}
		case kOptional:
			if hasVal {
				p.strs[k] = val
			} else {
				p.strs[k] = "true"
			}
		case kString, kList:
			if !hasVal {
				if i+1 >= len(args) {
					return nil, usagef("%s needs a value", "%s 需要一个值", name)
				}
				i++
				val = args[i]
			}
			if spec.kind == kString {
				p.strs[k] = val
			} else {
				p.lists[k] = append(p.lists[k], val)
			}
		}
	}
	return p, nil
}

func flagHelp(specs []flagSpec) [][2]string {
	var rows [][2]string
	for _, s := range specs {
		n := strings.Join(reverseShortFirst(s.names), ", ")
		if s.metavar != "" {
			if s.kind == kOptional {
				n += "[=" + s.metavar + "]"
			} else {
				n += " " + s.metavar
			}
		}
		rows = append(rows, [2]string{n, i18n.S(s.en, s.zh)})
	}
	return rows
}

func reverseShortFirst(n []string) []string {
	out := append([]string{}, n...)
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) < len(out[j]) && strings.Count(out[i], "-") == 1 })
	return out
}

func fmtRows(rows [][2]string) string {
	w := 0
	for _, r := range rows {
		if len(r[0]) > w {
			w = len(r[0])
		}
	}
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-*s  %s\n", w, r[0], r[1])
	}
	return b.String()
}
