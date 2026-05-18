package core

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
	"apiextractor/internal/urlutil"
)

var scriptSrcPattern = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)

// FetchURL 请求指定 URL，并返回限制大小后的文本响应体。
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

// ExtractJSURLs 从 HTML 中提取外链脚本地址，并基于页面 URL 补全。
func ExtractJSURLs(html string, baseURL string, sameOrigin bool) []string {
	matches := scriptSrcPattern.FindAllStringSubmatch(html, -1)
	seen := make(map[string]struct{})
	urls := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		normalized, ok := urlutil.NormalizeCandidate(match[1], baseURL, sameOrigin)
		if !ok {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(normalized), ".js") {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		urls = append(urls, normalized)
	}

	return urls
}

// FetchJSFiles 批量下载 JavaScript 文件，并在结果中保留单个文件的错误信息。
func FetchJSFiles(jsURLs []string, cfg config.Config) []model.SourceFile {
	files := make([]model.SourceFile, 0, len(jsURLs))
	for _, item := range jsURLs {
		body, err := FetchURL(item, cfg)
		if err != nil {
			files = append(files, model.SourceFile{URL: item, Error: err.Error()})
			continue
		}
		files = append(files, model.SourceFile{URL: item, Content: body})
	}

	return files
}
