package core

import (
	"regexp"
	"strings"

	"apiextractor/internal/model"
)

var (
	quotedURLPattern       = regexp.MustCompile("(?i)[\"'`]((?:https?:)?//[^\"'`\\s<>]+|/[A-Za-z0-9._~!$&'()*+,;=:@%/?#\\[\\]-]+)[\"'`]")
	apiKeywordPattern      = regexp.MustCompile("(?i)[\"'`]((?:\\.?\\.?/)?[^\"'`\\s<>]*(?:api|graphql|rest|v[0-9]+)[^\"'`\\s<>]*)[\"'`]")
	fetchPattern           = regexp.MustCompile("(?is)\\bfetch\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	xhrOpenPattern         = regexp.MustCompile("(?is)\\.open\\s*\\(\\s*[\"'`][A-Z]+[\"'`]\\s*,\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosPattern           = regexp.MustCompile("(?is)\\baxios(?:\\.[a-z]+)?\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosObjectURLPattern  = regexp.MustCompile("(?is)\\baxios\\s*\\(\\s*\\{[^{}]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	jqueryAjaxURLPattern   = regexp.MustCompile("(?is)\\$\\.(?:ajax|get|post|getJSON)\\s*\\([^)]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	jqueryShortcutPattern  = regexp.MustCompile("(?is)\\$\\.(?:get|post|getJSON)\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	requestObjectURLPattern = regexp.MustCompile("(?is)\\b(?:url|path|endpoint|uri|baseURL|baseUrl)\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	graphQLOperationPattern = regexp.MustCompile("(?is)\\b(?:query|mutation)\\s+[A-Za-z0-9_]*\\s*(?:\\([^)]*\\))?\\s*\\{")
	businessPathPattern     = regexp.MustCompile(`(?i)^/(?:v[0-9]+|admin|auth|user|users|account|accounts|order|orders|pay|payment|member|members|tenant|tenants|system|manage|backend|console)(?:/|$)`)
)

// ExtractFromText 从 HTML、JavaScript、source map 或 JSON 文本中提取疑似 API 路径或 URL。
func ExtractFromText(text string) []string {
	seen := make(map[string]struct{})
	results := make([]string, 0)

	mergeMatches := func(matches [][]string) {
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			addCandidate(match[1], seen, &results)
		}
	}

	mergeMatches(fetchPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(xhrOpenPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(axiosPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(axiosObjectURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(jqueryAjaxURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(jqueryShortcutPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(requestObjectURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(quotedURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(apiKeywordPattern.FindAllStringSubmatch(text, -1))

	if graphQLOperationPattern.MatchString(text) {
		addCandidate("/graphql", seen, &results)
	}

	return results
}

// ExtractAll 汇总页面 HTML 和已下载源文件中的 API 候选。
func ExtractAll(html string, jsFiles []model.SourceFile) []string {
	seen := make(map[string]struct{})
	all := make([]string, 0)

	merge := func(items []string) {
		for _, item := range items {
			addCandidate(item, seen, &all)
		}
	}

	merge(ExtractFromText(html))
	for _, file := range jsFiles {
		if file.Error != "" {
			continue
		}
		merge(ExtractFromText(file.Content))
	}

	return all
}

func addCandidate(raw string, seen map[string]struct{}, results *[]string) {
	candidate := cleanCandidate(raw)
	if !looksLikeAPI(candidate) {
		return
	}
	if _, exists := seen[candidate]; exists {
		return
	}

	seen[candidate] = struct{}{}
	*results = append(*results, candidate)
}

func cleanCandidate(raw string) string {
	candidate := strings.TrimSpace(raw)
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

func looksLikeAPI(candidate string) bool {
	if candidate == "" {
		return false
	}
	lower := strings.ToLower(candidate)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "mailto:") {
		return false
	}
	if strings.Contains(candidate, "${") || strings.Contains(candidate, "+") {
		return false
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") || strings.HasPrefix(candidate, "//") {
		return true
	}
	if strings.HasPrefix(lower, "api/") || strings.HasPrefix(lower, "graphql") || strings.HasPrefix(lower, "rest/") || strings.HasPrefix(lower, "v1/") || strings.HasPrefix(lower, "v2/") {
		return true
	}
	if !strings.HasPrefix(candidate, "/") {
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
