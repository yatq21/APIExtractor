package core

import (
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"apiextractor/internal/model"
)

var (
	nextDataPattern        = regexp.MustCompile(`(?i)\bid\s*=\s*["']__next["']`)
	nuxtDataPattern        = regexp.MustCompile(`(?i)\b(?:window\.)?__NUXT__\b|/[_-]nuxt/`)
	vueMarkerPattern       = regexp.MustCompile(`(?i)\bdata-v-[a-f0-9]{4,}\b|\bVue(?:\.|js\b)|\bcreateApp\s*\(`)
	reactMarkerPattern     = regexp.MustCompile(`(?i)\bid\s*=\s*["']root["']|\bReact(?:DOM)?\.|\bcreateRoot\s*\(`)
	angularMarkerPattern   = regexp.MustCompile(`(?i)\bng-version\b|\bng-app\b|\bplatformBrowserDynamic\b|polyfills(?:\.[a-z0-9-]+)?\.js`)
	viteMarkerPattern      = regexp.MustCompile(`(?i)@vite/client|/__vite_ping|/assets/index-[a-z0-9_-]+\.js|vite(?:\.|-)manifest`)
	webpackMarkerPattern   = regexp.MustCompile(`(?i)__webpack_require__|webpackJsonp|webpackChunk|webpack-runtime|runtime(?:\.[a-z0-9-]+)?\.js`)
	chunkNamePattern       = regexp.MustCompile(`(?i)(?:chunk|bundle|vendor|runtime|manifest|polyfills|framework|main|app|pages|static)`)
	routeListPattern       = regexp.MustCompile(`(?i)["']((?:/[_a-z0-9.-]+)+)["']`)
	webpackChunkRefPattern = regexp.MustCompile(`(?i)["']([^"']*(?:chunk|bundle|vendor|runtime|main|app|pages|static)[^"']*\.(?:m?js|json|map)(?:\?[^"']*)?)["']`)
)

