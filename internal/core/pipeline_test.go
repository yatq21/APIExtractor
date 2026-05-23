package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"apiextractor/internal/config"
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

	cfg := config.Default()
	cfg.UseBuiltinDictionary = false
	cfg.DictionaryPaths = nil

	originalBuiltin := builtinDictionary
	builtinDictionary = []string{"/api-docs"}
	defer func() {
		builtinDictionary = originalBuiltin
	}()
	cfg.UseBuiltinDictionary = true

	result := Run(server.URL, cfg)
	if len(result.Resources) != 1 {
		t.Fatalf("got %d resources, want 1", len(result.Resources))
	}
	wantCandidate := server.URL + "/api/from-docs"
	if !containsString(result.Candidates, wantCandidate) {
		t.Fatalf("missing candidate %q in %#v", wantCandidate, result.Candidates)
	}
	if result.Summary.ResourceCount != 1 || result.Summary.AnalyzableResourceCount != 1 {
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
	cfg.EnableDirectoryScan = false
	cfg.DictionaryPaths = []string{"missing-dictionary.txt"}

	result := Run(server.URL, cfg)
	if len(result.Errors) != 0 {
		t.Fatalf("expected no dictionary errors when directory scan is disabled, got %#v", result.Errors)
	}
	if len(result.Resources) != 0 {
		t.Fatalf("expected no resources when directory scan is disabled, got %#v", result.Resources)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
