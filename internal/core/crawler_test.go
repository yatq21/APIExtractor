package core

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"apiextractor/internal/config"
)

// 覆盖 HTML 首层发现：script/modulepreload 应保留，stylesheet 因非文本源码后缀被过滤。
func TestExtractSourceURLs(t *testing.T) {
	html := `
		<script src="/static/app.js"></script>
		<link rel="modulepreload" href="/assets/chunk-vendor.mjs">
		<link rel="manifest" href="/manifest.json">
		<link href="/style.css" rel="stylesheet">
	`

	got := ExtractSourceURLs(html, "https://example.com/index.html", true)
	want := []string{
		"https://example.com/static/app.js",
		"https://example.com/assets/chunk-vendor.mjs",
		"https://example.com/manifest.json",
	}

	assertStringSlice(t, got, want)
}

func TestFetchSourceFilesWithBudgetRecordsLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		switch r.URL.Path {
		case "/app.js":
			_, _ = w.Write([]byte(`import("./chunk.js");//# sourceMappingURL=app.js.map`))
		case "/chunk.js":
			_, _ = w.Write([]byte(`fetch("/api/chunk")`))
		case "/app.js.map":
			_, _ = w.Write([]byte(`{"version":3,"sources":["src/api.ts"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.MaxDepth = 0
	cfg.MaxSourceFiles = 10
	files, hits := FetchSourceFilesWithBudget([]string{server.URL + "/app.js"}, cfg)
	if len(files) != 1 {
		t.Fatalf("expected only root file at depth 0, got %d", len(files))
	}
	if files[0].Depth != 0 || files[0].ParentURL != "" {
		t.Fatalf("expected root depth 0, got %d", files[0].Depth)
	}
	if !slices.Contains(hits, "max_depth_reached") {
		t.Fatalf("expected max_depth_reached, got %#v", hits)
	}

	cfg.MaxDepth = 2
	cfg.MaxSourceFiles = 1
	files, hits = FetchSourceFilesWithBudget([]string{server.URL + "/app.js"}, cfg)
	if len(files) != 1 {
		t.Fatalf("expected max source files limit, got %d", len(files))
	}
	if !slices.Contains(hits, "max_source_files_reached") {
		t.Fatalf("expected max_source_files_reached, got %#v", hits)
	}
}

// 覆盖脚本内递归发现：dynamic import、ES module import 与 sourceMappingURL 的相对路径归一化。
func TestExtractNestedSourceURLs(t *testing.T) {
	source := `
		import("/assets/admin.chunk.js");
		import api from "./api-client.mjs";
		//# sourceMappingURL=app.js.map
	`

	got := ExtractNestedSourceURLs(source, "https://example.com/static/app.js", true)
	want := []string{
		"https://example.com/assets/admin.chunk.js",
		"https://example.com/static/api-client.mjs",
		"https://example.com/static/app.js.map",
	}

	assertStringSlice(t, got, want)
}
