package core

import (
	"encoding/json"
	"encoding/xml"
	"net/url"
	"path"
	"regexp"
	"strings"

	"apiextractor/internal/model"
)

var (
	quotedURLPattern          = regexp.MustCompile("(?i)[\"'`]((?:https?:)?//[^\"'`\\s<>]+|/[A-Za-z0-9._~!$&'()*+,;=:@%/?#\\[\\]-]+)[\"'`]")
	apiKeywordPattern         = regexp.MustCompile("(?i)[\"'`]((?:\\.?\\.?/)?[^\"'`\\s<>]*(?:api|graphql|rest|v[0-9]+)[^\"'`\\s<>]*)[\"'`]")
	fetchPattern              = regexp.MustCompile("(?is)\\bfetch\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	requestConstructorPattern = regexp.MustCompile("(?is)\\bnew\\s+Request\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	webSocketPattern          = regexp.MustCompile("(?is)\\bnew\\s+WebSocket\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	xhrOpenPattern            = regexp.MustCompile("(?is)\\.open\\s*\\(\\s*[\"'`]([A-Z]+)[\"'`]\\s*,\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosPattern              = regexp.MustCompile("(?is)\\baxios(?:\\.[a-z]+)?\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosMethodPattern        = regexp.MustCompile("(?is)\\baxios\\.(get|post|put|delete|patch|head|options)\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosObjectURLPattern     = regexp.MustCompile("(?is)\\baxios\\s*\\(\\s*\\{[^{}]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosObjectMethodPattern  = regexp.MustCompile("(?is)\\baxios\\s*\\(\\s*\\{[^{}]*?\\bmethod\\s*:\\s*[\"'`]([a-z]+)[\"'`][^{}]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	jqueryAjaxURLPattern      = regexp.MustCompile("(?is)\\$\\.(?:ajax|get|post|getJSON)\\s*\\([^)]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	jqueryAjaxMethodPattern   = regexp.MustCompile("(?is)\\$\\.ajax\\s*\\([^)]*?\\bmethod\\s*:\\s*[\"'`]([a-z]+)[\"'`][^)]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	jqueryShortcutPattern     = regexp.MustCompile("(?is)\\$\\.(get|post|getJSON)\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	newURLPattern             = regexp.MustCompile("(?is)\\bnew\\s+URL\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]\\s*,")
	requestObjectURLPattern   = regexp.MustCompile("(?is)\\b(?:url|path|endpoint|uri|baseURL|baseUrl)\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	requestObjectExprPattern  = regexp.MustCompile("(?is)\\b(?:url|path|endpoint|uri|baseURL|baseUrl)\\s*:\\s*([\"'`][^,\\n}]*|/(?:[^,\\n}]*)|https?://[^,\\n}]+|wss?://[^,\\n}]+)")
	graphQLOperationPattern   = regexp.MustCompile("(?is)\\b(?:query|mutation)\\s+[A-Za-z0-9_]*\\s*(?:\\([^)]*\\))?\\s*\\{")
	businessPathPattern       = regexp.MustCompile(`(?i)^/(?:v[0-9]+|admin|auth|user|users|account|accounts|order|orders|pay|payment|member|members|tenant|tenants|system|manage|backend|console)(?:/|$)`)
)

type openAPIServer struct {
	URL string `json:"url"`
}

// ExtractFromText extracts API-like candidates from one text body with source context.
func ExtractFromText(text string, sourceURL string, sourceResourceID string, sourceType string) []model.ExtractedCandidate {
	seen := make(map[string]struct{})
	results := make([]model.ExtractedCandidate, 0)

	mergeMatches := func(matches [][]string, discoverRule string, valueIndex int) {
		for _, match := range matches {
			if len(match) <= valueIndex {
				continue
			}
			addCandidate(match[valueIndex], discoverRule, sourceURL, sourceResourceID, sourceType, seen, &results)
		}
	}
	mergeMethodMatches := func(matches [][]string, discoverRule string, methodIndex int, valueIndex int) {
		for _, match := range matches {
			if len(match) <= valueIndex || len(match) <= methodIndex {
				continue
			}
			addCandidateWithHints(
				match[valueIndex],
				match[methodIndex],
				[]string{"method-from-code"},
				discoverRule,
				sourceURL,
				sourceResourceID,
				sourceType,
				seen,
				&results,
			)
		}
	}

	mergeMatches(fetchPattern.FindAllStringSubmatch(text, -1), "fetch-call", 1)
	mergeMatches(requestConstructorPattern.FindAllStringSubmatch(text, -1), "request-constructor", 1)
	mergeMatches(webSocketPattern.FindAllStringSubmatch(text, -1), "websocket-constructor", 1)
	mergeMethodMatches(xhrOpenPattern.FindAllStringSubmatch(text, -1), "xhr-open", 1, 2)
	mergeMethodMatches(axiosMethodPattern.FindAllStringSubmatch(text, -1), "axios-method", 1, 2)
	mergeMatches(axiosPattern.FindAllStringSubmatch(text, -1), "axios-call", 1)
	mergeMethodMatches(axiosObjectMethodPattern.FindAllStringSubmatch(text, -1), "axios-object", 1, 2)
	mergeMatches(axiosObjectURLPattern.FindAllStringSubmatch(text, -1), "axios-object", 1)
	mergeMethodMatches(jqueryAjaxMethodPattern.FindAllStringSubmatch(text, -1), "jquery-ajax", 1, 2)
	mergeMatches(jqueryAjaxURLPattern.FindAllStringSubmatch(text, -1), "jquery-ajax", 1)
	mergeMethodMatches(jqueryShortcutPattern.FindAllStringSubmatch(text, -1), "jquery-shortcut", 1, 2)
	mergeMatches(requestObjectURLPattern.FindAllStringSubmatch(text, -1), "object-property", 1)
	mergeMatches(requestObjectExprPattern.FindAllStringSubmatch(text, -1), "object-expression", 1)
	mergeMatches(newURLPattern.FindAllStringSubmatch(text, -1), "new-url", 1)
	mergeMatches(quotedURLPattern.FindAllStringSubmatch(text, -1), "quoted-url", 1)
	mergeMatches(apiKeywordPattern.FindAllStringSubmatch(text, -1), "api-keyword", 1)

	if graphQLOperationPattern.MatchString(text) {
		addCandidate("/graphql", "graphql-operation", sourceURL, sourceResourceID, sourceType, seen, &results)
	}

	return preferRicherExtractedCandidates(results)
}

// ExtractAll aggregates all extracted candidates from HTML and fetched source files.
func ExtractAll(html string, targetURL string, sourceFiles []model.SourceFile, resources []model.ResourceRecord) []model.ExtractedCandidate {
	seen := make(map[string]struct{})
	all := make([]model.ExtractedCandidate, 0)
	resourceMap := make(map[string]model.ResourceRecord, len(resources))
	for _, item := range resources {
		resourceMap[item.URL] = item
	}

	merge := func(items []model.ExtractedCandidate) {
		for _, item := range items {
			key := item.RawValue + "|" + item.MethodHint + "|" + item.SourceURL + "|" + item.DiscoverRule
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, item)
		}
	}

	entryInfo := DetectFrontendFromHTML(html, nil)
	if record, exists := resourceMap[targetURL]; exists && record.Frontend != nil {
		entryInfo = record.Frontend
	}
	merge(addFrontendTagsToExtracted(ExtractFromText(html, targetURL, "res-target", "html"), entryInfo))
	for _, file := range sourceFiles {
		if file.Error != "" {
			continue
		}
		record := resourceMap[file.URL]
		merge(ExtractKnownFrontendCandidates(file.Content, file.URL, record.ResourceID, file.SourceType, file.Frontend))
		for _, restored := range file.RestoredSources {
			if restored.Content == "" {
				continue
			}
			sourceURL := file.URL
			if restored.Name != "" {
				sourceURL = file.URL + "#" + restored.Name
			}
			items := ExtractKnownFrontendCandidates(restored.Content, sourceURL, record.ResourceID, restored.SourceType, file.Frontend)
			items = append(items, ExtractFromText(restored.Content, sourceURL, record.ResourceID, restored.SourceType)...)
			for idx := range items {
				items[idx].HintTags = mergeStringTags(mergeStringTags(items[idx].HintTags, FrontendTags(file.Frontend)), restored.Tags)
				if items[idx].DiscoverRule == "" {
					items[idx].DiscoverRule = "sourcemap-sources-content"
				}
			}
			merge(items)
		}
		merge(addFrontendTagsToExtracted(ExtractFromText(file.Content, file.URL, record.ResourceID, file.SourceType), file.Frontend))
	}

	return all
}

func addFrontendTagsToExtracted(items []model.ExtractedCandidate, info *model.FrontendInfo) []model.ExtractedCandidate {
	tags := FrontendTags(info)
	if len(tags) == 0 {
		return items
	}
	for idx := range items {
		items[idx].HintTags = mergeStringTags(items[idx].HintTags, tags)
	}
	return items
}

func addCandidate(raw string, discoverRule string, sourceURL string, sourceResourceID string, sourceType string, seen map[string]struct{}, results *[]model.ExtractedCandidate) {
	addCandidateWithHints(raw, "", nil, discoverRule, sourceURL, sourceResourceID, sourceType, seen, results)
}

func addCandidateWithHints(raw string, methodHint string, hintTags []string, discoverRule string, sourceURL string, sourceResourceID string, sourceType string, seen map[string]struct{}, results *[]model.ExtractedCandidate) {
	candidate := cleanCandidate(raw)
	if !looksLikeAPI(candidate) {
		return
	}
	upperMethod := strings.ToUpper(strings.TrimSpace(methodHint))
	baseKey := candidate + "|" + sourceURL + "|" + discoverRule
	anyMethodKey := baseKey + "|method-aware"
	if upperMethod == "" {
		if _, exists := seen[anyMethodKey]; exists {
			return
		}
	}
	key := candidate + "|" + upperMethod + "|" + sourceURL + "|" + discoverRule
	if _, exists := seen[key]; exists {
		return
	}

	seen[key] = struct{}{}
	if upperMethod != "" {
		seen[anyMethodKey] = struct{}{}
	}
	*results = append(*results, model.ExtractedCandidate{
		RawValue:         candidate,
		MethodHint:       upperMethod,
		HintTags:         append([]string(nil), hintTags...),
		SourceResourceID: sourceResourceID,
		SourceURL:        sourceURL,
		SourceType:       sourceType,
		DiscoverRule:     discoverRule,
	})
}

func preferRicherExtractedCandidates(items []model.ExtractedCandidate) []model.ExtractedCandidate {
	if len(items) <= 1 {
		return items
	}
	best := make(map[string]model.ExtractedCandidate, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		key := item.RawValue + "|" + item.SourceURL + "|" + item.SourceResourceID + "|" + item.SourceType
		current, exists := best[key]
		if !exists {
			best[key] = item
			order = append(order, key)
			continue
		}
		if extractedCandidateScore(item) > extractedCandidateScore(current) {
			best[key] = item
		}
	}
	out := make([]model.ExtractedCandidate, 0, len(order))
	for _, key := range order {
		out = append(out, best[key])
	}
	return out
}

func extractedCandidateScore(item model.ExtractedCandidate) int {
	score := 0
	if item.MethodHint != "" {
		score += 10
	}
	score += len(item.HintTags)
	switch item.DiscoverRule {
	case "openapi-path", "xhr-open", "axios-method", "axios-object", "jquery-shortcut", "jquery-ajax", "new-url":
		score += 4
	case "fetch-call", "request-constructor", "websocket-constructor":
		score += 3
	case "object-property", "object-expression":
		score += 2
	case "quoted-url", "api-keyword":
		score += 1
	}
	return score
}

func cleanCandidate(raw string) string {
	candidate := strings.TrimSpace(raw)
	candidate = extractStaticPrefix(candidate)
	candidate = strings.Trim(candidate, "\"'`")
	candidate = strings.ReplaceAll(candidate, `\/`, `/`)
	candidate = strings.ReplaceAll(candidate, `\u002f`, `/`)
	candidate = strings.ReplaceAll(candidate, `\u002F`, `/`)
	candidate = strings.TrimRight(candidate, `,;.)]}`)
	if strings.HasPrefix(candidate, "./") {
		candidate = strings.TrimPrefix(candidate, ".")
	}
	return candidate
}

func extractStaticPrefix(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	candidate = strings.Trim(candidate, "\"'`")
	if strings.HasPrefix(candidate, "${") {
		if idx := strings.Index(candidate, "}"); idx >= 0 && idx+1 < len(candidate) {
			candidate = candidate[idx+1:]
		}
	}
	if idx := strings.Index(candidate, "${"); idx >= 0 {
		candidate = candidate[:idx]
	}
	if idx := strings.Index(candidate, "+"); idx >= 0 {
		candidate = candidate[:idx]
	}
	if idx := strings.Index(candidate, "||"); idx >= 0 {
		candidate = candidate[:idx]
	}
	if idx := strings.Index(candidate, "&&"); idx >= 0 {
		candidate = candidate[:idx]
	}
	if idx := strings.Index(candidate, ")"); idx >= 0 {
		candidate = candidate[:idx]
	}
	if idx := strings.Index(candidate, ";"); idx >= 0 {
		candidate = candidate[:idx]
	}

	candidate = strings.TrimSpace(candidate)
	return strings.Trim(candidate, "\"'`")
}

func looksLikeAPI(candidate string) bool {
	if candidate == "" {
		return false
	}
	lower := strings.ToLower(candidate)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "mailto:") {
		return false
	}
	if strings.HasPrefix(lower, "ws://") || strings.HasPrefix(lower, "wss://") {
		return true
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") || strings.HasPrefix(candidate, "//") {
		return true
	}
	if strings.HasPrefix(candidate, "${") && (strings.Contains(lower, "/api") || strings.Contains(lower, "graphql")) {
		return true
	}
	if strings.HasPrefix(lower, "api/") || strings.HasPrefix(lower, "graphql") || strings.HasPrefix(lower, "rest/") || strings.HasPrefix(lower, "v1/") || strings.HasPrefix(lower, "v2/") {
		return true
	}
	if !strings.HasPrefix(candidate, "/") {
		if strings.Contains(candidate, "+") && (strings.Contains(lower, "/api") || strings.Contains(lower, "graphql")) {
			return true
		}
		return false
	}
	if hasStaticSuffix(lower) {
		return false
	}
	if strings.Contains(lower, "api") || strings.Contains(lower, "graphql") || strings.Contains(lower, "rest") {
		return true
	}
	if businessPathPattern.MatchString(candidate) {
		return true
	}
	if strings.Contains(candidate, "?") && strings.Count(candidate, "/") >= 1 {
		return true
	}
	return false
}

func hasStaticSuffix(lower string) bool {
	staticSuffixes := []string{
		".css", ".gif", ".ico", ".jpeg", ".jpg", ".js", ".map", ".png", ".svg", ".webp", ".woff", ".woff2",
		".mp3", ".mp4", ".pdf", ".txt", ".xml", ".zip",
	}
	for _, suffix := range staticSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// ExtractResponseCandidates tries to recover additional API-like candidates from verified text or JSON responses.
func ExtractResponseCandidates(results []model.APIResult) []model.ExtractedCandidate {
	seen := make(map[string]struct{})
	out := make([]model.ExtractedCandidate, 0)

	for _, result := range results {
		if result.StatusCode < 200 || result.StatusCode >= 300 {
			continue
		}
		lowerType := strings.ToLower(result.ContentType)
		if !strings.Contains(lowerType, "json") && !strings.Contains(lowerType, "text") {
			continue
		}
		extracted := ExtractFromText(result.ResponseSample, result.APIURL, result.SourceResourceID, "response-body")
		for _, item := range extracted {
			key := item.RawValue + "|" + item.MethodHint + "|" + item.SourceURL + "|" + item.DiscoverRule
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, item)
		}
	}

	return out
}

// ExtractCandidatesFromResourceBody parses discovery-oriented resources such as robots, sitemap, and manifest files.
func ExtractCandidatesFromResourceBody(body string, resource model.ResourceRecord) []model.ExtractedCandidate {
	switch resource.Type {
	case "robots":
		return extractRobotsCandidates(body, resource)
	case "sitemap":
		return extractSitemapCandidates(body, resource)
	case "manifest":
		return extractManifestCandidates(body, resource)
	case "json":
		if looksLikeOpenAPIDoc(body) {
			return extractOpenAPICandidates(body, resource)
		}
		return ExtractFromText(body, resource.URL, resource.ResourceID, resource.Type)
	default:
		return ExtractFromText(body, resource.URL, resource.ResourceID, resource.Type)
	}
}

func extractRobotsCandidates(body string, resource model.ResourceRecord) []model.ExtractedCandidate {
	lines := strings.Split(body, "\n")
	out := make([]model.ExtractedCandidate, 0)
	seen := make(map[string]struct{})

	appendItem := func(raw string, rule string) {
		raw = cleanCandidate(raw)
		if raw == "" {
			return
		}
		key := raw + "|" + rule
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, model.ExtractedCandidate{
			RawValue:         raw,
			SourceResourceID: resource.ResourceID,
			SourceURL:        resource.URL,
			SourceType:       resource.Type,
			DiscoverRule:     rule,
		})
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "allow:"), strings.HasPrefix(lower, "disallow:"):
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && looksLikeAPI(strings.TrimSpace(parts[1])) {
				appendItem(parts[1], "robots-rule")
			}
		case strings.HasPrefix(lower, "sitemap:"):
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				appendItem(parts[1], "robots-sitemap")
			}
		}
	}
	return out
}

func extractSitemapCandidates(body string, resource model.ResourceRecord) []model.ExtractedCandidate {
	type urlEntry struct {
		Loc string `xml:"loc"`
	}
	type urlSet struct {
		URLs []urlEntry `xml:"url"`
	}
	type sitemapEntry struct {
		Loc string `xml:"loc"`
	}
	type sitemapIndex struct {
		Maps []sitemapEntry `xml:"sitemap"`
	}

	out := make([]model.ExtractedCandidate, 0)
	seen := make(map[string]struct{})
	appendItem := func(raw string, rule string) {
		raw = cleanCandidate(raw)
		if raw == "" {
			return
		}
		key := raw + "|" + rule
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, model.ExtractedCandidate{
			RawValue:         raw,
			SourceResourceID: resource.ResourceID,
			SourceURL:        resource.URL,
			SourceType:       resource.Type,
			DiscoverRule:     rule,
		})
	}

	var set urlSet
	if err := xml.Unmarshal([]byte(body), &set); err == nil {
		for _, item := range set.URLs {
			appendItem(item.Loc, "sitemap-url")
		}
	}
	var index sitemapIndex
	if err := xml.Unmarshal([]byte(body), &index); err == nil {
		for _, item := range index.Maps {
			appendItem(item.Loc, "sitemap-index")
		}
	}
	return out
}

func extractManifestCandidates(body string, resource model.ResourceRecord) []model.ExtractedCandidate {
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil
	}
	out := make([]model.ExtractedCandidate, 0)
	seen := make(map[string]struct{})

	var walk func(node any)
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
			candidate := cleanCandidate(value)
			if candidate == "" {
				return
			}
			if !looksLikeAPI(candidate) && !strings.HasPrefix(candidate, "/") && !strings.HasPrefix(candidate, "http://") && !strings.HasPrefix(candidate, "https://") {
				return
			}
			key := candidate + "|manifest-value"
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			out = append(out, model.ExtractedCandidate{
				RawValue:         candidate,
				SourceResourceID: resource.ResourceID,
				SourceURL:        resource.URL,
				SourceType:       resource.Type,
				DiscoverRule:     "manifest-value",
			})
		}
	}

	walk(payload)
	return out
}

func looksLikeOpenAPIDoc(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, `"openapi"`) || strings.Contains(lower, `"swagger"`)
}

