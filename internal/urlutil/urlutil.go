package urlutil

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{[^{}]+\}|:[A-Za-z_][A-Za-z0-9_]*`)
var concatBasePattern = regexp.MustCompile("(?i)^(?:window\\.)?(?:baseurl|apibase|apihost)\\s*\\+\\s*[\"'`]([^\"'`]+)[\"'`]$")

// NormalizeCandidate resolves a candidate against a base URL and applies basic filters.
func NormalizeCandidate(raw string, base string, sameOrigin bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "//") {
		baseURL, err := url.Parse(base)
		if err != nil {
			return "", false
		}
		raw = baseURL.Scheme + ":" + raw
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", false
	}

	raw = replaceBaseTokens(raw, baseURL)

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	resolved := baseURL.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	if sameOrigin && !sameOriginURL(resolved, baseURL) {
		return "", false
	}
	if isStaticResource(resolved.Path) {
		return "", false
	}

	resolved.Fragment = ""
	resolved = normalizeDefaultPort(resolved)
	return resolved.String(), true
}

// NormalizeForResult returns normalized URL parts for result storage.
func NormalizeForResult(raw string, base string, sameOrigin bool) (normalizedURL string, normalizedPath string, sampleQuery string, isParameterized bool, ok bool) {
	normalizedURL, ok = NormalizeCandidate(raw, base, sameOrigin)
	if !ok {
		return "", "", "", false, false
	}

	parsed, err := url.Parse(normalizedURL)
	if err != nil {
		return "", "", "", false, false
	}

	normalizedPath = parsed.EscapedPath()
	if normalizedPath == "" {
		normalizedPath = "/"
	}
	sampleQuery = parsed.RawQuery
	isParameterized = placeholderPattern.MatchString(normalizedPath)
	return normalizedURL, normalizedPath, sampleQuery, isParameterized, true
}

// Origin extracts the origin portion from a URL.
func Origin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	parsed = normalizeDefaultPort(parsed)
	return parsed.Scheme + "://" + parsed.Host
}

func sameOriginURL(left *url.URL, right *url.URL) bool {
	left = normalizeDefaultPort(left)
	right = normalizeDefaultPort(right)
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func isStaticResource(rawPath string) bool {
	ext := strings.ToLower(path.Ext(rawPath))
	switch ext {
	case ".css", ".gif", ".ico", ".jpeg", ".jpg", ".png", ".svg", ".webp", ".woff", ".woff2", ".js", ".mjs", ".cjs", ".map", ".mp3", ".mp4", ".pdf", ".zip":
		return true
	default:
		return false
	}
}

func replaceBaseTokens(raw string, baseURL *url.URL) string {
	origin := normalizeDefaultPort(baseURL).Scheme + "://" + normalizeDefaultPort(baseURL).Host
	replacer := strings.NewReplacer(
		"${baseURL}", origin,
		"${baseUrl}", origin,
		"${BASE_URL}", origin,
		"${origin}", origin,
		"${ORIGIN}", origin,
	)
	raw = replacer.Replace(raw)

	if matches := concatBasePattern.FindStringSubmatch(strings.TrimSpace(raw)); len(matches) == 2 {
		return origin + matches[1]
	}
	return raw
}

func normalizeDefaultPort(input *url.URL) *url.URL {
	if input == nil {
		return input
	}
	cloned := *input
	port := cloned.Port()
	switch {
	case strings.EqualFold(cloned.Scheme, "http") && port == "80":
		cloned.Host = cloned.Hostname()
	case strings.EqualFold(cloned.Scheme, "https") && port == "443":
		cloned.Host = cloned.Hostname()
	}
	return &cloned
}
