package core

import "testing"

// 覆盖 HTML 首层发现：script/modulepreload 应保留，stylesheet 因非文本源码后缀被过滤。
func TestExtractSourceURLs(t *testing.T) {
	html := `
		<script src="/static/app.js"></script>
		<link rel="modulepreload" href="/assets/chunk-vendor.mjs">
		<link href="/style.css" rel="stylesheet">
	`

	got := ExtractSourceURLs(html, "https://example.com/index.html", true)
	want := []string{
		"https://example.com/static/app.js",
		"https://example.com/assets/chunk-vendor.mjs",
	}

	assertStringSlice(t, got, want)
}

// 覆盖脚本内递归发现：dynamic import、ES module import 与 sourceMappingURL 的相对路径归一化。
func TestExtractNestedSourceURLs(t *testing.T) {
	source := `
		import("/assets/admin.chunk.js");
		import api from "./api-client.mjs";
		//# sourceMappingURL=app.js.map
	`

	got := ExtractNestedSourceURLs(source, "https://example.com/static/app.js", true)
	want := []string{
		"https://example.com/assets/admin.chunk.js",
		"https://example.com/static/api-client.mjs",
		"https://example.com/static/app.js.map",
	}

	assertStringSlice(t, got, want)
}
