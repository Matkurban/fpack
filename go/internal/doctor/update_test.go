package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Matkurban/fpack/go/internal/i18n"
)

func TestUpdateLineOutdated(t *testing.T) {
	i18n.Set(i18n.EN)
	get := func(context.Context, string) ([]byte, error) {
		return []byte(`{"latest":{"version":"1.2.0"}}`), nil
	}
	l := updateLine("1.1.17", nil, false, get)
	if l == nil || l.Level != Info || l.Fix == "" || l.Text == "" {
		t.Fatalf("%+v", l)
	}
	if !containsAll(l.Text, "1.2.0", "1.1.17") || !containsAll(l.Fix, "dart pub global activate fpack", "1.2.0") {
		t.Fatalf("text=%q fix=%q", l.Text, l.Fix)
	}
}

func TestUpdateLineUpToDate(t *testing.T) {
	get := func(context.Context, string) ([]byte, error) {
		return []byte(`{"latest":{"version":"1.1.17"}}`), nil
	}
	if l := updateLine("1.1.17", nil, false, get); l != nil {
		t.Fatalf("normal mode should be silent: %+v", l)
	}
	l := updateLine("1.1.17", nil, true, get)
	if l == nil || l.Level != OK {
		t.Fatalf("verbose: %+v", l)
	}
}

func TestUpdateLineLocalNewer(t *testing.T) {
	get := func(context.Context, string) ([]byte, error) {
		return []byte(`{"latest":{"version":"1.0.0"}}`), nil
	}
	if l := updateLine("1.1.17", nil, false, get); l != nil {
		t.Fatalf("silent: %+v", l)
	}
	if l := updateLine("1.1.17", nil, true, get); l == nil || l.Level != Info {
		t.Fatalf("verbose: %+v", l)
	}
}

func TestUpdateLineError(t *testing.T) {
	get := func(context.Context, string) ([]byte, error) { return nil, errors.New("timeout") }
	if l := updateLine("1.1.17", nil, false, get); l != nil {
		t.Fatalf("silent on error: %+v", l)
	}
	if l := updateLine("1.1.17", nil, true, get); l == nil || l.Level != Info {
		t.Fatalf("verbose soft note: %+v", l)
	}
}

func TestUpdateLineSkipEnv(t *testing.T) {
	called := false
	get := func(context.Context, string) ([]byte, error) {
		called = true
		return []byte(`{"latest":{"version":"9.0.0"}}`), nil
	}
	env := func(k string) string {
		if k == EnvSkipUpdateCheck {
			return "1"
		}
		return ""
	}
	if l := updateLine("1.0.0", env, true, get); l != nil || called {
		t.Fatalf("skip: line=%+v called=%v", l, called)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
