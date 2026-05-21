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

func TestExtractFromTextHandlesTemplateLiteralConservatively(t *testing.T) {
	text := "fetch(`/api/users/${userId}`);\nconst options = { url: `/v1/orders/${orderId}` };"

	got := ExtractFromText(text)
	want := []string{
		"/api/users/",
		"/v1/orders/",
	}

	assertStringSlice(t, got, want)
}

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

func TestExtractFromTextCleansLogicalFallbackTail(t *testing.T) {
	text := `const options = { url: "/api/a" || fallback };`

	got := ExtractFromText(text)
	want := []string{"/api/a"}

	assertStringSlice(t, got, want)
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
