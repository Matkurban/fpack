package config

import (
	"encoding/json"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SchemaURL is where editors fetch the schema (yaml-language-server).
const SchemaURL = "https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json"

type obj = map[string]any

// envRefSchema accepts a ${VAR} / ${VAR:-default} reference.
var envRefSchema = obj{"type": "string", "pattern": `\$\{[A-Za-z_][A-Za-z0-9_]*(:-[^}]*)?\}`}

func keySchema(k Key) obj {
	var s obj
	switch k.Kind {
	case KString, KPath:
		s = obj{"type": "string"}
	case KURL:
		s = obj{"type": "string", "pattern": `^(https?://|\$\{)`}
	case KBool:
		s = obj{"type": "boolean"}
	case KInt:
		s = obj{"type": "integer"}
		switch k.Path {
		case "web.optimization_level":
			s["minimum"], s["maximum"] = 0, 4
		case "macos.dmg.icon_size":
			s["minimum"], s["maximum"] = 16, 512
		}
	case KEnum:
		s = obj{"type": "string", "enum": k.Enum}
	case KList:
		item := obj{"type": "string"}
		switch k.Path {
		case "build.targets":
			// Aliases are accepted like on the command line.
			item = obj{"type": "string", "enum": append(append([]string{}, TargetNames...), TargetAliases...)}
		case "android.abis":
			item = obj{"type": "string", "enum": KnownABIs}
		}
		s = obj{"anyOf": []any{obj{"type": "array", "items": item}, item}}
	case KIntList:
		s = obj{"type": "array", "items": obj{"type": "integer", "minimum": 16, "maximum": 1024}}
	case KMap:
		s = obj{"type": "object", "additionalProperties": obj{"type": []string{"string", "number", "boolean"}}}
		if k.Path == "output.names" {
			s["propertyNames"] = obj{"enum": TargetNames}
		}
	case KAnyMap:
		s = obj{"type": "object"}
	case KListMap:
		cmds := obj{"anyOf": []any{obj{"type": "array", "items": obj{"type": "string"}}, obj{"type": "string"}}}
		props := obj{}
		for _, t := range TargetNames {
			props[t] = cmds
		}
		s = obj{"type": "object", "properties": props, "additionalProperties": false}
	case KPair:
		s = obj{"anyOf": []any{
			obj{"type": "array", "items": obj{"type": "integer"}, "minItems": 2, "maxItems": 2},
			obj{"type": "string", "pattern": `^\d+\s*[,x]\s*\d+$`}}}
	case KScalar:
		s = obj{"type": []string{"string", "number"}}
	case KDefines:
		s = obj{"anyOf": []any{
			obj{"type": "object", "additionalProperties": obj{"type": []string{"string", "number", "boolean"}}},
			obj{"type": "array", "items": obj{"type": "string", "pattern": "^[^=]+=.*$"}}}}
	case KSplit:
		s = obj{"anyOf": []any{obj{"type": "boolean"}, obj{"type": "string", "enum": []string{"true", "false", "both", "universal", "split"}}}}
	default:
		s = obj{}
	}
	// Non-string values may come from the environment: `obfuscate: ${OBF:-false}`.
	switch k.Kind {
	case KBool, KInt, KEnum, KSplit, KIntList, KPair:
		s = obj{"anyOf": []any{s, envRefSchema}}
	}
	desc := k.Doc.EN
	if k.Default.EN != "" {
		desc += " Default: " + k.Default.EN + "."
	}
	if k.Env != "" {
		desc += " Env: " + k.Env + "."
	}
	desc += " Targets: " + k.Targets + "."
	s["description"] = desc
	md := "**" + k.Doc.EN + "**  \n" + k.Doc.ZH
	if k.Default.EN != "" {
		md += "\n\nDefault / 默认: `" + k.Default.Text("zh") + "`"
	}
	md += "\n\nTargets / 目标: " + k.Targets
	if k.Env != "" {
		md += " · Env: `" + k.Env + "`"
	}
	if k.Flag != "" {
		md += " · Flag: `" + k.Flag + "`"
	}
	s["markdownDescription"] = md
	var ex any
	if yaml.Unmarshal([]byte(k.Example), &ex) == nil && ex != nil {
		s["examples"] = []any{normalizeYAML(ex)}
	}
	return s
}

// normalizeYAML converts yaml.v3 maps to JSON-encodable maps.
func normalizeYAML(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = normalizeYAML(e)
		}
		return x
	case map[any]any:
		m := map[string]any{}
		for k, e := range x {
			m[strings.TrimSpace(toString(k))] = normalizeYAML(e)
		}
		return m
	case []any:
		for i := range x {
			x[i] = normalizeYAML(x[i])
		}
		return x
	}
	return v
}

func toString(v any) string {
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), `"`)
}

// Schema returns the JSON schema of fpack.yaml, generated from Keys.
func Schema() ([]byte, error) {
	root := obj{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"$id":                  SchemaURL,
		"title":                "fpack.yaml",
		"description":          "fpack configuration – https://matkurban.github.io/fpack/configuration",
		"type":                 "object",
		"properties":           obj{},
		"additionalProperties": false,
	}
	section := func(path string) obj {
		cur, acc := root, ""
		for _, part := range strings.Split(path, ".") {
			if acc != "" {
				acc += "."
			}
			acc += part
			props := cur["properties"].(obj)
			next, ok := props[part].(obj)
			if !ok {
				next = obj{"type": []string{"object", "null"}, "properties": obj{}, "additionalProperties": false}
				for _, s := range Sections {
					if s.Path == acc {
						next["description"] = s.Doc.EN
						next["markdownDescription"] = s.Doc.EN + "  \n" + s.Doc.ZH
					}
				}
				props[part] = next
			}
			cur = next
		}
		return cur
	}
	for _, k := range Keys {
		parent, name := "", k.Path
		if i := strings.LastIndex(k.Path, "."); i > 0 {
			parent, name = k.Path[:i], k.Path[i+1:]
		}
		target := root
		if parent != "" {
			target = section(parent)
		}
		target["properties"].(obj)[name] = keySchema(k)
	}
	return json.MarshalIndent(root, "", "  ")
}
