package core

import (
	"net/url"
	"sort"
	"strings"

	"apiextractor/internal/model"
	"apiextractor/internal/urlutil"
)

// NormalizeURL normalizes one candidate URL and returns whether it should be kept.
func NormalizeURL(rawURL string, baseURL string, sameOrigin bool) (string, bool) {
	return urlutil.NormalizeCandidate(rawURL, baseURL, sameOrigin)
}

// NormalizeURLs normalizes and deduplicates candidate URLs.
func NormalizeURLs(rawURLs []string, baseURL string, sameOrigin bool) []string {
	seen := make(map[string]struct{})
	results := make([]string, 0, len(rawURLs))

	for _, item := range rawURLs {
		normalized, ok := NormalizeURL(item, baseURL, sameOrigin)
		if !ok {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		results = append(results, normalized)
	}

	return results
}

// NormalizeCandidates converts extracted items into structured API candidates.
func NormalizeCandidates(items []model.ExtractedCandidate, baseURL string, sameOrigin bool) []model.APICandidate {
	seen := make(map[string]struct{})
	results := make([]model.APICandidate, 0, len(items))

	for idx, item := range items {
		normalized, pathValue, sampleQuery, isParameterized, usedRootFallback, ok := normalizeExtractedCandidate(item, baseURL, sameOrigin)
		if !ok {
			continue
		}
		method := guessMethod(item.RawValue, item.MethodHint)
		key := method + " " + normalized
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		results = append(results, model.APICandidate{
			CandidateID:      candidateID(idx + 1),
			RawValue:         item.RawValue,
			NormalizedURL:    normalized,
			MethodGuess:      method,
			Category:         classifyCandidate(pathValue, sampleQuery),
			SourceResourceID: item.SourceResourceID,
			SourceURL:        item.SourceURL,
			SourceType:       item.SourceType,
			DiscoverRule:     item.DiscoverRule,
			SameOrigin:       sameOriginMatch(normalized, baseURL),
			Confidence:       inferCandidateConfidence(item.RawValue, item.DiscoverRule, sampleQuery, isParameterized, usedRootFallback),
			Tags:             mergeStringTags(inferCandidateTags(pathValue, sampleQuery, isParameterized), item.HintTags),
			IsParameterized:  isParameterized,
			Path:             pathValue,
			SampleQuery:      sampleQuery,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].NormalizedURL < results[j].NormalizedURL
	})
	return results
}

func normalizeExtractedCandidate(item model.ExtractedCandidate, targetURL string, sameOrigin bool) (normalizedURL string, normalizedPath string, sampleQuery string, isParameterized bool, usedRootFallback bool, ok bool) {
	resolutionBase := strings.TrimSpace(item.SourceURL)
	if resolutionBase == "" {
		resolutionBase = targetURL
	}

	if rootValue, fallback := rootNormalizedAPIFallback(item.RawValue); fallback {
		if normalizedURL, normalizedPath, sampleQuery, isParameterized, ok = normalizeAgainstTargetBoundary(rootValue, targetURL, targetURL, sameOrigin); ok {
			return normalizedURL, normalizedPath, sampleQuery, isParameterized, true, true
		}
	}

	normalizedURL, normalizedPath, sampleQuery, isParameterized, ok = normalizeAgainstTargetBoundary(item.RawValue, resolutionBase, targetURL, sameOrigin)
	return normalizedURL, normalizedPath, sampleQuery, isParameterized, false, ok
}

func normalizeAgainstTargetBoundary(raw string, resolutionBase string, targetURL string, sameOrigin bool) (normalizedURL string, normalizedPath string, sampleQuery string, isParameterized bool, ok bool) {
	normalizedURL, normalizedPath, sampleQuery, isParameterized, ok = urlutil.NormalizeForResult(raw, resolutionBase, false)
	if !ok {
		return "", "", "", false, false
	}
	if sameOrigin && !sameOriginMatch(normalizedURL, targetURL) {
		return "", "", "", false, false
	}
	return normalizedURL, normalizedPath, sampleQuery, isParameterized, true
}

func rootNormalizedAPIFallback(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "../") {
		return "", false
	}

	for strings.HasPrefix(trimmed, "../") {
		trimmed = strings.TrimPrefix(trimmed, "../")
	}
	trimmed = strings.TrimLeft(trimmed, "/")
	if !hasAPISemanticPrefix(trimmed) {
		return "", false
	}
	return "/" + trimmed, true
}

