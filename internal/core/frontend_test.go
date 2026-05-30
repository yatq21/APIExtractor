package core

import (
	"slices"
	"strings"
	"testing"

	"apiextractor/internal/model"
)

func TestDetectFrontendFromHTMLAndURLs(t *testing.T) {
	html := `<div id="__next"></div><script src="/_next/static/chunks/webpack.js"></script>`
	info := DetectFrontendFromHTML(html, []string{"https://example.com/_next/static/chunks/main-app.js"})

	if !slices.Contains(info.Frameworks, "nextjs") || !slices.Contains(info.Frameworks, "react") {
		t.Fatalf("expected next/react framework detection, got %#v", info.Frameworks)
	}
	if !slices.Contains(info.BuildTools, "webpack") {
		t.Fatalf("expected webpack build detection, got %#v", info.BuildTools)
	}
	if !slices.Contains(info.Artifacts, "chunk") {
		t.Fatalf("expected chunk artifact detection, got %#v", info.Artifacts)
	}
	if info.AnalysisStrategy != "build-artifact-first" {
		t.Fatalf("expected build-artifact-first strategy, got %q", info.AnalysisStrategy)
	}
}

func TestDetectFrontendUnknownDoesNotBlockExtraction(t *testing.T) {
	info := DetectFrontendFromHTML(`<html><body>plain</body></html>`, nil)
	if !slices.Contains(info.Frameworks, "unknown") {
		t.Fatalf("expected unknown framework, got %#v", info.Frameworks)
	}

	items := ExtractAll(`fetch("/api/plain")`, "https://example.com", nil, nil)
	if len(items) != 1 || items[0].RawValue != "/api/plain" {
		t.Fatalf("expected fallback extraction to continue, got %#v", items)
	}
}

func TestParseSourceMapRestoresSourcesContent(t *testing.T) {
	sourceMap := `{
		"version":3,
		"file":"app.js",
		"sources":["webpack://src/api/client.ts","webpack://src/routes/admin.ts"],
		"sourcesContent":["export const endpoint = '/api/from-source-map';","const route = '/admin/panel';"]
	}`

	restored, related := ParseSourceMap(sourceMap, "https://example.com/app.js.map")
	if len(restored) != 2 {
		t.Fatalf("expected restored sources, got %#v", restored)
	}
	if !slices.Contains(related, "webpack://src/api/client.ts") {
		t.Fatalf("expected related sources to include source path, got %#v", related)
	}
	if restored[0].SourceType != "restored-source" || !slices.Contains(restored[0].Tags, "sourcemap-sources-content") {
		t.Fatalf("expected restored-source metadata, got %#v", restored[0])
	}
}

func TestExtractAllPrefersSourceMapSourcesContent(t *testing.T) {
	files := []model.SourceFile{
		{
			URL:        "https://example.com/app.js.map",
			SourceType: "sourcemap",
			Frontend:   DetectFrontendFromSource("https://example.com/app.js.map", ""),
			RestoredSources: []model.RestoredSource{
				{
					Name:       "webpack://src/api.ts",
					ParentURL:  "https://example.com/app.js.map",
					SourceType: "restored-source",
					Content:    `fetch("/api/restored")`,
					Tags:       []string{"restored-source", "sourcemap-sources-content"},
				},
			},
		},
	}
	resources := []model.ResourceRecord{{ResourceID: "res-1", URL: "https://example.com/app.js.map", Type: "sourcemap"}}

	items := ExtractAll("", "https://example.com", files, resources)
	if len(items) == 0 {
		t.Fatal("expected restored source candidates")
	}
	found := false
	for _, item := range items {
		if item.RawValue == "/api/restored" {
			found = true
			if item.SourceType != "restored-source" || !slices.Contains(item.HintTags, "restored-source") {
				t.Fatalf("expected restored source metadata, got %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("missing restored source candidate in %#v", items)
	}
}

func TestExtractSourceMapPathCandidatesWithoutSourcesContent(t *testing.T) {
	sourceMap := `{"version":3,"sources":["webpack://src/api/users.ts","webpack://src/pages/home.ts"]}`
	items := ExtractSourceMapPathCandidates(sourceMap, "https://example.com/app.js.map", "res-1", "sourcemap")

	if len(items) != 1 || items[0].RawValue != "/src/api/users.ts" {
		t.Fatalf("expected API-like source path candidate, got %#v", items)
	}
	if items[0].DiscoverRule != "sourcemap-source-path" {
		t.Fatalf("expected sourcemap source rule, got %q", items[0].DiscoverRule)
	}
}

func TestBuildArtifactCandidatesAndTags(t *testing.T) {
	info := DetectFrontendFromSource("https://example.com/assets/index-abcd1234.js", `__webpack_require__.u=function(id){return "chunk-"+id+".js"}; const route="/api/from-route";`)
	items := ExtractKnownFrontendCandidates(`const route="/api/from-route";`, "https://example.com/assets/index-abcd1234.js", "res-1", "javascript", info)

	if len(items) == 0 {
		t.Fatal("expected build-aware candidates")
	}
	if items[0].RawValue != "/api/from-route" {
		t.Fatalf("expected route candidate, got %#v", items)
	}
	if !slices.Contains(items[0].HintTags, "build:vite") && !slices.Contains(items[0].HintTags, "build:webpack") {
		t.Fatalf("expected build tags on candidate, got %#v", items[0].HintTags)
	}
}

func TestFormatJavaScriptForAnalysis(t *testing.T) {
	minified := strings.Repeat(`const a="/api/x";function f(){return a};`, 12)
	formatted := FormatJavaScriptForAnalysis(minified)
	if !strings.Contains(formatted, ";\n") {
		t.Fatalf("expected basic formatting, got %q", formatted[:80])
	}
}
