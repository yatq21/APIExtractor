package core

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

var (
	scriptSrcPattern     = regexp.MustCompile(`(?i)<script\b[^>]*\bsrc\s*=\s*["']([^"']+)["']`)
	linkHrefPattern      = regexp.MustCompile(`(?i)<link\b[^>]*\bhref\s*=\s*["']([^"']+)["']`)
	sourceMapPattern     = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL\s*=\s*([^\s]+)`)
	dynamicImportPattern = regexp.MustCompile(`(?i)(?:import\s*\(\s*|from\s+|import\s+)["']([^"']+\.(?:m?js|map|json)(?:\?[^"']*)?)["']`)
	chunkPattern         = regexp.MustCompile(`["']([^"']*(?:chunk|bundle|vendor|runtime|app|main)[^"']*\.(?:m?js|map|json)(?:\?[^"']*)?)["']`)
)

// FetchURL requests the target URL and returns a bounded text response body.
func FetchURL(rawURL string, cfg config.Config) (string, error) {
	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}

	for key, value := range cfg.DefaultHeaders {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ExtractJSURLs keeps backward compatibility and delegates to source discovery.
func ExtractJSURLs(html string, baseURL string, sameOrigin bool) []string {
	return ExtractSourceURLs(html, baseURL, sameOrigin)
}

// ExtractSourceURLs collects script, preload, and source-map style source URLs from HTML.
func ExtractSourceURLs(html string, baseURL string, sameOrigin bool) []string {
	seen := make(map[string]struct{})
	urls := make([]string, 0)

	addMatches := func(matches [][]string) {
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			addSourceURL(match[1], baseURL, sameOrigin, seen, &urls)
		}
	}

	addMatches(scriptSrcPattern.FindAllStringSubmatch(html, -1))
	addMatches(linkHrefPattern.FindAllStringSubmatch(html, -1))

	return urls
}

// FetchJSFiles keeps backward compatibility and delegates to source fetching.
func FetchJSFiles(jsURLs []string, cfg config.Config) []model.SourceFile {
	return FetchSourceFiles(jsURLs, cfg)
}

// FetchSourceFiles downloads initial source files and recursively discovers chunks/imports/maps.
func FetchSourceFiles(sourceURLs []string, cfg config.Config) []model.SourceFile {
	if cfg.MaxSourceFiles <= 0 {
		cfg.MaxSourceFiles = 40
	}

	queue := append([]string(nil), sourceURLs...)
	queued := make(map[string]struct{}, len(queue))
	downloaded := make(map[string]struct{})
	files := make([]model.SourceFile, 0, len(sourceURLs))

	for _, item := range queue {
		queued[item] = struct{}{}
	}

	for len(queue) > 0 && len(files) < cfg.MaxSourceFiles {
		item := queue[0]
		queue = queue[1:]
		if _, exists := downloaded[item]; exists {
			continue
		}

		downloaded[item] = struct{}{}
		body, err := FetchURL(item, cfg)
		file := model.SourceFile{
			URL:        item,
			SourceType: detectSourceType(item),
		}
		if err != nil {
			file.ErrorType = classifyError(err)
			file.Error = truncateError(err.Error())
			files = append(files, file)
			continue
		}

		file.Content = body
		files = append(files, file)

		nextURLs := ExtractNestedSourceURLs(body, item, cfg.SameOrigin)
		for _, next := range nextURLs {
			if _, exists := queued[next]; exists {
				continue
			}
			if _, exists := downloaded[next]; exists {
				continue
			}
			queued[next] = struct{}{}
			queue = append(queue, next)
		}
	}

	return files
}

// ExtractNestedSourceURLs discovers source maps, dynamic imports, and common build chunks.
func ExtractNestedSourceURLs(text string, baseURL string, sameOrigin bool) []string {
	seen := make(map[string]struct{})
	urls := make([]string, 0)

	addMatches := func(matches [][]string) {
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			addSourceURL(match[1], baseURL, sameOrigin, seen, &urls)
		}
	}

	addMatches(dynamicImportPattern.FindAllStringSubmatch(text, -1))
	addMatches(chunkPattern.FindAllStringSubmatch(text, -1))
	addMatches(sourceMapPattern.FindAllStringSubmatch(text, -1))

	return urls
}

func addSourceURL(raw string, baseURL string, sameOrigin bool, seen map[string]struct{}, urls *[]string) {
	normalized, ok := normalizeSourceURL(raw, baseURL, sameOrigin)
	if !ok {
		return
	}
	if _, exists := seen[normalized]; exists {
		return
	}

	seen[normalized] = struct{}{}
	*urls = append(*urls, normalized)
}

func normalizeSourceURL(raw string, baseURL string, sameOrigin bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return "", false
	}

	if strings.HasPrefix(raw, "//") {
		baseParsed, err := url.Parse(baseURL)
		if err != nil {
			return "", false
		}
		raw = baseParsed.Scheme + ":" + raw
	}

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	resolved := baseParsed.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	if sameOrigin && !strings.EqualFold(resolved.Scheme, baseParsed.Scheme) {
		return "", false
	}
	if sameOrigin && !strings.EqualFold(resolved.Host, baseParsed.Host) {
		return "", false
	}
	if !isTextSource(resolved.String()) {
		return "", false
	}

	resolved.Fragment = ""
	return resolved.String(), true
}

func isTextSource(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".js", ".mjs", ".map", ".json":
		return true
	default:
		return false
	}
}

func detectSourceType(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "source"
	}

	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".map":
		return "sourcemap"
	case ".json":
		return "json"
	case ".mjs":
		return "module"
	default:
		return "javascript"
	}
}
