package ui

import "strings"

// Table prints rows as aligned columns. Cells may contain ANSI codes.
func (u *UI) Table(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = DisplayWidth(h)
	}
	for _, r := range rows {
		for i := range headers {
			if i < len(r) {
				if w := DisplayWidth(StripANSI(r[i])); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}
	line := func(cells []string, style func(string) string) string {
		var parts []string
		for i := range headers {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			if i == len(headers)-1 {
				parts = append(parts, c)
			} else {
				parts = append(parts, Pad(c, widths[i]))
			}
		}
		s := "  " + strings.TrimRight(strings.Join(parts, "  "), " ")
		if style != nil {
			return style(s)
		}
		return s
	}
	u.Println(line(headers, u.Bold))
	for _, r := range rows {
		u.Println(line(r, nil))
	}
}
