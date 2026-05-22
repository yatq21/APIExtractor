package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

func TestNormalizeCandidates(t *testing.T) {
	items := NormalizeCandidates([]model.ExtractedCandidate{
		{RawValue: "/api/user?id=1", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "/api/user?id=1", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "//example.com/api/health", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "quoted-url"},
		{RawValue: "/assets/app.css", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "quoted-url"},
		{RawValue: "/api/:id/detail", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "xhr-open"},
		{RawValue: "${baseURL}/api/v1/users#section", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "object-property"},
	}, "https://example.com/app/index.html", false)

	if len(items) != 4 {
		t.Fatalf("expected 4 normalized candidates, got %d", len(items))
	}
	if items[0].MethodGuess == "" {
		t.Fatal("expected method guess to be set")
	}
	var foundLowConfidence bool
	for _, item := range items {
		if item.RawValue == "${baseURL}/api/v1/users#section" {
			foundLowConfidence = true
			if item.Confidence != "low" {
				t.Fatalf("expected low confidence for unresolved base token, got %s", item.Confidence)
			}
			if item.NormalizedURL != "https://example.com/api/v1/users" {
				t.Fatalf("unexpected normalized url: %s", item.NormalizedURL)
			}
			if item.SourceResourceID != "res-1" || item.SourceType != "javascript" {
				t.Fatal("expected source metadata to be preserved")
			}
		}
	}
	if !foundLowConfidence {
		t.Fatal("expected unresolved base token candidate to be retained")
	}
}

