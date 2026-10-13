package targets

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/runner"
)

// ---- property lists ----

// ParsePlist decodes an XML property list into Go values: map[string]any,
// []any, string, bool, int64, float64 and time.Time (<data> stays a string).
func ParsePlist(data []byte) (any, error) {
	if i := bytes.Index(data, []byte("<?xml")); i > 0 {
		data = data[i:] // tools may print warnings first
	}
	if i := bytes.LastIndex(data, []byte("</plist>")); i >= 0 {
		data = data[:i+len("</plist>")]
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("plist: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local != "plist" {
			return plistElem(d, se)
		}
	}
}

func plistElem(d *xml.Decoder, se xml.StartElement) (any, error) {
	switch se.Name.Local {
	case "dict":
		m := map[string]any{}
		key := ""
		for {
			tok, err := d.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.EndElement:
				return m, nil
			case xml.StartElement:
				if t.Name.Local == "key" {
					if err := d.DecodeElement(&key, &t); err != nil {
						return nil, err
					}
					continue
				}
				v, err := plistElem(d, t)
				if err != nil {
					return nil, err
				}
				m[key] = v
			}
		}
	case "array":
		var a []any
		for {
			tok, err := d.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.EndElement:
				return a, nil
			case xml.StartElement:
				v, err := plistElem(d, t)
				if err != nil {
					return nil, err
				}
				a = append(a, v)
			}
		}
	case "true", "false":
		if err := d.Skip(); err != nil {
			return nil, err
		}
		return se.Name.Local == "true", nil
	}
	var s string
	if err := d.DecodeElement(&s, &se); err != nil {
		return nil, err
	}
	s = strings.TrimSpace(s)
	switch se.Name.Local {
	case "integer":
		return strconv.ParseInt(s, 10, 64)
	case "real":
		return strconv.ParseFloat(s, 64)
	case "date":
		return time.Parse(time.RFC3339, s)
	}
	return s, nil
}

func plistDict(data []byte) (map[string]any, error) {
	v, err := ParsePlist(data)
	if errors.Is(err, io.EOF) {
		return map[string]any{}, nil // <plist/>
	}
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("plist: top level is not a dictionary")
	}
	return m, nil
}

// ---- provisioning profiles ----

// Profile is the part of a provisioning profile fpack checks.
type Profile struct {
	Path         string
	Name         string
	TeamID       string
	AppID        string // application-identifier, e.g. TEAM.com.example.app or TEAM.*
	AllDevices   bool   // ProvisionsAllDevices: Developer ID
	Devices      int    // ProvisionedDevices: development / ad hoc
	Expires      time.Time
	Entitlements map[string]any
	Certs        []string // SHA-1 (upper-case hex) of the DeveloperCertificates
}

// Development reports a development (or ad hoc) profile.
func (p *Profile) Development() bool {
	return p.Devices > 0 || p.Entitlements["com.apple.developer.aps-environment"] == "development"
}

// LoadProfile decodes a .provisionprofile with `security cms -D -i`.
func LoadProfile(t Tools, path string) (*Profile, error) {
	out, ok := t.Probe("security", "cms", "-D", "-i", path)
	if !ok || !strings.Contains(out, "<plist") {
		return nil, fmt.Errorf("%s", i18n.F("cannot read provisioning profile %s (security cms -D -i)", "无法读取描述文件 %s（security cms -D -i）", path))
	}
	m, err := plistDict([]byte(out))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	p := &Profile{Path: path}
	p.Name, _ = m["Name"].(string)
	if teams, ok := m["TeamIdentifier"].([]any); ok && len(teams) > 0 {
		p.TeamID, _ = teams[0].(string)
	}
	p.AllDevices, _ = m["ProvisionsAllDevices"].(bool)
	if devs, ok := m["ProvisionedDevices"].([]any); ok {
		p.Devices = len(devs)
	}
	p.Expires, _ = m["ExpirationDate"].(time.Time)
	p.Entitlements, _ = m["Entitlements"].(map[string]any)
	if p.Entitlements == nil {
		p.Entitlements = map[string]any{}
	}
	if certs, ok := m["DeveloperCertificates"].([]any); ok {
		for _, c := range certs {
			if s, ok := c.(string); ok {
				if der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), "")); err == nil {
					p.Certs = append(p.Certs, fmt.Sprintf("%X", sha1.Sum(der))) //nolint:gosec // certificate fingerprint, not a security use
				}
			}
		}
	}
	for _, k := range []string{"com.apple.application-identifier", "application-identifier"} {
		if s, ok := p.Entitlements[k].(string); ok {
			p.AppID = s
			break
		}
	}
	return p, nil
}

