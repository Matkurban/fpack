package config

import (
	"fmt"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"reflect"
	"strconv"
	"strings"
)

// leafTypes are struct types treated as single values, not sections.
func isSection(t reflect.Type) bool { return t.Kind() == reflect.Struct }

// LeafPaths returns the dotted path of every value field of Config.
func LeafPaths() []string {
	var out []string
	var walk func(prefix string, t reflect.Type)
	walk = func(prefix string, t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := yamlName(f)
			if tag == "" {
				continue
			}
			p := tag
			if prefix != "" {
				p = prefix + "." + tag
			}
			if isSection(f.Type) {
				walk(p, f.Type)
				continue
			}
			out = append(out, p)
		}
	}
	walk("", reflect.TypeOf(Config{}))
	return out
}

func yamlName(f reflect.StructField) string {
	tag := strings.Split(f.Tag.Get("yaml"), ",")[0]
	if tag == "-" || !f.IsExported() {
		return ""
	}
	return tag
}

// field returns the settable value for a dotted path.
func field(c *Config, path string) (reflect.Value, bool) {
	v := reflect.ValueOf(c).Elem()
	for _, part := range strings.Split(path, ".") {
		found := false
		for i := 0; i < v.NumField(); i++ {
			if yamlName(v.Type().Field(i)) == part {
				v = v.Field(i)
				found = true
				break
			}
		}
		if !found {
			return reflect.Value{}, false
		}
	}
	return v, true
}

// FieldType returns the Go type of a dotted path (for the schema).
func FieldType(path string) (reflect.Type, bool) {
	v, ok := field(&Config{}, path)
	if !ok {
		return nil, false
	}
	return v.Type(), true
}

// Get returns the value at a dotted path.
func (c *Config) Get(path string) (any, bool) {
	v, ok := field(c, path)
	if !ok {
		return nil, false
	}
	return v.Interface(), true
}

// IsSet reports whether the value at path differs from its zero value.
func (c *Config) IsSet(path string) bool {
	v, ok := field(c, path)
	return ok && !v.IsZero()
}

// SetString parses s according to the field type at path and stores it
// (used for FPACK_* variables). Lists are comma separated.
func (c *Config) SetString(path, s string) error {
	v, ok := field(c, path)
	if !ok {
		return fmt.Errorf(i18n.S("unknown key %s", "未知键 %s"), path)
	}
	switch x := v.Addr().Interface().(type) {
	case *string:
		*x = s
	case *Scalar:
		*x = Scalar(s)
	case **bool:
		b, err := ParseBool(s)
		if err != nil {
			return err
		}
		*x = &b
	case **int:
		i, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf(i18n.S("expected a number, got %q", "应为数字，实际为 %q"), s)
		}
		*x = &i
	case *int:
		i, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf(i18n.S("expected a number, got %q", "应为数字，实际为 %q"), s)
		}
		*x = i
	case *List:
		*x = splitList(s)
	case *Defines:
		l := splitList(s)
		for _, e := range l {
			if !strings.Contains(e, "=") {
				return fmt.Errorf(i18n.S("entry %q must be KEY=VALUE", "条目 %q 必须是 KEY=VALUE 形式"), e)
			}
		}
		*x = Defines(l)
	case *ABIMode:
		m, err := ParseABIMode(s)
		if err != nil {
			return err
		}
		*x = m
	case *Pair:
		var p Pair
		for _, f := range splitList(strings.ReplaceAll(s, "x", ",")) {
			i, err := strconv.Atoi(f)
			if err != nil {
				return fmt.Errorf(i18n.S("expected two numbers like 600,400, got %q", "应为两个数字（如 600,400），实际为 %q"), s)
			}
			p = append(p, i)
		}
		if len(p) != 2 {
			return fmt.Errorf(i18n.S("expected two numbers like 600,400, got %q", "应为两个数字（如 600,400），实际为 %q"), s)
		}
		*x = p
	case *[]int:
		var out []int
		for _, f := range splitList(s) {
			i, err := strconv.Atoi(f)
			if err != nil {
				return fmt.Errorf(i18n.S("expected numbers, got %q", "应为数字，实际为 %q"), f)
			}
			out = append(out, i)
		}
		*x = out
	default:
		return fmt.Errorf(i18n.S("%s cannot be set from a string", "%s 不能用字符串设置"), path)
	}
	return nil
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
