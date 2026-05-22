package core

import (
	"net/http"
	"testing"

	"apiextractor/internal/model"
)

func TestExtractFromTextFindsCommonRequestPatterns(t *testing.T) {
	text := `
		fetch("/api/users?active=1");
		axios.post('/v1/orders', payload);
		xhr.open("GET", "/admin/config");
		$.ajax({ method: "POST", url: "/rest/report" });
		const endpoint = { path: "/account/profile" };
		const base = "api/tenant/list";
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/users?active=1",
		"/v1/orders",
		"/admin/config",
		"/rest/report",
		"/account/profile",
		"api/tenant/list",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextAddsGraphQLEndpoint(t *testing.T) {
	text := `query UserInfo { viewer { id name } }`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/graphql"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextFindsRequestConstructorAndWebSocket(t *testing.T) {
	text := `
		const req = new Request("/api/profile");
		const ws = new WebSocket("wss://example.com/socket");
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/profile",
		"wss://example.com/socket",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextHandlesTemplateLiteralConservatively(t *testing.T) {
	text := "fetch(`/api/users/${userId}`);\nconst options = { url: `/v1/orders/${orderId}` };"

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/users/",
		"/v1/orders/",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextHandlesStringConcatConservatively(t *testing.T) {
	text := `
		fetch("/api/" + userId + "/detail");
		const req = new Request("/auth/token" + suffix);
		const options = { url: "/rest/report" + query };
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/",
		"/auth/token",
		"/rest/report",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextPreservesMethodHints(t *testing.T) {
	text := `
		axios.post("/api/v1/orders", payload);
		$.ajax({ method: "DELETE", url: "/api/v1/orders/1" });
		xhr.open("PATCH", "/api/v1/profile");
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	methods := make(map[string]string, len(got))
	for _, item := range got {
		methods[item.RawValue] = item.MethodHint
	}

	if methods["/api/v1/orders"] != http.MethodPost {
		t.Fatalf("expected POST hint, got %q", methods["/api/v1/orders"])
	}
	if methods["/api/v1/orders/1"] != http.MethodDelete {
		t.Fatalf("expected DELETE hint, got %q", methods["/api/v1/orders/1"])
	}
	if methods["/api/v1/profile"] != http.MethodPatch {
		t.Fatalf("expected PATCH hint, got %q", methods["/api/v1/profile"])
	}
}

func TestExtractFromTextKeepsUsefulStaticPrefixAfterBaseToken(t *testing.T) {
	text := `
		const endpoint = "${baseURL}/api/v1/users";
		const gql = "${origin}/graphql";
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/v1/users",
		"/graphql",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextHandlesRequestObjectConcatConservatively(t *testing.T) {
	text := `const options = { url: "/api/users" + id + "?a=1" };`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/users"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextFindsNewURLFirstArg(t *testing.T) {
	text := `const full = new URL("/api/order", location.origin);`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/order"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextIgnoresFunctionExpressionURL(t *testing.T) {
	text := `const options = { url: getApiUrl() };`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func TestExtractFromTextSkipsStaticJSConcatPath(t *testing.T) {
	text := `const options = { path: "/static/app.js" + v };`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func TestExtractFromTextCleansLogicalFallbackTail(t *testing.T) {
	text := `const options = { url: "/api/a" || fallback };`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/a"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextAxiosObjectURL(t *testing.T) {
	text := `const resp = axios({ method: "GET", url: "/api/users/list" });`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/users/list"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextJQueryShortcutGetPostGetJSON(t *testing.T) {
	text := `
		$.get("/api/ping");
		$.post("/v1/order/create", payload);
		$.getJSON("/rest/meta?scope=all");
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/ping",
		"/v1/order/create",
		"/rest/meta?scope=all",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextCleanTailAndOrParenSemicolon(t *testing.T) {
	text := `
		const a = { url: "/api/a" || fallback };
		const b = { url: "/api/b" && enabled };
		const c = { url: "/api/c") };
		const d = { url: "/api/d"; };
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{
		"/api/a",
		"/api/b",
		"/api/c",
		"/api/d",
	}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextDecodeEscapedSlash(t *testing.T) {
	text := `const options = { url: "\/api\/users\u002fdetail\u002Fv1" };`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/users/detail/v1"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextFilterJavascriptDataMailto(t *testing.T) {
	text := `
		fetch("javascript:alert(1)");
		axios({ url: "data:text/plain,api" });
		const mail = { url: "mailto:security@example.com" };
	`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func TestExtractFromTextXHRLowercaseMethod(t *testing.T) {
	t.Skip("future work: explicitly decide lowercase XHR method handling and keep this as a tracked case")

	text := `xhr.open("get", "/api/lowercase-method");`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/lowercase-method"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextJQueryAjaxStringFirstArg(t *testing.T) {
	t.Skip("future work: support $.ajax(\"/api/x\", {...}) first-argument URL extraction")

	text := `$.ajax("/api/from-first-arg", { method: "GET" });`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/api/from-first-arg"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextGraphQLSubscription(t *testing.T) {
	t.Skip("future work: support GraphQL subscription operation heuristic")

	text := `subscription NewMessages { messageAdded { id } }`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	want := []string{"/graphql"}

	assertExtractedRawValues(t, got, want)
}

func TestExtractFromTextAbsoluteStaticURLFalsePositive(t *testing.T) {
	t.Skip("future work: reduce absolute static asset false positives")

	text := `const img = "https://example.com/assets/logo.png";`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func TestExtractFromTextBusinessRouteFalsePositive(t *testing.T) {
	t.Skip("future work: reduce business route false positives for non-API pages")

	text := `const route = "/admin/login";`

	got := ExtractFromText(text, "https://example.com/app.js", "res-1", "javascript")
	if len(got) != 0 {
		t.Fatalf("expected empty candidates, got %#v", got)
	}
}

func assertExtractedRawValues(t *testing.T, got []model.ExtractedCandidate, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d %#v, want %d %#v", len(got), extractedRawValues(got), len(want), want)
	}

	items := make(map[string]struct{}, len(got))
	for _, item := range got {
		items[item.RawValue] = struct{}{}
	}
	for _, item := range want {
		if _, exists := items[item]; !exists {
			t.Fatalf("missing %q in %#v", item, extractedRawValues(got))
		}
	}
}

func extractedRawValues(items []model.ExtractedCandidate) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.RawValue)
	}
	return out
}
