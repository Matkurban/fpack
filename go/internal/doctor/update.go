package doctor

import (
	"context"
	"time"

	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/version"
)

// EnvSkipUpdateCheck turns off the pub.dev version check in doctor (CI).
const EnvSkipUpdateCheck = "FPACK_SKIP_UPDATE_CHECK"

// updateLine returns a doctor line for the pub.dev version check, or nil when
// there is nothing to print (up to date in normal mode, skipped, or offline).
func updateLine(current string, getenv func(string) string, verbose bool, get version.Getter) *Line {
	if getenv != nil {
		switch getenv(EnvSkipUpdateCheck) {
		case "1", "true", "yes", "YES", "TRUE":
			return nil
		}
	}
	if get == nil {
		get = version.HTTPGetter(version.DefaultUpdateTimeout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), version.DefaultUpdateTimeout+500*time.Millisecond)
	defer cancel()
	latest, err := version.FetchLatest(ctx, get)
	if err != nil || latest == "" {
		if verbose {
			return &Line{Info, i18n.S("could not check for updates (offline or timed out)", "无法检查更新（离线或超时）"), ""}
		}
		return nil
	}
	switch version.CompareSemver(current, latest) {
	case -1:
		return &Line{
			Info,
			i18n.F("update available: fpack %s (you have %s)", "有新版本：fpack %s（当前 %s）", latest, current),
			i18n.F("dart pub global activate fpack %s", "dart pub global activate fpack %s", latest),
		}
	case 0:
		if verbose {
			return &Line{OK, i18n.F("fpack %s is up to date", "fpack %s 已是最新", current), ""}
		}
		return nil
	default: // local newer than pub (dev / unreleased)
		if verbose {
			return &Line{Info, i18n.F("fpack %s is newer than pub.dev %s", "fpack %s 新于 pub.dev 上的 %s", current, latest), ""}
		}
		return nil
	}
}
