package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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

func TestRunIncludesFrontendRecognitionInResourcesAndCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<div id="__next"></div><script src="/_next/static/chunks/main-app.js"></script>`))
		case "/_next/static/chunks/main-app.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`__webpack_require__.u=function(id){return "chunk-"+id+".js"};fetch("/api/next-data");`))
		case "/api/next-data":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.DisableDirectoryScan = true

	result := Run(server.URL, cfg)
	if len(result.Resources) < 2 {
		t.Fatalf("expected entrypoint and source resources, got %#v", result.Resources)
	}
	if result.Resources[0].Frontend == nil || !slices.Contains(result.Resources[0].Frontend.Frameworks, "nextjs") {
		t.Fatalf("expected nextjs frontend info on entrypoint, got %#v", result.Resources[0].Frontend)
	}
	foundCandidate := false
	for _, item := range result.Candidates {
		if item.Path == "/api/next-data" {
			foundCandidate = true
			if !slices.Contains(item.Tags, "framework:nextjs") || !slices.Contains(item.Tags, "build:webpack") {
				t.Fatalf("expected frontend tags on candidate, got %#v", item.Tags)
			}
		}
	}
	if !foundCandidate {
		t.Fatalf("missing /api/next-data candidate in %#v", result.Candidates)
	}
}

func TestRunDiscoversManifestLinkAndTracksSourceBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<link rel="manifest" href="/manifest.json"><script src="/app.js"></script>`))
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"start_url":"/api/mobile/bootstrap","src":"/assets/entry.js"}`))
		case "/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`import("./chunk.js");`))
		case "/assets/entry.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`fetch("/api/from-manifest-entry")`))
		case "/chunk.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`fetch("/api/chunk")`))
		case "/api/mobile/bootstrap", "/api/from-manifest-entry", "/api/chunk":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.DisableDirectoryScan = true
	cfg.MaxDepth = 0
	cfg.MaxSourceFiles = 10

	result := Run(server.URL, cfg)
	if !containsResource(result.Resources, server.URL+"/manifest.json") {
		t.Fatalf("expected manifest resource in %#v", result.Resources)
	}
	if !containsCandidate(result.Candidates, server.URL+"/api/mobile/bootstrap") {
		t.Fatalf("expected manifest API candidate in %#v", result.Candidates)
	}
	if !slices.Contains(result.BudgetHits, "max_depth_reached") || result.Summary.BudgetHitCount == 0 {
		t.Fatalf("expected max depth budget hit, got hits=%#v summary=%#v", result.BudgetHits, result.Summary)
	}
}

func TestScanDirectoryResourcesReportsBudgetHit(t *testing.T) {
	cfg := config.Default()
	cfg.MaxResources = 1
	resources, hits := ScanDirectoryResources([]string{"http://127.0.0.1:1/a", "http://127.0.0.1:1/b"}, cfg)

	if len(resources) != 0 {
		t.Fatalf("expected no reachable resources, got %#v", resources)
	}
	if !slices.Contains(hits, "max_resources_reached") {
		t.Fatalf("expected max_resources_reached, got %#v", hits)
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

func containsResource(items []model.ResourceRecord, want string) bool {
	for _, item := range items {
		if item.URL == want {
			return true
		}
	}
	return false
}