func TestNormalizeCandidateCategoriesAndTags(t *testing.T) {
	items := NormalizeCandidates([]model.ExtractedCandidate{
		{RawValue: "/api/internal/health", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "/api/v1/auth/login", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "/api/v1/profile", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
		{RawValue: "/openapi.json", SourceURL: "https://example.com/app.js", SourceResourceID: "res-1", SourceType: "javascript", DiscoverRule: "fetch-call"},
	}, "https://example.com/index.html", false)

	if len(items) != 4 {
		t.Fatalf("expected 4 normalized items, got %d", len(items))
	}

	categories := make(map[string]model.APICandidate, len(items))
	for _, item := range items {
		categories[item.Path] = item
	}
	if categories["/api/internal/health"].Category != "internal-api" {
		t.Fatal("expected internal-api category")
	}
	if !containsTag(categories["/api/internal/health"].Tags, "internal-semantic") {
		t.Fatal("expected internal semantic tag")
	}
	if categories["/api/v1/auth/login"].Category != "auth-endpoint" {
		t.Fatal("expected auth-endpoint category")
	}
	if categories["/api/v1/profile"].Category != "authenticated-api" {
		t.Fatal("expected authenticated-api category")
	}
	if categories["/openapi.json"].Category != "api-doc" {
		t.Fatal("expected api-doc category")
	}
}

func TestNormalizeCandidatesRespectsMethodHint(t *testing.T) {
	items := NormalizeCandidates([]model.ExtractedCandidate{
		{RawValue: "/api/v1/orders", MethodHint: http.MethodPost, HintTags: []string{"auth-required-hint"}, SourceURL: "https://example.com/openapi.json", SourceResourceID: "res-1", SourceType: "json", DiscoverRule: "openapi-path"},
	}, "https://example.com/index.html", false)

	if len(items) != 1 {
		t.Fatalf("expected 1 normalized candidate, got %d", len(items))
	}
	if items[0].MethodGuess != http.MethodPost {
		t.Fatalf("expected POST method guess from hint, got %s", items[0].MethodGuess)
	}
	if !containsTag(items[0].Tags, "auth-required-hint") {
		t.Fatal("expected hint tags to flow into normalized candidate tags")
	}
}

func TestExtractFromTextCapturesMethodAwareCalls(t *testing.T) {
	items := ExtractFromText(`
		axios.post("/api/v1/orders", payload)
		xhr.open("DELETE", "/api/v1/users/1")
		$.post("/api/v1/login", data)
	`, "https://example.com/app.js", "res-1", "javascript")

	if len(items) < 3 {
		t.Fatalf("expected method-aware extracted candidates, got %d", len(items))
	}
	methods := make(map[string]string, len(items))
	for _, item := range items {
		methods[item.RawValue] = item.MethodHint
	}
	if methods["/api/v1/orders"] != http.MethodPost {
		t.Fatal("expected POST hint from axios.post")
	}
	if methods["/api/v1/users/1"] != http.MethodDelete {
		t.Fatal("expected DELETE hint from xhr.open")
	}
	if methods["/api/v1/login"] != http.MethodPost {
		t.Fatal("expected POST hint from jquery post")
	}
}

func TestRequestAndAnalyzeResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != "" && got != "session=test" {
			t.Fatalf("unexpected cookie header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"admin@example.com","token":"abc.def.ghi","debug":"stacktrace"}`))
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.Cookies = "session=test"
	candidates := NormalizeCandidates([]model.ExtractedCandidate{{RawValue: server.URL + "/api/profile", SourceURL: server.URL + "/app.js", SourceResourceID: "res-2", SourceType: "javascript", DiscoverRule: "fetch-call"}}, server.URL, false)
	results := AnalyzeResults(RequestAll(candidates, cfg), cfg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", results[0].StatusCode)
	}
	if len(results[0].SensitiveMatches) == 0 {
		t.Fatal("expected sensitive matches")
	}
	if len(results[0].RiskTags) == 0 {
		t.Fatal("expected risk tags")
	}
	if results[0].SourceResourceID != "res-2" {
		t.Fatal("expected source resource id to propagate to request result")
	}
	if results[0].SourceURL == "" || results[0].SourceType == "" {
		t.Fatal("expected source url and source type to propagate to request result")
	}
	if !strings.Contains(results[0].CurlCommand, "curl") || !strings.Contains(results[0].CurlCommand, server.URL+"/api/profile") {
		t.Fatal("expected reproducible curl command")
	}
	var hasFieldNameHit bool
	var hasFieldValueHit bool
	for _, match := range results[0].SensitiveMatches {
		if match.MatchType == "token" && match.MatchScope == "field_name" {
			hasFieldNameHit = true
		}
		if match.MatchType == "email" && match.MatchScope == "field_value" {
			hasFieldValueHit = true
		}
	}
	if !hasFieldNameHit || !hasFieldValueHit {
		t.Fatal("expected both field_name and field_value matches")
	}
}

func TestAnalyzeRiskTags(t *testing.T) {
	cfg := config.Default()
	cfg.LargeJSONThreshold = 32

	t.Run("auth required", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:         "https://example.com/api/private",
			StatusCode:     http.StatusUnauthorized,
			ContentType:    "application/json",
			ResponseSample: `{"message":"authentication required"}`,
		}}, cfg)
		if !containsTag(result[0].RiskTags, "auth_required") {
			t.Fatal("expected auth_required tag")
		}
		if len(result[0].RiskEvidence) == 0 || result[0].RiskEvidence[0].Reason == "" {
			t.Fatal("expected risk evidence with reason")
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:      "https://example.com/api/private",
			StatusCode:  http.StatusForbidden,
			ContentType: "application/json",
		}}, cfg)
		if !containsTag(result[0].RiskTags, "forbidden") {
			t.Fatal("expected forbidden tag")
		}
	})

	t.Run("redirect to login", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:           "https://example.com/api/profile",
			StatusCode:       http.StatusOK,
			RedirectLocation: "https://example.com/login",
			ContentType:      "text/html",
			ResponseSample:   "<html>login</html>",
		}}, cfg)
		if !containsTag(result[0].RiskTags, "redirect_to_login") {
			t.Fatal("expected redirect_to_login tag")
		}
	})

	t.Run("soft 404 hints only", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:         "https://example.com/api/missing",
			StatusCode:     http.StatusOK,
			ContentType:    "text/html",
			ResponseSample: "<html><title>404</title>not found</html>",
		}}, cfg)
		if len(result[0].RiskTags) != 0 {
			t.Fatal("expected no hard risk tags for soft-404 style response")
		}
		if !slices.Contains(result[0].RiskHints, "body_contains_not_found") || !slices.Contains(result[0].RiskHints, "title_contains_404") {
			t.Fatal("expected soft-404 hints to be recorded")
		}
	})

	t.Run("admin api exposed", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:        "https://example.com/admin/users",
			StatusCode:    http.StatusOK,
			ContentType:   "application/json",
			ContentLength: 64,
		}}, cfg)
		if !containsTag(result[0].RiskTags, "admin_api_exposed") {
			t.Fatal("expected admin_api_exposed tag")
		}
	})

	t.Run("large json and low confidence", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:        "https://example.com/api/export",
			StatusCode:    http.StatusOK,
			ContentType:   "application/json",
			ContentLength: 256,
			Confidence:    "low",
		}}, cfg)
		if !containsTag(result[0].RiskTags, "large_json_response") {
			t.Fatal("expected large_json_response tag")
		}
		if !containsTag(result[0].RiskTags, "low_confidence") {
			t.Fatal("expected low_confidence tag")
		}
	})

	t.Run("internal api exposed", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:      "https://example.com/api/internal/metrics",
			Category:    "internal-api",
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}}, cfg)
		if !containsTag(result[0].RiskTags, "internal_api_exposed") {
			t.Fatal("expected internal_api_exposed tag")
		}
	})

	t.Run("graphql endpoint exposed", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:      "https://example.com/graphql",
			Category:    "graphql",
			StatusCode:  http.StatusBadRequest,
			ContentType: "application/json",
		}}, cfg)
		if !containsTag(result[0].RiskTags, "graphql_endpoint_exposed") {
			t.Fatal("expected graphql_endpoint_exposed tag")
		}
	})

	t.Run("authenticated api exposed", func(t *testing.T) {
		result := AnalyzeResults([]model.APIResult{{
			APIURL:      "https://example.com/api/v1/profile",
			Category:    "authenticated-api",
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}}, cfg)
		if !containsTag(result[0].RiskTags, "authenticated_api_exposed") {
			t.Fatal("expected authenticated_api_exposed tag")
		}
	})
}

func TestRequestAllConcurrentPreservesOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "slow") {
			<-timeAfterChan()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.RequestConcurrency = 3

	candidates := []model.APICandidate{
		{NormalizedURL: server.URL + "/slow-1", MethodGuess: http.MethodGet},
		{NormalizedURL: server.URL + "/fast-2", MethodGuess: http.MethodGet},
		{NormalizedURL: server.URL + "/slow-3", MethodGuess: http.MethodGet},
	}
	results := RequestAll(candidates, cfg)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !strings.Contains(results[0].APIURL, "slow-1") || !strings.Contains(results[1].APIURL, "fast-2") || !strings.Contains(results[2].APIURL, "slow-3") {
		t.Fatal("expected concurrent requester to preserve result order")
	}
}

func TestExtractResponseCandidatesAndRecovery(t *testing.T) {
	results := []model.APIResult{
		{
			APIURL:         "https://example.com/api/bootstrap",
			StatusCode:     http.StatusOK,
			ContentType:    "application/json",
			ResponseSample: `{"links":["/api/v1/orders","/graphql"]}`,
		},
	}
	items := ExtractResponseCandidates(results)
	if len(items) < 2 {
		t.Fatalf("expected response candidate recovery, got %d", len(items))
	}
	normalized := NormalizeCandidates(items, "https://example.com/index.html", true)
	if len(normalized) < 2 {
		t.Fatalf("expected normalized recovered candidates, got %d", len(normalized))
	}
}

func timeAfterChan() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		close(ch)
	}()
	return ch
}

func TestBuiltinWordlistsReservedMeta(t *testing.T) {
	items := builtinWordlists()
	if len(items) == 0 {
		t.Fatal("expected reserved builtin wordlist metadata")
	}
	if items[0].SourceType != "builtin" {
		t.Fatal("expected builtin source type")
	}
}

