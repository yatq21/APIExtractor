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
	"apiextractor/internal/urlutil"
)

var (
	// scriptSrcPattern 覆盖传统 <script src="..."> 入口。
	scriptSrcPattern = regexp.MustCompile(`(?i)<script\b[^>]*\bsrc\s*=\s*["']([^"']+)["']`)
	// linkHrefPattern 同时扫描 <link href="...">，用于捕获 modulepreload 等预加载脚本入口。
	// 非脚本 link（如 css）会在 normalizeSourceURL/isTextSource 阶段被后置过滤。
	linkHrefPattern = regexp.MustCompile(`(?i)<link\b[^>]*\bhref\s*=\s*["']([^"']+)["']`)
	// sourceMapPattern 支持 //# sourceMappingURL= 与 //@ sourceMappingURL= 两种注释写法。
	sourceMapPattern = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL\s*=\s*([^\s]+)`)
	// dynamicImportPattern 捕获 import("...")、import ... from "..."、import "..." 三类 ES module 线索。
	dynamicImportPattern = regexp.MustCompile(`(?i)(?:import\s*\(\s*|from\s+|import\s+)["']([^"']+\.(?:m?js|map|json)(?:\?[^"']*)?)["']`)
	// chunkPattern 作为构建产物兜底，覆盖 chunk/vendor/runtime/main/app 等常见命名。
	// 该启发式偏召回，最终是否可下载由 normalizeSourceURL 与后缀判断共同约束。
	chunkPattern = regexp.MustCompile(`["']([^"']*(?:chunk|bundle|vendor|runtime|app|main)[^"']*\.(?:m?js|map|json)(?:\?[^"']*)?)["']`)
)

// FetchURL 请求指定 URL，并返回限制大小后的文本响应体。
// 响应体读取上限为 2MiB，用于控制异常大资源对提取链路的影响。
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

// ExtractJSURLs 保留旧入口，内部升级为更宽的脚本/source map 来源发现。
func ExtractJSURLs(html string, baseURL string, sameOrigin bool) []string {
	return ExtractSourceURLs(html, baseURL, sameOrigin)
}

// ExtractSourceURLs 从 HTML 中收集可能包含接口线索的脚本、预加载脚本和 source map。
// 提取阶段不直接限定标签类型，只要 URL 解析后属于文本源码后缀（js/mjs/map/json）即保留。
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

// FetchJSFiles 保留旧入口，内部升级为递归抓取 JS、动态 chunk 和 source map。
func FetchJSFiles(jsURLs []string, cfg config.Config) []model.SourceFile {
	return FetchSourceFiles(jsURLs, cfg)
}

// FetchSourceFiles 下载初始脚本后继续发现动态 chunk、ES module import 和 source map。
// 采用按队列展开的递归发现，并用 MaxSourceFiles 控制最大抓取数量，避免站点资源图过大导致失控。
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
			file.Error = err.Error()
			files = append(files, file)
			continue
		}

		file.Content = body
		files = append(files, file)

		// 从已下载文本继续挖掘下一层资源，形成“HTML -> JS -> chunk/map/json”的递进发现链路。
		nextURLs := ExtractNestedSourceURLs(body, item, cfg.SameOrigin)
		for _, next := range nextURLs {
			// queued/downloaded 双重去重：既避免重复排队，也避免同一资源多次下载。
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

// ExtractNestedSourceURLs 从脚本文本里继续提取 source map、动态 import 和常见构建产物 chunk。
// 这里的规则覆盖“脚本内部引用脚本”的主要通道，不尝试执行 JavaScript，仅做静态字符串发现。
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
	// data: 不是可独立抓取的远程源码资源，直接过滤。
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return "", false
	}

	// 统一做相对路径解析、协议补全与同源策略约束。
	normalized, ok := urlutil.NormalizeCandidate(raw, baseURL, sameOrigin)
	if !ok {
		return "", false
	}
	// 模块二只递归可解析文本源，静态二进制资源不进入下载队列。
	if !isTextSource(normalized) {
		return "", false
	}

	return normalized, true
}

func isTextSource(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// 限制为可直接文本解析的前端源码类型，避免把图片/字体等静态资源当成提取输入。
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

	// SourceType 用于后续输出分层展示，不参与提取逻辑分支。
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