func extractOpenAPICandidates(body string, resource model.ResourceRecord) []model.ExtractedCandidate {
	type operationMap map[string]any
	type doc struct {
		Servers  []openAPIServer         `json:"servers"`
		Security []map[string][]string   `json:"security"`
		Paths    map[string]operationMap `json:"paths"`
	}

	var payload doc
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil
	}
	if len(payload.Paths) == 0 {
		return nil
	}

	allowedMethods := map[string]struct{}{
		"GET": {}, "POST": {}, "PUT": {}, "DELETE": {}, "PATCH": {}, "HEAD": {}, "OPTIONS": {},
	}
	seen := make(map[string]struct{})
	out := make([]model.ExtractedCandidate, 0, len(payload.Paths))
	basePrefix := extractOpenAPIBasePath(payload.Servers, resource.URL)
	globalProtected := len(payload.Security) > 0

	for rawPath, operations := range payload.Paths {
		pathValue := applyOpenAPIBasePath(basePrefix, cleanCandidate(rawPath))
		if pathValue == "" || !strings.HasPrefix(pathValue, "/") {
			continue
		}
		for method, operation := range operations {
			upperMethod := strings.ToUpper(strings.TrimSpace(method))
			if _, ok := allowedMethods[upperMethod]; !ok {
				continue
			}
			key := upperMethod + " " + pathValue
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			hintTags := []string{"openapi-doc"}
			if globalProtected || openAPIOperationProtected(operation) {
				hintTags = append(hintTags, "auth-required-hint")
			}
			if openAPIOperationInternal(operation) || strings.Contains(strings.ToLower(pathValue), "internal") {
				hintTags = append(hintTags, "internal-doc-hint")
			}
			out = append(out, model.ExtractedCandidate{
				RawValue:         pathValue,
				MethodHint:       upperMethod,
				HintTags:         hintTags,
				SourceResourceID: resource.ResourceID,
				SourceURL:        resource.URL,
				SourceType:       resource.Type,
				DiscoverRule:     "openapi-path",
			})
		}
	}

	return out
}