func hasAPISemanticPrefix(raw string) bool {
	lower := strings.ToLower(strings.TrimLeft(raw, "/"))
	first := lower
	if idx := strings.IndexAny(first, "/?"); idx >= 0 {
		first = first[:idx]
	}
	switch first {
	case "api", "graphql", "auth", "admin", "rest", "v1", "v2":
		return true
	default:
		return false
	}
}

func candidateID(index int) string {
	return "cand-" + intString(index)
}

func intString(v int) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	return string(buf)
}

func guessMethod(raw string, hint string) string {
	if hint != "" {
		return strings.ToUpper(strings.TrimSpace(hint))
	}
	if strings.Contains(strings.ToLower(raw), "graphql") {
		return "POST"
	}
	return "GET"
}

func sameOriginMatch(left string, right string) bool {
	leftURL, err := url.Parse(left)
	if err != nil {
		return false
	}
	rightURL, err := url.Parse(right)
	if err != nil {
		return false
	}
	return strings.EqualFold(leftURL.Scheme, rightURL.Scheme) && strings.EqualFold(leftURL.Host, rightURL.Host)
}

func inferCandidateConfidence(raw string, discoverRule string, sampleQuery string, isParameterized bool, usedRootFallback bool) string {
	switch {
	case usedRootFallback:
		return "medium"
	case strings.Contains(raw, "${") || strings.Contains(raw, "+"):
		return "low"
	case strings.HasPrefix(strings.TrimSpace(raw), "../") && !isRootAPIPath(raw):
		return "low"
	case discoverRule == "html-a-href":
		return "low"
	case isParameterized:
		return "medium"
	case sampleQuery != "":
		return "medium"
	default:
		return "high"
	}
}

func isRootAPIPath(raw string) bool {
	_, ok := rootNormalizedAPIFallback(raw)
	return ok
}

func inferCandidateTags(pathValue string, sampleQuery string, isParameterized bool) []string {
	tags := make([]string, 0, 6)
	if sampleQuery != "" {
		tags = append(tags, "has-query")
	}
	if isParameterized {
		tags = append(tags, "parameterized")
	}
	lower := strings.ToLower(pathValue)
	if strings.Contains(lower, "admin") || strings.Contains(lower, "manage") || strings.Contains(lower, "system") {
		tags = append(tags, "admin-semantic")
	}
	if strings.Contains(lower, "internal") {
		tags = append(tags, "internal-semantic")
	}
	if strings.Contains(lower, "auth") || strings.Contains(lower, "login") || strings.Contains(lower, "token") || strings.Contains(lower, "session") {
		tags = append(tags, "auth-semantic")
	}
	if strings.Contains(lower, "profile") || strings.Contains(lower, "orders") || strings.Contains(lower, "account") || strings.Contains(lower, "me") {
		tags = append(tags, "user-data-semantic")
	}
	return tags
}

func mergeStringTags(base []string, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	for _, item := range extra {
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func classifyCandidate(pathValue string, sampleQuery string) string {
	lower := strings.ToLower(pathValue)
	switch {
	case strings.Contains(lower, "graphql"):
		return "graphql"
	case strings.Contains(lower, "swagger") || strings.Contains(lower, "openapi") || strings.Contains(lower, "api-docs"):
		return "api-doc"
	case strings.Contains(lower, "internal"):
		return "internal-api"
	case strings.Contains(lower, "auth") || strings.Contains(lower, "login"):
		return "auth-endpoint"
	case strings.Contains(lower, "profile") || strings.Contains(lower, "orders") || strings.Contains(lower, "account") || strings.Contains(lower, "me"):
		return "authenticated-api"
	case strings.Contains(lower, "admin") || strings.Contains(lower, "manage") || strings.Contains(lower, "system"):
		return "admin-endpoint"
	case sampleQuery != "":
		return "public-api"
	default:
		return "public-api"
	}
}
