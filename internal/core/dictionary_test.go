package core

import (
	"os"
	"path/filepath"
	"testing"

	"apiextractor/internal/config"
)

func TestCleanDictionary(t *testing.T) {
	got := CleanDictionary([]string{
		" api ",
		"/api",
		"admin # comment",
		"# skipped",
		"",
		"https://cdn.example.com/app.js",
	})
	want := []string{
		"/api",
		"/admin",
		"https://cdn.example.com/app.js",
	}

	assertStringSlice(t, got, want)
}

func TestLoadDictionaryReadsLocalFiles(t *testing.T) {
	tempDir := t.TempDir()
	dictPath := filepath.Join(tempDir, "dict.txt")
	content := []byte("api\n/admin # comment\n\napi\n")
	if err := os.WriteFile(dictPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.UseBuiltinDictionary = false
	cfg.DictionaryPaths = []string{dictPath}

	got, errs := LoadDictionary(cfg)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %#v", errs)
	}

	want := []string{"/api", "/admin"}
	assertStringSlice(t, got, want)
}