func extractOpenAPIBasePath(servers []openAPIServer, resourceURL string) string {
	if len(servers) == 0 {
		return ""
	}
	resourceParsed, err := url.Parse(resourceURL)
	if err != nil {
		return ""
	}
	for _, item := range servers {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		serverParsed, err := resourceParsed.Parse(item.URL)
		if err != nil {
			continue
		}
		if serverParsed.Path == "" || serverParsed.Path == "/" {
			return ""
		}
		return strings.TrimRight(serverParsed.Path, "/")
	}
	return ""
}

func applyOpenAPIBasePath(basePath string, rawPath string) string {
	if rawPath == "" {
		return ""
	}
	if basePath == "" || strings.HasPrefix(rawPath, basePath+"/") || rawPath == basePath {
		return rawPath
	}
	return path.Clean(basePath + "/" + strings.TrimLeft(rawPath, "/"))
}

func openAPIOperationProtected(operation any) bool {
	op, ok := operation.(map[string]any)
	if !ok {
		return false
	}
	if security, exists := op["security"]; exists {
		if items, ok := security.([]any); ok && len(items) > 0 {
			return true
		}
	}
	return false
}

func openAPIOperationInternal(operation any) bool {
	op, ok := operation.(map[string]any)
	if !ok {
		return false
	}
	if value, exists := op["x-internal"]; exists {
		if internal, ok := value.(bool); ok && internal {
			return true
		}
	}
	if tags, exists := op["tags"]; exists {
		if items, ok := tags.([]any); ok {
			for _, item := range items {
				if text, ok := item.(string); ok && strings.Contains(strings.ToLower(text), "internal") {
					return true
				}
			}
		}
	}
	return false
}
