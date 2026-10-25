package version

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLatestFromPubJSON(t *testing.T) {
	v, err := LatestFromPubJSON([]byte(`{"latest":{"version":"1.2.3"}}`))
	if err != nil || v != "1.2.3" {
		t.Fatalf("got %q %v", v, err)
	}
	if _, err := LatestFromPubJSON([]byte(`{}`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := LatestFromPubJSON([]byte(`not-json`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.1.16", "1.1.17", -1},
		{"1.1.17", "1.1.17", 0},
		{"1.2.0", "1.1.99", 1},
		{"v1.1.17", "1.1.17", 0},
		{"1.1.17-dev", "1.1.17", 0},
		{"1.1.17+1", "1.1.17", 0},
		{"2.0.0", "1.9.9", 1},
		{"", "1.0.0", 0},
		{"1.0", "1.0.0", 0},
	}
	for _, c := range cases {
		if got := CompareSemver(c.a, c.b); got != c.want {
			t.Errorf("CompareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestFetchLatest(t *testing.T) {
	v, err := FetchLatest(context.Background(), func(_ context.Context, url string) ([]byte, error) {
		if url != PubPackageAPI {
			t.Errorf("url %s", url)
		}
		return []byte(`{"latest":{"version":"9.9.9"}}`), nil
	})
	if err != nil || v != "9.9.9" {
		t.Fatalf("%q %v", v, err)
	}
	_, err = FetchLatest(context.Background(), func(context.Context, string) ([]byte, error) {
		return nil, errors.New("offline")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, err = FetchLatest(ctx, func(ctx context.Context, _ string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
