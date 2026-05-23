package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

func TestRunFeedsDiscoveredResourcesIntoExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><body>home</body></html>`))
		case "/api-docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"endpoint":"/api/from-docs"}`))
		case "/api/from-docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	file, err := os.CreateTemp("", "wordlist-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	_, _ = file.WriteString("/api-docs\n")
	_ = file.Close()

	cfg := config.Default()
	cfg.DisableBuiltinWordlist = true
	cfg.WordlistPath = file.Name()
	cfg.EnableSoft404Detection = false

	result := Run(server.URL, cfg)
	if len(result.Resources) < 2 {
		t.Fatalf("got %d resources, want at least entrypoint and api-docs", len(result.Resources))
	}
	wantCandidate := server.URL + "/api/from-docs"
	if !containsCandidate(result.Candidates, wantCandidate) {
		t.Fatalf("missing candidate %q in %#v", wantCandidate, result.Candidates)
	}
	if result.Summary.ResourceCount < 2 || result.Summary.AnalyzedResourceCount < 2 {
		t.Fatalf("unexpected summary: %#v", result.Summary)
	}
}

func TestRunSkipsResourceDiscoveryWhenDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>home</body></html>`))
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.DisableDirectoryScan = true
	cfg.WordlistPath = "missing-dictionary.txt"

	result := Run(server.URL, cfg)
	if len(result.Errors) != 0 {
		t.Fatalf("expected no dictionary errors when directory scan is disabled, got %#v", result.Errors)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("expected only entrypoint resource when directory scan is disabled, got %#v", result.Resources)
	}
}

func containsCandidate(items []model.APICandidate, want string) bool {
	for _, item := range items {
		if item.NormalizedURL == want {
			return true
		}
	}
	return false
}
