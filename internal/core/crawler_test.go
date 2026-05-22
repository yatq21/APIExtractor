package core

import "testing"

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