// restrictedEntitlement reports entitlements that only work when a
// provisioning profile authorizes them.
func restrictedEntitlement(key string) bool {
	return strings.HasPrefix(key, "com.apple.developer.") || key == "keychain-access-groups" ||
		key == "com.apple.application-identifier" || key == "application-identifier"
}

// RestrictedKeys lists the restricted entitlements in ents, sorted.
func RestrictedKeys(ents map[string]any) []string {
	var out []string
	for k := range ents {
		if restrictedEntitlement(k) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// wildcardMatch matches s against a profile value that may end in "*"
// (TEAM.* or TEAM.com.example.*).
func wildcardMatch(pattern, s string) bool {
	if p, ok := strings.CutSuffix(pattern, "*"); ok {
		return strings.HasPrefix(s, p)
	}
	return pattern == s
}

var teamInIdentity = regexp.MustCompile(`\(([A-Z0-9]{10})\)\s*$`)

// identityTeam returns the team ID in "Developer ID Application: Name (TEAMID)".
func identityTeam(identity string) string {
	if m := teamInIdentity.FindStringSubmatch(identity); m != nil {
		return m[1]
	}
	return ""
}

// Signer is the signing certificate a profile must match.
type Signer struct {
	Team string // team ID ("" skips the check)
	SHA1 string // certificate fingerprint ("" skips the check)
}

// resolveSigner finds the team and certificate fingerprint of a codesign
// identity (a name or a SHA-1) in the keychain.
func resolveSigner(t Tools, identity string) Signer {
	s := Signer{Team: identityTeam(identity)}
	out, _ := t.Probe("security", "find-identity", "-v", "-p", "codesigning")
	var hits []string
	for _, line := range strings.Split(out, "\n") {
		m := identityLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if strings.EqualFold(m[1], identity) {
			if s.Team == "" {
				s.Team = identityTeam(m[2])
			}
			return Signer{Team: s.Team, SHA1: m[1]}
		}
		if m[2] == identity && !slices.Contains(hits, m[1]) {
			hits = append(hits, m[1])
		}
	}
	if len(hits) == 1 {
		s.SHA1 = hits[0]
	}
	return s
}

// CheckProfile validates a profile for Developer ID distribution of
// bundleID signed by signer with the entitlements ents. Empty fields skip
// those checks. It returns one message per problem.
func CheckProfile(p *Profile, signer Signer, bundleID string, ents map[string]any, now time.Time) []string {
	var out []string
	name := filepath.Base(p.Path)
	team := signer.Team
	switch {
	case p.Devices > 0:
		out = append(out, i18n.F("%s is a development/ad hoc profile (%d registered devices); Developer ID apps need a \"Developer ID\" profile", "%s 是开发/Ad Hoc 描述文件（登记了 %d 台设备）；Developer ID App 需要 “Developer ID” 类型的描述文件", name, p.Devices))
	case !p.AllDevices:
		out = append(out, i18n.F("%s is not a Developer ID profile (ProvisionsAllDevices is missing); App Store profiles cannot be used outside the App Store", "%s 不是 Developer ID 描述文件（缺少 ProvisionsAllDevices）；App Store 描述文件不能在 App Store 之外使用", name))
	}
	if team != "" && p.TeamID != team {
		out = append(out, i18n.F("%s belongs to team %s, but the signing certificate is from team %s", "%s 属于团队 %s，但签名证书属于团队 %s", name, p.TeamID, team))
	}
	if signer.SHA1 != "" && len(p.Certs) > 0 && !slices.Contains(p.Certs, strings.ToUpper(signer.SHA1)) {
		out = append(out, i18n.F("%s was generated for another Developer ID Application certificate (%s), not the signing certificate %s; macOS will refuse to launch the app (\"Unsatisfied entitlements\"): edit the profile in the developer portal, select this certificate and download it again", "%s 是为另一张 Developer ID Application 证书（%s）生成的，而不是签名证书 %s；macOS 会拒绝启动 App（“Unsatisfied entitlements”）：请在开发者网站编辑该描述文件，选择此证书后重新下载", name, short(p.Certs), short([]string{signer.SHA1})))
	}
	if bundleID != "" && p.AppID != "" && !wildcardMatch(p.AppID, p.TeamID+"."+bundleID) {
		out = append(out, i18n.F("%s is for app id %s, not %s.%s", "%s 对应的 App ID 是 %s，而不是 %s.%s", name, p.AppID, p.TeamID, bundleID))
	}
	if !p.Expires.IsZero() && now.After(p.Expires) {
		out = append(out, i18n.F("%s expired on %s", "%s 已于 %s 过期", name, p.Expires.Format("2006-01-02")))
	}
	for _, k := range RestrictedKeys(ents) {
		if k == "com.apple.application-identifier" || k == "application-identifier" {
			continue // checked as the app id above
		}
		have, ok := p.Entitlements[k]
		if !ok {
			out = append(out, i18n.F("entitlement %s is not authorized by %s (enable the capability for the app id in the developer portal and download the profile again)", "%[2]s 未授权 entitlement %[1]s（在开发者网站为该 App ID 启用对应功能后重新下载描述文件）", k, name))
			continue
		}
		if msg := entitlementMismatch(k, ents[k], have); msg != "" {
			out = append(out, msg)
		}
	}
	return out
}

// short abbreviates certificate fingerprints for messages.
func short(certs []string) string {
	out := make([]string, len(certs))
	for i, c := range certs {
		out[i] = strings.ToUpper(c)
		if len(c) > 8 {
			out[i] = out[i][:8] + "…"
		}
	}
	return strings.Join(out, ", ")
}

// entitlementMismatch compares a signing entitlement value with the value
// a profile authorizes ("" when allowed).
func entitlementMismatch(key string, want, have any) string {
	str := func(v any) []string {
		switch x := v.(type) {
		case string:
			return []string{x}
		case []any:
			var out []string
			for _, e := range x {
				if s, ok := e.(string); ok {
					out = append(out, s)
				}
			}
			return out
		}
		return nil
	}
	if b, ok := want.(bool); ok {
		if hb, ok := have.(bool); b && ok && !hb {
			return i18n.F("entitlement %s is true but the profile sets it to false", "entitlement %s 为 true，但描述文件中为 false", key)
		}
		return ""
	}
	allowed := str(have)
	for _, w := range str(want) {
		ok := false
		for _, a := range allowed {
			if wildcardMatch(a, w) {
				ok = true
				break
			}
		}
		if !ok {
			if key == "com.apple.developer.aps-environment" {
				return i18n.F("aps-environment is %q in the entitlements but the profile allows %q: Developer ID apps use \"production\" (set it in the release entitlements)", "entitlements 中 aps-environment 为 %q，但描述文件只允许 %q：Developer ID App 应使用 \"production\"（在发布用 entitlements 中修改）", w, strings.Join(allowed, ", "))
			}
			return i18n.F("entitlement %s value %q is not allowed by the profile (allowed: %s)", "描述文件不允许 entitlement %s 的值 %q（允许：%s）", key, w, strings.Join(allowed, ", "))
		}
	}
	return ""
}

// readEntitlements reads an entitlements .plist file (nil when path is "").
func readEntitlements(path string) (map[string]any, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return plistDict(b)
}

func (c *Context) profileCheck() bool {
	p := c.Config.MacOS.Sign.ProfileCheck
	return p == nil || *p
}

func (c *Context) mainProfile() string {
	if p := c.Config.MacOS.Sign.ProvisioningProfile; p != "" {
		return c.Project.Abs(p)
	}
	return ""
}

func profileFix() string {
	return i18n.S("create a \"Developer ID\" provisioning profile for this app id (developer.apple.com → Profiles → + → Developer ID), download it and set macos.sign.provisioning_profile (or --provisioning-profile / FPACK_MACOS_PROVISIONING_PROFILE); macos.sign.profile_check: false skips this check",
		"为该 App ID 创建 “Developer ID” 描述文件（developer.apple.com → Profiles → + → Developer ID），下载后设置 macos.sign.provisioning_profile（或 --provisioning-profile / FPACK_MACOS_PROVISIONING_PROFILE）；macos.sign.profile_check: false 可跳过此检查")
}

// profilePreflight checks the configured profiles against the project
// (before the long Xcode build) and refuses restricted entitlements
// without a profile.
func profilePreflight(c *Context) []Issue {
	if !c.Mac.Enabled || !c.profileCheck() {
		return nil
	}
	var out []Issue
	ents, err := readEntitlements(c.entitlements())
	if err != nil {
		return []Issue{fatal(i18n.F("cannot read entitlements %s: %v", "无法读取 entitlements %s：%v", c.Rel(c.entitlements()), err), "")}
	}
	signer := resolveSigner(c.Tools, c.Mac.Identity)
	bundle := c.Project.MacBundleID
	if strings.Contains(bundle, "$") {
		bundle = ""
	}
	if p := c.mainProfile(); p != "" {
		prof, err := LoadProfile(c.Tools, p)
		if err != nil {
			return append(out, fatal(err.Error(), profileFix()))
		}
		for _, m := range CheckProfile(prof, signer, bundle, ents, time.Now()) {
			out = append(out, fatal(m, profileFix()))
		}
	} else if r := RestrictedKeys(ents); len(r) > 0 {
		out = append(out, fatal(i18n.F("%s uses restricted entitlements (%s) that a Developer ID app may only use with a provisioning profile; without one the signed app will not launch (RBSRequestErrorDomain Code=5)", "%s 使用了受限 entitlements（%s），Developer ID App 必须配合描述文件才能使用；否则签名后的 App 无法启动（RBSRequestErrorDomain Code=5）", c.Rel(c.entitlements()), strings.Join(r, ", ")), profileFix()))
	}
	for id, p := range c.Config.MacOS.Sign.ProvisioningProfiles {
		prof, err := LoadProfile(c.Tools, c.Project.Abs(p))
		if err != nil {
			out = append(out, fatal(err.Error(), profileFix()))
			continue
		}
		for _, m := range CheckProfile(prof, signer, id, nil, time.Now()) {
			out = append(out, fatal(m, profileFix()))
		}
	}
	// An app built earlier shows what Xcode embeds (doctor, dry runs).
	if c.mainProfile() == "" {
		if in, err := locateMacApp(c, false); err == nil {
			if prof, err := LoadProfile(c.Tools, filepath.Join(in["app"], "Contents", "embedded.provisionprofile")); err == nil && prof.Development() {
				out = append(out, Issue{Msg: i18n.F("%s embeds Xcode's development profile %q; a Developer ID signed copy will not launch", "%s 嵌入了 Xcode 的开发描述文件 %q；Developer ID 签名后将无法启动", c.Rel(in["app"]), prof.Name), Fix: profileFix(), NotReady: true})
			}
		}
	}
	return out
}

// nestedBundles lists app extensions, login items and system extensions
// inside app (relative paths, sorted).
func nestedBundles(app string) []string {
	var out []string
	for _, pat := range []string{"Contents/PlugIns/*.appex", "Contents/Library/LoginItems/*.app", "Contents/Library/SystemExtensions/*.systemextension", "Contents/Helpers/*.app"} {
		m, _ := filepath.Glob(filepath.Join(app, filepath.FromSlash(pat)))
		for _, p := range m {
			if r, err := filepath.Rel(app, p); err == nil {
				out = append(out, r)
			}
		}
	}
	sort.Strings(out)
	return out
}

func bundleInfo(t Tools, bundle, key string) string {
	out, ok := t.Probe("plutil", "-extract", key, "raw", "-o", "-", filepath.Join(bundle, "Contents", "Info.plist"))
	if !ok {
		return ""
	}
	return strings.TrimSpace(out)
}

// signedEntitlements reads the entitlements of a signed bundle.
func signedEntitlements(t Tools, bundle string) (map[string]any, string) {
	out, ok := t.Probe("codesign", "-d", "--entitlements", "-", "--xml", bundle)
	if !ok || !strings.Contains(out, "<plist") {
		return nil, ""
	}
	m, err := plistDict([]byte(out))
	if err != nil {
		return nil, ""
	}
	i := strings.Index(out, "<?xml")
	if i < 0 {
		i = strings.Index(out, "<plist")
	}
	return m, out[i:]
}

// nestedSignature is a nested bundle re-signed with its own entitlements.
type nestedSignature struct {
	Bundle, Entitlements string // absolute; Entitlements "" = none
}

// macProfileOps embeds the provisioning profiles into the staged app (and
// its nested bundles) and records the nested bundles' entitlements, which
// the --deep signing pass drops, so they can be signed again inside-out.
func macProfileOps(c *Context, app string, nested *[]nestedSignature) Op {
	var preview []*runner.Cmd
	if p := c.mainProfile(); p != "" {
		preview = append(preview, cmd("security", "cms", "-D", "-i", p))
	}
	return Op{Desc: i18n.S("embed provisioning profiles and check entitlements", "嵌入描述文件并检查 entitlements"),
		Preview: preview,
		Hint:    profileFix(),
		Run: func(_ context.Context, env OpEnv) (string, error) {
			return "", embedProfiles(c, app, nested, env)
		}}
}

func embedProfiles(c *Context, app string, nested *[]nestedSignature, env OpEnv) error {
	check := c.profileCheck()
	signer := resolveSigner(c.Tools, c.Mac.Identity)
	var problems []string
	embed := func(src, bundle string) error {
		dst := filepath.Join(bundle, "Contents", "embedded.provisionprofile")
		if err := pack.CopyFile(src, dst); err != nil {
			return err
		}
		return os.Chmod(dst, 0o644)
	}
	// main app
	ents, err := readEntitlements(c.entitlements())
	if err != nil {
		return err
	}
	if p := c.mainProfile(); p != "" {
		prof, err := LoadProfile(c.Tools, p)
		if err != nil {
			return err
		}
		if check {
			problems = append(problems, CheckProfile(prof, signer, bundleInfo(c.Tools, app, "CFBundleIdentifier"), ents, time.Now())...)
		}
		if err := embed(p, app); err != nil {
			return err
		}
		env.Print("info", i18n.F("embedded %s (%s)", "已嵌入 %s（%s）", filepath.Base(p), prof.Name))
	} else if check {
		if prof, err := LoadProfile(c.Tools, filepath.Join(app, "Contents", "embedded.provisionprofile")); err == nil && prof.Development() {
			problems = append(problems, i18n.F("the app embeds Xcode's development profile %q, which Developer ID signed apps cannot use", "App 嵌入了 Xcode 的开发描述文件 %q，Developer ID 签名的 App 不能使用它", prof.Name))
		}
	}
	// nested bundles
	*nested = nil
	for i, rel := range nestedBundles(app) {
		b := filepath.Join(app, rel)
		id := bundleInfo(c.Tools, b, "CFBundleIdentifier")
		nents, raw := signedEntitlements(c.Tools, b)
		sig := nestedSignature{Bundle: b}
		if raw != "" {
			sig.Entitlements = filepath.Join(filepath.Dir(app), fmt.Sprintf("nested-%d.entitlements", i))
			if err := os.WriteFile(sig.Entitlements, []byte(raw), 0o644); err != nil {
				return err
			}
		}
		*nested = append(*nested, sig)
		if p, ok := c.Config.MacOS.Sign.ProvisioningProfiles[id]; ok {
			prof, err := LoadProfile(c.Tools, c.Project.Abs(p))
			if err != nil {
				return err
			}
			if check {
				problems = append(problems, CheckProfile(prof, signer, id, nents, time.Now())...)
			}
			if err := embed(c.Project.Abs(p), b); err != nil {
				return err
			}
			env.Print("info", i18n.F("embedded %s into %s", "已将 %s 嵌入 %s", filepath.Base(p), rel))
			continue
		}
		if !check {
			continue
		}
		prof, perr := LoadProfile(c.Tools, filepath.Join(b, "Contents", "embedded.provisionprofile"))
		hasProfile := perr == nil
		if hasProfile && prof.Development() {
			problems = append(problems, i18n.F("%s (%s) embeds a development profile; add it to macos.sign.provisioning_profiles", "%s（%s）嵌入了开发描述文件；请在 macos.sign.provisioning_profiles 中为它配置描述文件", rel, id))
		} else if r := RestrictedKeys(nents); len(r) > 0 && !hasProfile {
			problems = append(problems, i18n.F("%s (%s) uses restricted entitlements (%s) but has no provisioning profile; add it to macos.sign.provisioning_profiles", "%s（%s）使用了受限 entitlements（%s），但没有描述文件；请在 macos.sign.provisioning_profiles 中配置", rel, id, strings.Join(r, ", ")))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

// signNestedOp signs nested bundles again with their own entitlements after
// the --deep pass (which signs them without entitlements).
func signNestedOp(c *Context, nested *[]nestedSignature, rt []string) Op {
	id := c.Mac.IdentityLabel()
	return Op{Desc: i18n.S("sign nested bundles (extensions, login items) with their own entitlements", "用各自的 entitlements 签名内嵌 bundle（扩展、登录项）"),
		Hint: hintIdentity(),
		Run: func(_ context.Context, env OpEnv) (string, error) {
			for _, n := range *nested {
				args := append(append([]string{"--force"}, rt...), "--timestamp")
				if n.Entitlements != "" {
					args = append(args, "--entitlements", n.Entitlements)
				}
				if _, err := env.Run(runner.Cmd{Name: "codesign", Args: append(args, "--sign", id, n.Bundle)}); err != nil {
					return "", err
				}
			}
			return "", nil
		}}
}

// ---- launch smoke test ----

func (c *Context) smokeTest() bool {
	s := c.Config.MacOS.SmokeTest
	return s != nil && *s
}

func (c *Context) smokeSeconds() int {
	if n := c.Config.MacOS.SmokeTestSeconds; n > 0 {
		return n
	}
	return 5
}

// smokeTestOp launches the signed app with LaunchServices (as a user
// would), checks that it is still running after a few seconds and quits it.
func smokeTestOp(c *Context, app string) Op {
	secs := c.smokeSeconds()
	return Op{Desc: i18n.F("launch smoke test (%ds)", "启动测试（%d 秒）", secs),
		Preview: []*runner.Cmd{cmd("open", "-n", app), cmd("pgrep", "-f", app+"/Contents/MacOS/"), cmd("pkill", "-f", app+"/Contents/MacOS/")},
		Hint: i18n.S("look for the reason in Console.app or: log show --last 2m --predicate 'process == \"amfid\" OR process == \"taskgated-helper\" OR subsystem == \"com.apple.runningboard\"'",
			"在“控制台”App 中查找原因，或运行：log show --last 2m --predicate 'process == \"amfid\" OR process == \"taskgated-helper\" OR subsystem == \"com.apple.runningboard\"'"),
		Run: func(ctx context.Context, env OpEnv) (string, error) {
			return runSmokeTest(ctx, c, env, app, time.Duration(secs)*time.Second)
		}}
}

func runSmokeTest(ctx context.Context, c *Context, env OpEnv, app string, wait time.Duration) (string, error) {
	exe := bundleInfo(c.Tools, app, "CFBundleExecutable")
	if exe == "" {
		exe = strings.TrimSuffix(filepath.Base(app), ".app")
	}
	proc := filepath.Join(app, "Contents", "MacOS", exe)
	if _, err := env.Run(runner.Cmd{Name: "open", Args: []string{"-n", app}}); err != nil {
		return "", fmt.Errorf("%s", i18n.F("the app did not launch: %v", "App 未能启动：%v", err))
	}
	select {
	case <-ctx.Done():
		return "", runner.ErrInterrupted
	case <-time.After(wait):
	}
	if _, ok := c.Tools.Probe("pgrep", "-f", proc); !ok {
		return "", fmt.Errorf("%s", i18n.F("the app was not running %s after launch (it crashed or macOS refused to start it)", "启动 %s 后 App 已不在运行（崩溃或被 macOS 拒绝启动）", wait))
	}
	_, _ = c.Tools.Probe("pkill", "-f", proc)
	return i18n.F("launch smoke test passed (running after %s)", "启动测试通过（%s 后仍在运行）", wait), nil
}
