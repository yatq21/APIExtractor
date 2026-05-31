package core

import (
	"net/http"
	"slices"
	"testing"

	"apiextractor/internal/model"
)

func TestNormalizeCandidatesUsesSourceURLAndTargetOrigin(t *testing.T) {
	items := NormalizeCandidates([]model.ExtractedCandidate{
		{RawValue: "../api/user", SourceURL: "https://example.com/assets/chunk/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "../about", SourceURL: "https://example.com/assets/chunk/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "html-a-href"},
		{RawValue: "api/fallback", SourceResourceID: "res-target", SourceType: "html", DiscoverRule: "fetch-call"},
		{RawValue: "https://example.com/api/from-cdn", SourceURL: "https://cdn.example.net/assets/chunk/app.js", SourceResourceID: "res-cdn", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "/api/cdn-relative", SourceURL: "https://cdn.example.net/assets/chunk/app.js", SourceResourceID: "res-cdn", SourceType: "javascript", DiscoverRule: "fetch-call"},
	}, "https://example.com/app/index.html", true)

	byRaw := make(map[string]model.APICandidate, len(items))
	for _, item := range items {
		byRaw[item.RawValue] = item
	}

	if byRaw["../api/user"].NormalizedURL != "https://example.com/api/user" {
		t.Fatalf("expected root-normalized API path, got %#v", byRaw["../api/user"])
	}
	if byRaw["../api/user"].SourceURL != "https://example.com/assets/chunk/app.js" || byRaw["../api/user"].DiscoverRule != "fetch-call" {
		t.Fatal("expected source metadata to be preserved")
	}
	if byRaw["../about"].NormalizedURL == "https://example.com/about" || byRaw["../about"].Confidence == "high" {
		t.Fatalf("expected ../about to remain conservative, got %#v", byRaw["../about"])
	}
	if byRaw["api/fallback"].NormalizedURL != "https://example.com/app/api/fallback" {
		t.Fatalf("expected empty SourceURL to fall back to target URL, got %#v", byRaw["api/fallback"])
	}
	if byRaw["https://example.com/api/from-cdn"].NormalizedURL != "https://example.com/api/from-cdn" {
		t.Fatalf("expected same-origin absolute target URL to pass, got %#v", byRaw["https://example.com/api/from-cdn"])
	}
	if _, exists := byRaw["/api/cdn-relative"]; exists {
		t.Fatal("expected relative CDN candidate to be rejected by target same-origin boundary")
	}
}

func TestExtractAndNormalizeParentRelativeAPIPath(t *testing.T) {
	extracted := ExtractFromText(`fetch("../api/user")`, "https://example.com/assets/chunk/app.js", "res-js", "javascript")
	candidates := NormalizeCandidates(extracted, "https://example.com/index.html", true)
	if len(candidates) != 1 {
		t.Fatalf("expected parent-relative API candidate, got %#v", candidates)
	}
	if candidates[0].NormalizedURL != "https://example.com/api/user" {
		t.Fatalf("expected root-normalized parent-relative API, got %#v", candidates[0])
	}
}

func TestExtractAllParsesSourceMapSourcesContent(t *testing.T) {
	sourceMapURL := "https://example.com/assets/app.js.map"
	items := ExtractAll("", "https://example.com/index.html", []model.SourceFile{
		{
			URL:        sourceMapURL,
			SourceType: "sourcemap",
			Content:    `{"version":3,"sources":["webpack://src/app.js"],"sourcesContent":["fetch(\"/api/user/profile\")"]}`,
		},
	}, []model.ResourceRecord{
		{ResourceID: "res-map", URL: sourceMapURL, Type: "sourcemap"},
	})

	if len(items) != 1 {
		t.Fatalf("expected 1 source-map candidate, got %#v", items)
	}
	item := items[0]
	if item.RawValue != "/api/user/profile" {
		t.Fatalf("expected restored source API, got %#v", item)
	}
	if item.SourceType != "restored-source" {
		t.Fatalf("expected restored-source type, got %s", item.SourceType)
	}
	if item.DiscoverRule != "sourcemap-sources-content" {
		t.Fatalf("expected sourcemap-sources-content rule, got %s", item.DiscoverRule)
	}
	if item.SourceURL != sourceMapURL || item.SourceResourceID != "res-map" {
		t.Fatal("expected source map URL and resource id to be preserved")
	}
	if !containsTag(item.HintTags, "source-map") || !containsTag(item.HintTags, "restored-source") {
		t.Fatalf("expected source map hint tags, got %#v", item.HintTags)
	}
}

func TestSourceMapSourcesOnlyDoesNotCreateCandidates(t *testing.T) {
	sourceMapURL := "https://example.com/assets/app.js.map"
	content := `{"version":3,"sources":["../api/user.ts"],"sourceRoot":"webpack://src"}`
	items := ExtractAll("", "https://example.com/index.html", []model.SourceFile{
		{URL: sourceMapURL, SourceType: "sourcemap", Content: content},
	}, []model.ResourceRecord{
		{ResourceID: "res-map", URL: sourceMapURL, Type: "sourcemap"},
	})

	if len(items) != 0 {
		t.Fatalf("expected no candidates from sources-only source map, got %#v", items)
	}
	tags := resourceTags("sourcemap", sourceMapURL, content)
	if !containsTag(tags, "source-map") || !containsTag(tags, "sourcemap-sources-only") {
		t.Fatalf("expected sources-only resource tags, got %#v", tags)
	}
}

func TestHTMLElementDiscoveryFindsResourcesAndAPICandidates(t *testing.T) {
	html := `
		<script type="module" src="/assets/app.js"></script>
		<link rel="modulepreload" href="/assets/chunk-vendor.mjs">
		<link rel="manifest" href="/manifest.json">
		<link rel="stylesheet" href="/assets/app.css">
		<iframe src="/api/frame"></iframe>
		<form action="/auth/login"></form>
		<a href="/api/users">users</a>
		<a href="/about">about</a>
		<a href="/assets/logo.png">logo</a>
		<meta http-equiv="refresh" content="0; url=/api/redirect">
	`

	sourceURLs := ExtractSourceURLs(html, "https://example.com/index.html", true)
	assertStringSlice(t, sourceURLs, []string{
		"https://example.com/assets/app.js",
		"https://example.com/assets/chunk-vendor.mjs",
		"https://example.com/manifest.json",
	})

	candidates := NormalizeCandidates(ExtractAll(html, "https://example.com/index.html", nil, nil), "https://example.com/index.html", true)
	urls := make([]string, 0, len(candidates))
	for _, item := range candidates {
		urls = append(urls, item.NormalizedURL)
	}

	for _, want := range []string{
		"https://example.com/api/frame",
		"https://example.com/auth/login",
		"https://example.com/api/users",
		"https://example.com/api/redirect",
	} {
		if !slices.Contains(urls, want) {
			t.Fatalf("missing %s in %#v", want, urls)
		}
	}
	for _, unwanted := range []string{
		"https://example.com/assets/app.css",
		"https://example.com/assets/logo.png",
	} {
		if slices.Contains(urls, unwanted) {
			t.Fatalf("did not expect static resource candidate %s in %#v", unwanted, urls)
		}
	}
}

func TestResourceTagsDetectFrameworksAndBuildTools(t *testing.T) {
	resources := buildResources("https://example.com/index.html", `
		<script id="__NEXT_DATA__" type="application/json">{}</script>
		<script type="module" src="/assets/main.js"></script>
		<div ng-version="17" data-v-abcd data-reactroot></div>
	`, []model.SourceFile{
		{URL: "https://example.com/_nuxt/app.js", SourceType: "javascript", Content: `window.__NUXT__={}`},
		{URL: "https://example.com/assets/chunk.js", SourceType: "javascript", Content: `__webpack_require__; webpackJsonp=[];`},
		{URL: "https://example.com/assets/polyfills.123.js", SourceType: "javascript", Content: ``},
	})

	allTags := make(map[string]struct{})
	for _, resource := range resources {
		for _, tag := range resource.Tags {
			allTags[tag] = struct{}{}
		}
	}
	for _, want := range []string{
		"build:next",
		"build:nuxt",
		"build:vite",
		"build:webpack",
		"build:angular",
		"framework:vue",
		"framework:react",
	} {
		if _, exists := allTags[want]; !exists {
			t.Fatalf("missing framework/build tag %s in %#v", want, allTags)
		}
	}
}

func TestUnknownFrameworkStillExtractsAPI(t *testing.T) {
	items := ExtractAll("", "https://example.com/index.html", []model.SourceFile{
		{URL: "https://example.com/static/app.js", SourceType: "javascript", Content: `fetch("/api/unknown")`},
	}, []model.ResourceRecord{
		{ResourceID: "res-js", URL: "https://example.com/static/app.js", Type: "javascript"},
	})
	candidates := NormalizeCandidates(items, "https://example.com/index.html", true)
	if len(candidates) != 1 || candidates[0].NormalizedURL != "https://example.com/api/unknown" || candidates[0].MethodGuess != http.MethodGet {
		t.Fatalf("expected unknown framework JS to keep API extraction working, got %#v", candidates)
	}
}
