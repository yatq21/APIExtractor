package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

func TestBuildResourceURLKeepsOriginPathJoin(t *testing.T) {
	base, err := url.Parse("https://example.com/app/index.html")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := buildResourceURL("/api", base, true)
	if !ok {
		t.Fatal("expected resource URL to be built")
	}
	want := "https://example.com/api"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCollectResourceSourceURLsFromHTML(t *testing.T) {
	files := []model.SourceFile{
		{
			URL:        "https://example.com/admin",
			SourceType: "html",
			Content:    `<script src="/static/admin.js"></script>`,
		},
	}

	got := collectResourceSourceURLs(files, true)
	want := []string{"https://example.com/static/admin.js"}
	assertStringSlice(t, got, want)
}

func TestBuildResourceURLRejectsCrossOriginWhenEnabled(t *testing.T) {
	base, err := url.Parse("https://example.com/app/index.html")
	if err != nil {
		t.Fatal(err)
	}

	_, ok := buildResourceURL("https://other.example.com/api", base, true)
	if ok {
		t.Fatal("expected cross-origin dictionary entry to be rejected")
	}
}

func TestDetectResourceType(t *testing.T) {
	cases := map[string]string{
		"https://example.com/static/app.js":        "javascript",
		"https://example.com/static/user.chunk.js": "chunk_js",
		"https://example.com/app.js.map":           "source_map",
		"https://example.com/v3/api-docs":          "api_doc",
		"https://example.com/robots.txt":           "text",
	}

	for rawURL, want := range cases {
		got := DetectResourceType(rawURL, "")
		if got != want {
			t.Fatalf("DetectResourceType(%q) = %q, want %q", rawURL, got, want)
		}
	}
}

func TestDiscoverResourcesReturnsAnalyzableContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api-docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"url":"/api/users"}`))
		case "/missing":
			http.NotFound(w, r)
		default:
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<script src="/app.js"></script>`))
		}
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.UseBuiltinDictionary = false
	cfg.MaxDirectoryScanEntries = 10

	records, files := DiscoverResources(server.URL, []string{"/api-docs", "/missing"}, cfg)
	if len(records) != 1 {
		t.Fatalf("got %d resource records, want 1", len(records))
	}
	if records[0].ResourceType != "api_doc" || !records[0].ShouldAnalyze {
		t.Fatalf("unexpected record: %#v", records[0])
	}
	if len(files) != 1 {
		t.Fatalf("got %d source files, want 1", len(files))
	}
	if files[0].Content == "" {
		t.Fatal("expected analyzable resource content to be retained")
	}
}