type sourceMapPayload struct {
	Version        int      `json:"version"`
	File           string   `json:"file"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
}

// DetectFrontendFromHTML identifies framework and build fingerprints from the entry HTML and its resource URLs.
func DetectFrontendFromHTML(html string, sourceURLs []string) *model.FrontendInfo {
	info := newFrontendInfo()
	matchFrontendText(html, info)
	for _, item := range sourceURLs {
		matchFrontendURL(item, info)
	}
	finalizeFrontendInfo(info)
	return info
}

// DetectFrontendFromSource identifies framework and build fingerprints from one source file.
func DetectFrontendFromSource(rawURL string, content string) *model.FrontendInfo {
	info := newFrontendInfo()
	matchFrontendURL(rawURL, info)
	matchFrontendText(content, info)
	finalizeFrontendInfo(info)
	return info
}

// FrontendTags converts recognition data into stable resource/candidate tags.
func FrontendTags(info *model.FrontendInfo) []string {
	if info == nil {
		return nil
	}
	tags := make([]string, 0, len(info.Frameworks)+len(info.BuildTools)+len(info.Artifacts)+1)
	for _, item := range info.Frameworks {
		tags = append(tags, "framework:"+item)
	}
	for _, item := range info.BuildTools {
		tags = append(tags, "build:"+item)
	}
	for _, item := range info.Artifacts {
		tags = append(tags, "artifact:"+item)
	}
	if info.AnalysisStrategy != "" {
		tags = append(tags, "analysis:"+info.AnalysisStrategy)
	}
	return tags
}

// FormatJavaScriptForAnalysis adds light structure to minified JavaScript before fallback extraction.
func FormatJavaScriptForAnalysis(text string) string {
	if !looksMinifiedJavaScript(text) {
		return text
	}
	replacer := strings.NewReplacer(
		";", ";\n",
		"{", "{\n",
		"}", "}\n",
		",", ",\n",
	)
	return replacer.Replace(text)
}

// ParseSourceMap restores embedded source content and records source paths/modules.
func ParseSourceMap(content string, sourceMapURL string) ([]model.RestoredSource, []string) {
	var payload sourceMapPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, nil
	}

	related := make([]string, 0, len(payload.Sources)+1)
	if payload.File != "" {
		related = appendUniqueStrings(related, payload.File)
	}
	related = appendUniqueStrings(related, payload.Sources...)

	restored := make([]model.RestoredSource, 0, len(payload.SourcesContent))
	for idx, sourceContent := range payload.SourcesContent {
		if strings.TrimSpace(sourceContent) == "" {
			continue
		}
		name := ""
		if idx < len(payload.Sources) {
			name = payload.Sources[idx]
		}
		restored = append(restored, model.RestoredSource{
			Name:       name,
			ParentURL:  sourceMapURL,
			SourceType: "restored-source",
			Content:    FormatJavaScriptForAnalysis(sourceContent),
			Tags:       []string{"restored-source", "sourcemap-sources-content"},
		})
	}

	return restored, related
}

// ExtractKnownFrontendCandidates runs framework/build-aware recovery before regex fallback.
func ExtractKnownFrontendCandidates(text string, sourceURL string, sourceResourceID string, sourceType string, info *model.FrontendInfo) []model.ExtractedCandidate {
	if info == nil {
		info = DetectFrontendFromSource(sourceURL, text)
	}

	out := make([]model.ExtractedCandidate, 0)
	merge := func(items []model.ExtractedCandidate) {
		for _, item := range items {
			item.HintTags = mergeStringTags(item.HintTags, FrontendTags(info))
			out = append(out, item)
		}
	}

	if containsString(info.Artifacts, "sourcemap") {
		for _, item := range ExtractSourceMapPathCandidates(text, sourceURL, sourceResourceID, sourceType) {
			item.HintTags = mergeStringTags(item.HintTags, []string{"sourcemap-source-path"})
			out = append(out, item)
		}
	}
	if containsAny(info.BuildTools, "webpack", "vite", "nextjs", "nuxt") || containsString(info.Artifacts, "manifest") {
		merge(ExtractBuildArtifactCandidates(text, sourceURL, sourceResourceID, sourceType, info))
	}
	return preferRicherExtractedCandidates(out)
}

// ExtractSourceMapPathCandidates keeps API-like source paths when sourcesContent is unavailable.
func ExtractSourceMapPathCandidates(content string, sourceURL string, sourceResourceID string, sourceType string) []model.ExtractedCandidate {
	var payload sourceMapPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	out := make([]model.ExtractedCandidate, 0)
	seen := make(map[string]struct{})
	for _, item := range payload.Sources {
		candidate := sourceMapSourceCandidate(item)
		if !looksLikeAPI(candidate) {
			continue
		}
		key := candidate + "|sourcemap-source-path"
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model.ExtractedCandidate{
			RawValue:         candidate,
			HintTags:         []string{"sourcemap-source-path"},
			SourceResourceID: sourceResourceID,
			SourceURL:        sourceURL,
			SourceType:       sourceType,
			DiscoverRule:     "sourcemap-source-path",
		})
	}
	return out
}

func sourceMapSourceCandidate(raw string) string {
	candidate := cleanCandidate(raw)
	lower := strings.ToLower(candidate)
	if strings.HasPrefix(lower, "webpack://") || strings.HasPrefix(lower, "ng://") {
		candidate = candidate[strings.Index(candidate, "://")+3:]
		candidate = strings.TrimLeft(candidate, "/")
	}
	if candidate != "" && !strings.HasPrefix(candidate, "/") && strings.Contains(strings.ToLower(candidate), "api") {
		candidate = "/" + candidate
	}
	return candidate
}

// ExtractBuildArtifactCandidates recovers route and manifest-style clues from known frontend artifacts.
func ExtractBuildArtifactCandidates(text string, sourceURL string, sourceResourceID string, sourceType string, info *model.FrontendInfo) []model.ExtractedCandidate {
	seen := make(map[string]struct{})
	out := make([]model.ExtractedCandidate, 0)
	add := func(raw string, rule string) {
		candidate := cleanCandidate(raw)
		if !looksLikeAPI(candidate) {
			return
		}
		key := candidate + "|" + rule
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, model.ExtractedCandidate{
			RawValue:         candidate,
			HintTags:         FrontendTags(info),
			SourceResourceID: sourceResourceID,
			SourceURL:        sourceURL,
			SourceType:       sourceType,
			DiscoverRule:     rule,
		})
	}

	if sourceType == "json" || containsString(info.Artifacts, "manifest") {
		extractJSONStrings(text, func(value string) {
			add(value, "frontend-manifest-value")
		})
	}
	for _, match := range routeListPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			add(match[1], "frontend-route-list")
		}
	}
	for _, match := range webpackChunkRefPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			add(match[1], "frontend-chunk-reference")
		}
	}
	return out
}

func newFrontendInfo() *model.FrontendInfo {
	return &model.FrontendInfo{
		Frameworks: make([]string, 0),
		BuildTools: make([]string, 0),
		Artifacts:  make([]string, 0),
	}
}

func matchFrontendText(text string, info *model.FrontendInfo) {
	if text == "" {
		return
	}
	lower := strings.ToLower(text)
	switch {
	case nextDataPattern.MatchString(text) || strings.Contains(lower, "/_next/static/"):
		addFrontendValue(&info.Frameworks, "nextjs")
		addFrontendValue(&info.Frameworks, "react")
		addFrontendValue(&info.BuildTools, "webpack")
	case nuxtDataPattern.MatchString(text):
		addFrontendValue(&info.Frameworks, "nuxt")
		addFrontendValue(&info.Frameworks, "vue")
	case angularMarkerPattern.MatchString(text):
		addFrontendValue(&info.Frameworks, "angular")
	case vueMarkerPattern.MatchString(text):
		addFrontendValue(&info.Frameworks, "vue")
	case reactMarkerPattern.MatchString(text):
		addFrontendValue(&info.Frameworks, "react")
	}
	if viteMarkerPattern.MatchString(text) {
		addFrontendValue(&info.BuildTools, "vite")
	}
	if webpackMarkerPattern.MatchString(text) {
		addFrontendValue(&info.BuildTools, "webpack")
	}
	if sourceMapPattern.MatchString(text) {
		addFrontendValue(&info.Artifacts, "sourcemap")
	}
	if chunkNamePattern.MatchString(text) {
		addFrontendValue(&info.Artifacts, "chunk")
	}
}

func matchFrontendURL(rawURL string, info *model.FrontendInfo) {
	parsed, err := url.Parse(rawURL)
	pathValue := rawURL
	if err == nil {
		pathValue = parsed.Path
	}
	lower := strings.ToLower(pathValue)
	switch {
	case strings.Contains(lower, "/_next/"):
		addFrontendValue(&info.Frameworks, "nextjs")
		addFrontendValue(&info.Frameworks, "react")
		addFrontendValue(&info.BuildTools, "webpack")
	case strings.Contains(lower, "/_nuxt/"):
		addFrontendValue(&info.Frameworks, "nuxt")
		addFrontendValue(&info.Frameworks, "vue")
	case strings.Contains(lower, "polyfills") || strings.Contains(lower, "angular"):
		addFrontendValue(&info.Frameworks, "angular")
	}
	if strings.Contains(lower, "/assets/") && regexp.MustCompile(`(?i)[.-][a-z0-9_-]{6,}\.(?:m?js|css)$`).MatchString(lower) {
		addFrontendValue(&info.BuildTools, "vite")
	}
	if strings.Contains(lower, "webpack") || strings.Contains(lower, "runtime") || strings.Contains(lower, "webpack-runtime") {
		addFrontendValue(&info.BuildTools, "webpack")
	}
	ext := strings.ToLower(path.Ext(pathValue))
	switch ext {
	case ".map":
		addFrontendValue(&info.Artifacts, "sourcemap")
	case ".json":
		if strings.Contains(lower, "manifest") || strings.Contains(lower, "_buildmanifest") || strings.Contains(lower, "build-manifest") {
			addFrontendValue(&info.Artifacts, "manifest")
		}
	case ".js", ".mjs":
		if chunkNamePattern.MatchString(lower) {
			addFrontendValue(&info.Artifacts, "chunk")
		}
	}
}

func finalizeFrontendInfo(info *model.FrontendInfo) {
	info.Frameworks = sortedUniqueOrUnknown(info.Frameworks)
	info.BuildTools = sortedUnique(info.BuildTools)
	info.Artifacts = sortedUnique(info.Artifacts)
	switch {
	case containsString(info.Artifacts, "sourcemap"):
		info.AnalysisStrategy = "sourcemap-first"
	case containsAny(info.BuildTools, "webpack", "vite", "nextjs", "nuxt"):
		info.AnalysisStrategy = "build-artifact-first"
	default:
		info.AnalysisStrategy = "heuristic-fallback"
	}
}

func addFrontendValue(items *[]string, value string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return
	}
	for _, item := range *items {
		if item == value {
			return
		}
	}
	*items = append(*items, value)
}

func sortedUniqueOrUnknown(items []string) []string {
	items = sortedUnique(items)
	if len(items) == 0 {
		return []string{"unknown"}
	}
	return items
}

func sortedUnique(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsAny(items []string, wants ...string) bool {
	for _, want := range wants {
		if containsString(items, want) {
			return true
		}
	}
	return false
}

func looksMinifiedJavaScript(text string) bool {
	if len(text) < 256 {
		return false
	}
	lines := strings.Count(text, "\n") + 1
	return len(text)/lines > 180
}

func extractJSONStrings(text string, visit func(string)) {
	var payload any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return
	}
	var walk func(any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		case string:
			visit(value)
		}
	}
	walk(payload)
}
