package config

import (
	"fmt"
	"strings"
)

func kindLabel(k Key, lang string) string {
	zh := lang == "zh"
	switch k.Kind {
	case KEnum:
		return "`" + strings.Join(k.Enum, "` \\| `") + "`"
	case KSplit:
		return "`false` \\| `true` \\| `both`"
	case KList:
		if zh {
			return "list（或单个字符串）"
		}
		return "list (or one string)"
	case KIntList:
		return "list of int"
	case KMap:
		return "map"
	case KAnyMap:
		return "map (any)"
	case KListMap:
		if zh {
			return "map：目标 → 命令列表"
		}
		return "map: target → commands"
	case KPair:
		return "`[x, y]`"
	case KScalar:
		return "string/number"
	case KDefines:
		if zh {
			return "map 或 `KEY=VALUE` 列表"
		}
		return "map or `KEY=VALUE` list"
	}
	return string(k.Kind)
}

func mdCell(s string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(s)
}

// Markdown renders the key reference tables (one per section).
func Markdown(lang string) string {
	zh := lang == "zh"
	var b strings.Builder
	head := "| Key | Type | Default | Targets | Env / flag | Description |\n| --- | --- | --- | --- | --- | --- |\n"
	if zh {
		head = "| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |\n| --- | --- | --- | --- | --- | --- |\n"
	}
	n := 0
	for _, s := range Sections {
		var rows []string
		for _, k := range Keys {
			parent := ""
			if i := strings.LastIndex(k.Path, "."); i > 0 {
				parent = k.Path[:i]
			}
			if parent != s.Path {
				continue
			}
			def := k.Default.Text(lang)
			if def == "" {
				def = "—"
			} else {
				def = "`" + strings.ReplaceAll(def, "`", "'") + "`"
			}
			var ef []string
			if k.Env != "" {
				ef = append(ef, "`"+k.Env+"`")
			}
			if k.Flag != "" {
				ef = append(ef, "`"+k.Flag+"`")
			}
			desc := k.Doc.Text(lang)
			ex := "`" + strings.ReplaceAll(k.Example, "`", "'") + "`"
			if zh {
				desc += " 示例：" + ex
			} else {
				desc += " Example: " + ex
			}
			targets := k.Targets
			if targets == "all" && zh {
				targets = "全部"
			}
			rows = append(rows, fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |", k.Path, mdCell(kindLabel(k, lang)), mdCell(def), targets, strings.Join(ef, "<br>"), mdCell(desc)))
		}
		if len(rows) == 0 {
			continue
		}
		if !strings.Contains(s.Path, ".") {
			n++
			fmt.Fprintf(&b, "### 2.%d `%s`\n\n%s\n\n", n, s.Path, s.Doc.Text(lang))
		} else {
			fmt.Fprintf(&b, "#### `%s`\n\n%s\n\n", s.Path, s.Doc.Text(lang))
		}
		b.WriteString(head)
		b.WriteString(strings.Join(rows, "\n") + "\n\n")
	}
	return b.String()
}

// EnvMarkdown renders the FPACK_* variables that map to keys.
func EnvMarkdown(lang string) string {
	var b strings.Builder
	if lang == "zh" {
		b.WriteString("| 变量 | 对应 fpack.yaml | 类型 | 说明 |\n| --- | --- | --- | --- |\n")
	} else {
		b.WriteString("| Variable | fpack.yaml key | Type | Description |\n| --- | --- | --- | --- |\n")
	}
	for _, k := range Keys {
		if k.Env == "" {
			continue
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n", k.Env, k.Path, mdCell(kindLabel(k, lang)), mdCell(k.Doc.Text(lang)))
	}
	return b.String()
}

// ReplaceGenerated swaps the text between <!-- BEGIN GENERATED name -->
// and <!-- END GENERATED name --> markers.
func ReplaceGenerated(doc, name, content string) (string, bool) {
	begin, end := "<!-- BEGIN GENERATED "+name+" -->", "<!-- END GENERATED "+name+" -->"
	i, j := strings.Index(doc, begin), strings.Index(doc, end)
	if i < 0 || j < i {
		return doc, false
	}
	return doc[:i+len(begin)] + "\n" + content + doc[j:], true
}
