package doctor

import "testing"

// must fails the test immediately when a setup step returns an error.
func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
