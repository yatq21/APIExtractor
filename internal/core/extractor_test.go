package core

import "testing"

func TestExtractFromTextFindsCommonRequestPatterns(t *testing.T) {
	text := `
		fetch("/api/users?active=1");
		axios.post('/v1/orders', payload);
		xhr.open("GET", "/admin/config");
		$.ajax({ method: "POST", url: "/rest/report" });
		const endpoint = { path: "/account/profile" };
		const base = "api/tenant/list";
	`

	got := ExtractFromText(text)
	want := []string{
		"/api/users?active=1",
		"/v1/orders",
		"/admin/config",
		"/rest/report",
		"/account/profile",
		"api/tenant/list",
	}

	assertStringSlice(t, got, want)
}

func TestExtractFromTextAddsGraphQLEndpoint(t *testing.T) {
	text := `query UserInfo { viewer { id name } }`

	got := ExtractFromText(text)
	want := []string{"/graphql"}

	assertStringSlice(t, got, want)
}

func TestExtractFromTextFindsRequestConstructorAndWebSocket(t *testing.T) {
	text := `
		const req = new Request("/api/profile");
		const ws = new WebSocket("wss://example.com/socket");
	`

	got := ExtractFromText(text)
	want := []string{
		"/api/profile",
		"wss://example.com/socket",
	}

	assertStringSlice(t, got, want)
}

// 模板字符串场景只保留 `${}` 之前的静态前缀，验证“保守提取”不会误拼接动态变量。
func TestExtractFromTextHandlesTemplateLiteralConservatively(t *testing.T) {
	text := "fetch(`/api/users/${userId}`);\nconst options = { url: `/v1/orders/${orderId}` };"

	got := ExtractFromText(text)
	want := []string{
		"/api/users/",
		"/v1/orders/",
	}

	assertStringSlice(t, got, want)
}

// 字符串拼接场景验证只截取左侧静态路径，不把运行期拼接表达式写入候选。
func TestExtractFromTextHandlesStringConcatConservatively(t *testing.T) {
	text := `
		fetch("/api/" + userId + "/detail");
		const req = new Request("/auth/token" + suffix);
		const options = { url: "/rest/report" + query };
	`

	got := ExtractFromText(text)
	want := []string{
		"/api/",
		"/auth/token",
		"/rest/report",
	}

	assertStringSlice(t, got, want)
}

// 对象字面量中的 url 拼接路径也应遵循同样的静态前缀规则。
func TestExtractFromTextHandlesRequestObjectConcatConservatively(t *testing.T) {
	text := `const options = { url: "/api/users" + id + "?a=1" };`

	got := ExtractFromText(text)
	want := []string{"/api/users"}

	assertStringSlice(t, got, want)
}

func TestExtractFromTextFindsNewURLFirstArg(t *testing.T) {
	text := `const full = new URL("/api/order", location.origin);`

	got := ExtractFromText(text)
	want := []string{"/api/order"}

	assertStringSlice(t, got, want)
}