func TestCleanWordlistEntriesAndBuildTargets(t *testing.T) {
	items := CleanWordlistEntries([]string{
		"",
		"# comment",
		"api",
		"/api",
		"/admin  # inline",
		"\\swagger",
	})
	if len(items) != 3 {
		t.Fatalf("expected 3 cleaned items, got %d", len(items))
	}
	targets := BuildScanTargets("https://example.com/path/index.html", items)
	if !slices.Contains(targets, "https://example.com/api") {
		t.Fatal("expected /api target")
	}
	if !slices.Contains(targets, "https://example.com/admin") {
		t.Fatal("expected /admin target")
	}
	if !slices.Contains(targets, "https://example.com/swagger") {
		t.Fatal("expected /swagger target")
	}
}

func TestLoadWordlistsWithUserFile(t *testing.T) {
	file, err := os.CreateTemp("", "wordlist-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	_, _ = file.WriteString("/api\nadmin\n/api\n")
	_ = file.Close()

	cfg := config.Default()
	cfg.WordlistPath = file.Name()
	entries, metas, err := LoadWordlists("https://example.com/index.html", cfg)
	if err != nil {
		t.Fatalf("load wordlists: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected loaded entries")
	}
	if len(metas) < 2 {
		t.Fatal("expected builtin and user wordlist metadata")
	}
	foundUser := false
	for _, item := range metas {
		if item.SourceType == "user_file" {
			foundUser = true
			if item.EntryCount == 0 || item.SHA256 == "" {
				t.Fatal("expected user wordlist manifest data")
			}
		}
	}
	if !foundUser {
		t.Fatal("expected user_file metadata")
	}
}

func TestScanDirectoryResources(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/swagger":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>swagger</html>"))
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"openapi":"3.0.0"}`))
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Disallow: /admin\nSitemap: " + baseURL + "/sitemap.xml"))
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><urlset><url><loc>` + baseURL + `/api/v1/products</loc></url></urlset>`))
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"start_url":"/api/mobile/bootstrap","related_applications":[{"url":"/graphql"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	cfg := config.Default()
	cfg.MaxResources = 10
	targets := []string{server.URL + "/swagger", server.URL + "/openapi.json", server.URL + "/robots.txt", server.URL + "/sitemap.xml", server.URL + "/manifest.json", server.URL + "/missing"}
	resources, hits := ScanDirectoryResources(targets, cfg)
	if len(hits) != 0 {
		t.Fatal("expected no budget hits")
	}
	if len(resources) != 5 {
		t.Fatalf("expected 5 discovered resources, got %d", len(resources))
	}
	types := make([]string, 0, len(resources))
	for _, resource := range resources {
		types = append(types, resource.Type)
		if resource.Path == "" {
			t.Fatal("expected normalized resource path")
		}
		if resource.ContentType == "" {
			t.Fatal("expected resource content type")
		}
		if resource.Type == "robots" || resource.Type == "sitemap" || resource.Type == "manifest" {
			if !containsTag(resource.Tags, "discovery-hint") {
				t.Fatal("expected discovery tag on specialized resources")
			}
		}
	}
	if !slices.Contains(types, "html") || !slices.Contains(types, "json") || !slices.Contains(types, "robots") || !slices.Contains(types, "sitemap") || !slices.Contains(types, "manifest") {
		t.Fatal("expected specialized discovery resource types")
	}
}

func TestExtractCandidatesFromDiscoveryResources(t *testing.T) {
	robots := model.ResourceRecord{ResourceID: "res-r", URL: "https://example.com/robots.txt", Type: "robots"}
	robotsItems := ExtractCandidatesFromResourceBody("Disallow: /admin\nSitemap: https://example.com/sitemap.xml", robots)
	if len(robotsItems) < 2 {
		t.Fatalf("expected robots-derived candidates, got %d", len(robotsItems))
	}

	sitemap := model.ResourceRecord{ResourceID: "res-s", URL: "https://example.com/sitemap.xml", Type: "sitemap"}
	sitemapItems := ExtractCandidatesFromResourceBody(`<?xml version="1.0"?><urlset><url><loc>https://example.com/api/v1/products</loc></url></urlset>`, sitemap)
	if len(sitemapItems) != 1 {
		t.Fatalf("expected sitemap-derived candidate, got %d", len(sitemapItems))
	}

	manifest := model.ResourceRecord{ResourceID: "res-m", URL: "https://example.com/manifest.json", Type: "manifest"}
	manifestItems := ExtractCandidatesFromResourceBody(`{"start_url":"/api/mobile/bootstrap","related_applications":[{"url":"/graphql"}]}`, manifest)
	if len(manifestItems) < 2 {
		t.Fatalf("expected manifest-derived candidates, got %d", len(manifestItems))
	}

	openapi := model.ResourceRecord{ResourceID: "res-o", URL: "https://example.com/openapi.json", Type: "json"}
	openapiItems := ExtractCandidatesFromResourceBody(`{"openapi":"3.0.0","servers":[{"url":"https://example.com/base"}],"security":[{"bearerAuth":[]}],"paths":{"/api/v1/orders":{"post":{"tags":["internal"]}},"/api/v1/products":{"get":{"x-internal":true}}}}`, openapi)
	if len(openapiItems) != 2 {
		t.Fatalf("expected openapi-derived candidates, got %d", len(openapiItems))
	}
	methods := make(map[string]string, len(openapiItems))
	tags := make(map[string][]string, len(openapiItems))
	for _, item := range openapiItems {
		methods[item.RawValue] = item.MethodHint
		tags[item.RawValue] = item.HintTags
	}
	if methods["/base/api/v1/orders"] != http.MethodPost {
		t.Fatal("expected POST hint for openapi orders path")
	}
	if methods["/base/api/v1/products"] != http.MethodGet {
		t.Fatal("expected GET hint for openapi products path")
	}
	if !containsTag(tags["/base/api/v1/orders"], "auth-required-hint") || !containsTag(tags["/base/api/v1/orders"], "internal-doc-hint") {
		t.Fatal("expected auth/internal hint tags for openapi orders path")
	}
	if !containsTag(tags["/base/api/v1/products"], "internal-doc-hint") {
		t.Fatal("expected internal hint tag for openapi products path")
	}
}

func TestRequestPreviewTruncationAndRedirectCapture(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login-redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(strings.Repeat("x", 128)))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := config.Default()
	cfg.MaxResponsePreview = 16

	candidate := NormalizeCandidates([]model.ExtractedCandidate{{RawValue: server.URL + "/api/login-redirect", SourceURL: server.URL + "/app.js", SourceResourceID: "res-3", SourceType: "javascript", DiscoverRule: "fetch-call"}}, server.URL, false)[0]
	results := AnalyzeResults(RequestAll([]model.APICandidate{candidate}, cfg), cfg)

	if results[0].RedirectLocation == "" || !strings.Contains(results[0].RedirectLocation, "/login") {
		t.Fatalf("expected redirect location to be captured, got %q", results[0].RedirectLocation)
	}
	if len(results[0].ResponseSample) != 16 {
		t.Fatalf("expected truncated response sample length 16, got %d", len(results[0].ResponseSample))
	}
	if !containsTag(results[0].RiskTags, "redirect_to_login") {
		t.Fatal("expected redirect_to_login risk tag after following redirect")
	}
}

