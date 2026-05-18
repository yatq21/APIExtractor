package core

import "apiextractor/internal/urlutil"

// NormalizeURL 规范化单个候选 URL，并返回是否应该保留。
func NormalizeURL(rawURL string, baseURL string, sameOrigin bool) (string, bool) {
	return urlutil.NormalizeCandidate(rawURL, baseURL, sameOrigin)
}

// NormalizeURLs 批量规范化、过滤并去重候选 URL。
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
