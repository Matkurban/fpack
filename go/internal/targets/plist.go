package targets

import (
	"fmt"
	"sort"
	"strings"
)

// PlistXML renders a property list (string, bool, numbers, lists, maps).
func PlistXML(root map[string]any) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
`)
	plistValue(&b, root, 0)
	b.WriteString("</plist>\n")
	return b.String()
}

func plistValue(b *strings.Builder, v any, depth int) {
	ind := strings.Repeat("\t", depth)
	switch x := v.(type) {
	case map[string]any:
		b.WriteString(ind + "<dict>\n")
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(ind + "\t<key>" + xmlText(k) + "</key>\n")
			plistValue(b, x[k], depth+1)
		}
		b.WriteString(ind + "</dict>\n")
	case map[string]string:
		m := map[string]any{}
		for k, s := range x {
			m[k] = s
		}
		plistValue(b, m, depth)
	case []any:
		b.WriteString(ind + "<array>\n")
		for _, e := range x {
			plistValue(b, e, depth+1)
		}
		b.WriteString(ind + "</array>\n")
	case []string:
		b.WriteString(ind + "<array>\n")
		for _, e := range x {
			plistValue(b, e, depth+1)
		}
		b.WriteString(ind + "</array>\n")
	case bool:
		if x {
			b.WriteString(ind + "<true/>\n")
		} else {
			b.WriteString(ind + "<false/>\n")
		}
	case int, int64, uint64:
		fmt.Fprintf(b, "%s<integer>%d</integer>\n", ind, x)
	case float64:
		fmt.Fprintf(b, "%s<real>%v</real>\n", ind, x)
	case nil:
		b.WriteString(ind + "<string></string>\n")
	default:
		b.WriteString(ind + "<string>" + xmlText(fmt.Sprint(x)) + "</string>\n")
	}
}
