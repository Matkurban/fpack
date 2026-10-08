package targets

import (
	"fmt"
	"strings"
	"time"
)

// Identifier is the reverse-DNS app id: app.identifier, else detected from
// the project (Linux APPLICATION_ID, Android applicationId, iOS bundle id).
func (c *Context) Identifier() string {
	if id := c.Config.App.Identifier; id != "" {
		return id
	}
	if id := c.Project.Identifier(); id != "" {
		return id
	}
	return "com.example." + strings.ReplaceAll(c.Project.Name, "-", "_")
}

// Description is app.description, else the pubspec description, else the
// display name.
func (c *Context) Description() string {
	for _, s := range []string{c.Config.App.Description, c.Project.Description, c.DisplayName()} {
		if s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " ")); s != "" {
			return s
		}
	}
	return c.Project.Name
}

// Publisher is app.publisher, else the CompanyName of the Windows runner
// (may be empty).
func (c *Context) Publisher() string {
	if p := c.Config.App.Publisher; p != "" {
		return p
	}
	return c.Project.WindowsCompany
}

// Maintainer is "Name <email>" for deb/rpm.
func (c *Context) Maintainer() string {
	if m := c.Config.App.Maintainer; m != "" {
		return m
	}
	if p := c.Publisher(); p != "" {
		if strings.Contains(p, "<") {
			return p
		}
		return p + " <noreply@example.com>"
	}
	return c.PackageName() + " maintainers <noreply@example.com>"
}

// Copyright is app.copyright, else "© <year> <publisher>".
func (c *Context) Copyright() string {
	if s := c.Config.App.Copyright; s != "" {
		return s
	}
	who := c.Publisher()
	if who == "" {
		who = c.DisplayName()
	}
	return fmt.Sprintf("© %d %s", time.Now().Year(), who)
}

// License is app.license, else Proprietary.
func (c *Context) License() string {
	if s := c.Config.App.License; s != "" {
		return s
	}
	return "Proprietary"
}

// SupportURL is app.support_url, else app.homepage.
func (c *Context) SupportURL() string {
	if s := c.Config.App.SupportURL; s != "" {
		return s
	}
	return c.Config.App.Homepage
}

func boolOr(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
