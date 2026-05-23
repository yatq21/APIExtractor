package core

import "testing"

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d %#v, want %d %#v", len(got), got, len(want), want)
	}

	items := make(map[string]struct{}, len(got))
	for _, item := range got {
		items[item] = struct{}{}
	}
	for _, item := range want {
		if _, exists := items[item]; !exists {
			t.Fatalf("missing %q in %#v", item, got)
		}
	}
}