func TestExtractFromTextIgnoresFunctionExpressionURL(t *testing.T) {
	text := `const options = { url: getApiUrl() };`

	got := ExtractFromText(text)
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func TestExtractFromTextSkipsStaticJSConcatPath(t *testing.T) {
	text := `const options = { path: "/static/app.js" + v };`

	got := ExtractFromText(text)
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

// 验证 `|| fallback` 这类逻辑尾巴会被清理，确保候选可直接用于请求重放。
func TestExtractFromTextCleansLogicalFallbackTail(t *testing.T) {
	text := `const options = { url: "/api/a" || fallback };`

	got := ExtractFromText(text)
	want := []string{"/api/a"}

	assertStringSlice(t, got, want)
}

func TestExtractFromTextAxiosObjectURL(t *testing.T) {
	text := `const resp = axios({ method: "GET", url: "/api/users/list" });`

	got := ExtractFromText(text)
	want := []string{"/api/users/list"}

	assertStringSlice(t, got, want)
}

func TestExtractFromTextJQueryShortcutGetPostGetJSON(t *testing.T) {
	text := `
		$.get("/api/ping");
		$.post("/v1/order/create", payload);
		$.getJSON("/rest/meta?scope=all");
	`

	got := ExtractFromText(text)
	want := []string{
		"/api/ping",
		"/v1/order/create",
		"/rest/meta?scope=all",
	}

	assertStringSlice(t, got, want)
}

// 集中覆盖 `||`、`&&`、`)`、`;` 等尾随符号清理规则，避免正则捕获后残留语法噪声。
func TestExtractFromTextCleanTailAndOrParenSemicolon(t *testing.T) {
	text := `
		const a = { url: "/api/a" || fallback };
		const b = { url: "/api/b" && enabled };
		const c = { url: "/api/c") };
		const d = { url: "/api/d"; };
	`

	got := ExtractFromText(text)
	want := []string{
		"/api/a",
		"/api/b",
		"/api/c",
		"/api/d",
	}

	assertStringSlice(t, got, want)
}

// 覆盖 `\/` 与 `\u002f` 两类 escaped slash，确保混淆写法能还原为可读路径。
func TestExtractFromTextDecodeEscapedSlash(t *testing.T) {
	text := `const options = { url: "\/api\/users\u002fdetail\u002Fv1" };`

	got := ExtractFromText(text)
	want := []string{"/api/users/detail/v1"}

	assertStringSlice(t, got, want)
}

// 协议型伪链接容易被字符串规则误抓，这里确认 javascript:/data:/mailto: 必须被过滤。
func TestExtractFromTextFilterJavascriptDataMailto(t *testing.T) {
	text := `
		fetch("javascript:alert(1)");
		axios({ url: "data:text/plain,api" });
		const mail = { url: "mailto:security@example.com" };
	`

	got := ExtractFromText(text)
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

// 该用例保留为能力边界跟踪：当前 XHR method 规则仅匹配大写方法名。
func TestExtractFromTextXHRLowercaseMethod(t *testing.T) {
	t.Skip("future work: explicitly decide lowercase XHR method handling and keep this as a tracked case")

	text := `xhr.open("get", "/api/lowercase-method");`

	got := ExtractFromText(text)
	want := []string{"/api/lowercase-method"}

	assertStringSlice(t, got, want)
}

// 该用例保留为能力边界跟踪：当前 $.ajax("url", {...}) 首参数提取尚未启用。
func TestExtractFromTextJQueryAjaxStringFirstArg(t *testing.T) {
	t.Skip("future work: support $.ajax(\"/api/x\", {...}) first-argument URL extraction")

	text := `$.ajax("/api/from-first-arg", { method: "GET" });`

	got := ExtractFromText(text)
	want := []string{"/api/from-first-arg"}

	assertStringSlice(t, got, want)
}

// 该用例保留为能力边界跟踪：当前 GraphQL 启发式仅覆盖 query/mutation，不含 subscription。
func TestExtractFromTextGraphQLSubscription(t *testing.T) {
	t.Skip("future work: support GraphQL subscription operation heuristic")

	text := `subscription NewMessages { messageAdded { id } }`

	got := ExtractFromText(text)
	want := []string{"/graphql"}

	assertStringSlice(t, got, want)
}

// 该用例跟踪绝对静态资源 URL 误报问题，提醒后续收敛策略不要破坏现有召回行为。
func TestExtractFromTextAbsoluteStaticURLFalsePositive(t *testing.T) {
	t.Skip("future work: reduce absolute static asset false positives")

	text := `const img = "https://example.com/assets/logo.png";`

	got := ExtractFromText(text)
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

// 该用例跟踪 businessPathPattern 带来的路由误报风险，明确它是“召回优先”的已知折中。
func TestExtractFromTextBusinessRouteFalsePositive(t *testing.T) {
	t.Skip("future work: reduce business route false positives for non-API pages")

	text := `const route = "/admin/login";`

	got := ExtractFromText(text)
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d %#v, want %d %#v", len(got), got, len(want), want)
	}

	items := make(map[string]struct{}, len(got))
	for _, item := range got {
		items[item] = struct{}{}
	}
	for _, item := range want {
		if _, exists := items[item]; !exists {
			t.Fatalf("missing %q in %#v", item, got)
		}
	}
}
