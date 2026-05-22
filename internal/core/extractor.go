package core

import (
	"regexp"
	"strings"

	"apiextractor/internal/model"
)

var (
	// quotedURLPattern 兜底抓取引号中的绝对/相对 URL。
	// 该规则覆盖 fetch/axios 之外的散落字符串，但后续必须再经过 looksLikeAPI 过滤，避免直接放大误报。
	quotedURLPattern = regexp.MustCompile("(?i)[\"'`]((?:https?:)?//[^\"'`\\s<>]+|/[A-Za-z0-9._~!$&'()*+,;=:@%/?#\\[\\]-]+)[\"'`]")
	// apiKeywordPattern 补充抓取包含 api/graphql/rest/vN 关键词的字符串，优先提升 API 线索召回率。
	apiKeywordPattern = regexp.MustCompile("(?i)[\"'`]((?:\\.?\\.?/)?[^\"'`\\s<>]*(?:api|graphql|rest|v[0-9]+)[^\"'`\\s<>]*)[\"'`]")
	// 下列模式对应常见请求调用：fetch、Request、WebSocket、XHR、axios、jQuery。
	// 模式尽量只提取第一个静态参数，动态表达式交由 cleanCandidate/extractStaticPrefix 做保守截断。
	fetchPattern              = regexp.MustCompile("(?is)\\bfetch\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	requestConstructorPattern = regexp.MustCompile("(?is)\\bnew\\s+Request\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	webSocketPattern          = regexp.MustCompile("(?is)\\bnew\\s+WebSocket\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	xhrOpenPattern            = regexp.MustCompile("(?is)\\.open\\s*\\(\\s*[\"'`][A-Z]+[\"'`]\\s*,\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosPattern              = regexp.MustCompile("(?is)\\baxios(?:\\.[a-z]+)?\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	axiosObjectURLPattern     = regexp.MustCompile("(?is)\\baxios\\s*\\(\\s*\\{[^{}]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	newURLPattern             = regexp.MustCompile("(?is)\\bnew\\s+URL\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]\\s*,")
	jqueryAjaxURLPattern      = regexp.MustCompile("(?is)\\$\\.(?:ajax|get|post|getJSON)\\s*\\([^)]*?\\burl\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	jqueryShortcutPattern     = regexp.MustCompile("(?is)\\$\\.(?:get|post|getJSON)\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
	// requestObjectURLPattern 处理配置对象中静态 url/path 字段。
	// requestObjectExprPattern 允许先抓取对象里的表达式片段，再在后续清洗阶段保守裁剪动态部分。
	requestObjectURLPattern  = regexp.MustCompile("(?is)\\b(?:url|path|endpoint|uri|baseURL|baseUrl)\\s*:\\s*[\"'`]([^\"'`]+)[\"'`]")
	requestObjectExprPattern = regexp.MustCompile("(?is)\\b(?:url|path|endpoint|uri|baseURL|baseUrl)\\s*:\\s*([\"'`][^,\\n}]*|/(?:[^,\\n}]*)|https?://[^,\\n}]+|wss?://[^,\\n}]+)")
	// graphQLOperationPattern 仅对 query/mutation 做弱推断并回填 /graphql。
	// 该启发式不覆盖 subscription 等变体，避免“看见大括号就当 GraphQL”。
	graphQLOperationPattern = regexp.MustCompile("(?is)\\b(?:query|mutation)\\s+[A-Za-z0-9_]*\\s*(?:\\([^)]*\\))?\\s*\\{")
	// businessPathPattern 允许保留常见业务路由前缀，避免漏掉未显式带 api 关键字的内部接口路径。
	// 这是一条偏召回的规则，可能引入少量页面路由误报，由测试用例显式跟踪。
	businessPathPattern = regexp.MustCompile(`(?i)^/(?:v[0-9]+|admin|auth|user|users|account|accounts|order|orders|pay|payment|member|members|tenant|tenants|system|manage|backend|console)(?:/|$)`)
)

// ExtractFromText 从 HTML、JavaScript、source map 或 JSON 文本中提取疑似 API 路径或 URL。
// 它采用“多规则召回 + 统一清洗过滤”的流程：先匹配候选，再通过 cleanCandidate/looksLikeAPI 收敛误报。
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
	mergeMatches(requestConstructorPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(webSocketPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(xhrOpenPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(axiosPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(axiosObjectURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(newURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(jqueryAjaxURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(jqueryShortcutPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(requestObjectURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(requestObjectExprPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(quotedURLPattern.FindAllStringSubmatch(text, -1))
	mergeMatches(apiKeywordPattern.FindAllStringSubmatch(text, -1))

	// 当脚本里出现 GraphQL query/mutation 操作体但没有显式 endpoint 时，补一个保守默认值 /graphql。
	if graphQLOperationPattern.MatchString(text) {
		addCandidate("/graphql", seen, &results)
	}

	return results
}

// ExtractAll 汇总页面 HTML 和已下载源文件中的 API 候选。
// 仅处理下载成功的文本源文件，避免把网络错误当成可解析内容。
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
	// 先做形态过滤再去重，保证 seen 中只保留可用候选。
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
	// 先提取静态前缀，避免把模板变量或拼接表达式当成完整 URL。
	candidate = extractStaticPrefix(candidate)
	candidate = strings.Trim(candidate, "\"'`")
	// 兼容前端常见的 escaped slash 写法，统一为标准路径分隔符。
	candidate = strings.ReplaceAll(candidate, `\/`, `/`)
	candidate = strings.ReplaceAll(candidate, `\u002f`, `/`)
	candidate = strings.ReplaceAll(candidate, `\u002F`, `/`)
	// 去掉收尾符号，覆盖 `"/api/a" || x`、`"/api/a");`、`"/api/a";` 等场景。
	candidate = strings.TrimRight(candidate, `,;.)]}`)
	if strings.HasPrefix(candidate, "./") {
		candidate = strings.TrimPrefix(candidate, ".")
	}
	return candidate
}

func extractStaticPrefix(raw string) string {
	candidate := strings.TrimSpace(raw)

	// 模板字符串 `${...}` 后半段通常为动态值，保留前缀即可。
	if idx := strings.Index(candidate, "${"); idx >= 0 {
		candidate = candidate[:idx]
	}
	// 字符串拼接场景统一取左侧静态片段，避免把表达式噪声带入候选。
	if idx := strings.Index(candidate, "+"); idx >= 0 {
		candidate = candidate[:idx]
	}
	// 对引号包裹文本，清理逻辑表达式尾部，避免 `|| fallback`/`&& cond` 污染候选。
	if strings.ContainsAny(candidate, "\"'`") {
		if idx := strings.Index(candidate, "||"); idx >= 0 {
			candidate = candidate[:idx]
		}
		if idx := strings.Index(candidate, "&&"); idx >= 0 {
			candidate = candidate[:idx]
		}
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
	// 明确排除协议型伪链接，防止 javascript:/data:/mailto: 进入结果。
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "mailto:") {
		return false
	}
	// 仍含模板变量或拼接符号说明静态化失败，直接丢弃以保证结果可用性。
	if strings.Contains(candidate, "${") || strings.Contains(candidate, "+") {
		return false
	}
	if strings.HasPrefix(lower, "ws://") || strings.HasPrefix(lower, "wss://") {
		return true
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
	// 相对根路径若落在静态后缀，优先判定为资源文件而非接口。
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
	// 静态资源后缀用于控制前端资源误报，不参与“是否可下载”的判断。
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
