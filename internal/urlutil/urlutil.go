package urlutil

import (
	"net/url"
	"path"
	"strings"
)

// NormalizeCandidate 基于基础 URL 补全原始候选地址，并应用基础过滤规则。
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
	return resolved.String(), true
}

// sameOriginURL 判断两个已解析 URL 是否具有相同协议和主机。
func sameOriginURL(left *url.URL, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

// isStaticResource 判断路径是否指向常见静态资源类型。
func isStaticResource(rawPath string) bool {
	ext := strings.ToLower(path.Ext(rawPath))
	switch ext {
	case ".css", ".gif", ".ico", ".jpeg", ".jpg", ".png", ".svg", ".webp", ".woff", ".woff2":
		return true
	default:
		return false
	}
}
