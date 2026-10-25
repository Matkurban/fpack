package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// PubPackageAPI is the pub.dev endpoint that returns the latest fpack version.
const PubPackageAPI = "https://pub.dev/api/packages/fpack"

// DefaultUpdateTimeout caps how long doctor waits for pub.dev so CI and
// offline machines are never blocked.
const DefaultUpdateTimeout = 2500 * time.Millisecond

// Getter fetches a URL's body. Tests inject a fake; production uses
// [HTTPGetter].
type Getter func(ctx context.Context, url string) ([]byte, error)

// HTTPGetter returns a Getter that uses a short-lived HTTP client.
func HTTPGetter(timeout time.Duration) Getter {
	if timeout <= 0 {
		timeout = DefaultUpdateTimeout
	}
	client := &http.Client{Timeout: timeout}
	return func(ctx context.Context, url string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "fpack/"+Version+" (doctor update check)")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = res.Body.Close() }()
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("pub.dev HTTP %d", res.StatusCode)
		}
		return body, nil
	}
}

// LatestFromPubJSON extracts latest.version from a pub.dev package API body.
func LatestFromPubJSON(body []byte) (string, error) {
	var doc struct {
		Latest struct {
			Version string `json:"version"`
		} `json:"latest"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	v := strings.TrimSpace(doc.Latest.Version)
	if v == "" {
		return "", fmt.Errorf("pub.dev response has no latest.version")
	}
	return v, nil
}

// FetchLatest returns the latest fpack version published on pub.dev.
func FetchLatest(ctx context.Context, get Getter) (string, error) {
	if get == nil {
		get = HTTPGetter(DefaultUpdateTimeout)
	}
	body, err := get(ctx, PubPackageAPI)
	if err != nil {
		return "", err
	}
	return LatestFromPubJSON(body)
}

// CompareSemver compares dotted release versions (optionally with a leading
// "v"). Pre-release / build suffixes after "-" or "+" are ignored. Returns
// -1 if a < b, 0 if equal, 1 if a > b. Unparseable inputs compare as equal
// so a bad response never claims a false upgrade.
func CompareSemver(a, b string) int {
	pa, oka := parseSemver(a)
	pb, okb := parseSemver(b)
	if !oka || !okb {
		return 0
	}
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(s string) ([3]int, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "v"))
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			break
		}
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