func TestExtractAllPreservesSourceContext(t *testing.T) {
	resources := []model.ResourceRecord{
		{ResourceID: "res-1", URL: "https://example.com/static/app.js", Type: "javascript"},
	}
	files := []model.SourceFile{
		{
			URL:        "https://example.com/static/app.js",
			SourceType: "javascript",
			Content:    `fetch("/api/v1/users"); axios("/graphql")`,
		},
	}

	items := ExtractAll(`<script src="/static/app.js"></script>`, "https://example.com/index.html", files, resources)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 extracted candidates, got %d", len(items))
	}
	foundJS := false
	for _, item := range items {
		if item.SourceResourceID == "res-1" && item.SourceType == "javascript" {
			foundJS = true
			break
		}
	}
	if !foundJS {
		t.Fatal("expected extracted candidate to preserve source resource context")
	}
}

func TestBuildSummaryTracksErrorTypesAndRiskStats(t *testing.T) {
	result := model.ScanResult{
		Resources: []model.ResourceRecord{
			{ResourceID: "res-1", Type: "html", ShouldAnalyze: true},
			{ResourceID: "res-2", Type: "robots", ErrorType: "timeout"},
		},
		Candidates: []model.APICandidate{{CandidateID: "cand-1"}},
		APIResults: []model.APIResult{
			{
				APIURL:      "https://example.com/api/a",
				ContentType: "application/json",
				RiskTags:    []string{"auth_required", "low_confidence"},
			},
			{
				APIURL:      "https://example.com/api/b",
				ErrorReason: "dial tcp timeout",
				ErrorType:   "timeout",
			},
		},
	}

	summary := buildSummary(result)
	if summary.ResourceCount != 2 {
		t.Fatalf("expected resource count 2, got %d", summary.ResourceCount)
	}
	if summary.AnalyzedResourceCount != 1 {
		t.Fatalf("expected analyzed resource count 1, got %d", summary.AnalyzedResourceCount)
	}
	if summary.ResourceTypeStats["html"] != 1 || summary.ResourceTypeStats["robots"] != 1 {
		t.Fatal("expected resource type stats to be counted")
	}
	if summary.ErrorTypeStats["timeout"] != 2 {
		t.Fatal("expected timeout error stats from resources and api results to be counted")
	}
	if summary.RiskTagStats["auth_required"] != 1 || summary.RiskTagStats["low_confidence"] != 1 {
		t.Fatal("expected risk tag stats to be counted")
	}
}

func TestRunIncludesMetaAndEntrypointResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><script src="/app.js"></script></html>`))
		case "/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`fetch("/api/ping")`))
		case "/api/ping":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.DisableBuiltinWordlist = true
	cfg.MaxResources = 0

	result := Run(server.URL, cfg)
	if result.Meta.ScanID == "" {
		t.Fatal("expected scan id in meta")
	}
	if result.Meta.ScanTime == "" {
		t.Fatal("expected scan time in meta")
	}
	if result.Target.URL != server.URL || result.Target.Origin == "" {
		t.Fatal("expected target metadata")
	}
	if result.Target.ConfigSummary.TimeoutSeconds == 0 {
		t.Fatal("expected config summary in target metadata")
	}
	if len(result.Resources) == 0 {
		t.Fatal("expected resources")
	}
	if result.Resources[0].URL != server.URL {
		t.Fatal("expected first resource to be entrypoint")
	}
	if result.Resources[0].Type != "html" || result.Resources[0].Category != "page" {
		t.Fatal("expected entrypoint resource metadata")
	}
	if result.Resources[0].BodyPreview == "" {
		t.Fatal("expected entrypoint body preview")
	}
}

func TestFilterUnverifiedCandidatesRespectsMethod(t *testing.T) {
	candidates := []model.APICandidate{
		{MethodGuess: http.MethodGet, NormalizedURL: "https://example.com/graphql"},
		{MethodGuess: http.MethodPost, NormalizedURL: "https://example.com/graphql"},
	}
	existing := []model.APIResult{
		{Method: http.MethodGet, APIURL: "https://example.com/graphql"},
	}

	filtered := filterUnverifiedCandidates(candidates, existing)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 unverified candidate, got %d", len(filtered))
	}
	if filtered[0].MethodGuess != http.MethodPost {
		t.Fatal("expected POST candidate to remain unverified")
	}
}

func TestRequestAPIUsesCandidateMethod(t *testing.T) {
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := config.Default()
	result := RequestAPI(model.APICandidate{
		NormalizedURL: server.URL + "/api/v1/orders",
		MethodGuess:   http.MethodPost,
		Category:      "authenticated-api",
	}, cfg)

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST request, got %s", gotMethod)
	}
	if result.Method != http.MethodPost {
		t.Fatalf("expected POST result method, got %s", result.Method)
	}
}
