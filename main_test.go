package main

import (
	"testing"

	"apiextractor/internal/config"
)

func TestApplyHeaders(t *testing.T) {
	cfg := config.Default()
	err := applyHeaders(&cfg, []string{"X-Test: demo", "Authorization: Bearer token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DefaultHeaders["X-Test"] != "demo" {
		t.Fatal("expected X-Test header to be applied")
	}
	if cfg.DefaultHeaders["Authorization"] != "Bearer token" {
		t.Fatal("expected Authorization header to be applied")
	}
}

func TestApplyHeadersRejectsInvalidFormat(t *testing.T) {
	cfg := config.Default()
	if err := applyHeaders(&cfg, []string{"broken-header"}); err == nil {
		t.Fatal("expected invalid header format error")
	}
}

func TestIsValidLogLevel(t *testing.T) {
	valid := []string{"silent", "info", "warn", "debug"}
	for _, level := range valid {
		if !isValidLogLevel(level) {
			t.Fatalf("expected valid log level: %s", level)
		}
	}
	if isValidLogLevel("verbose") {
		t.Fatal("expected verbose to be rejected")
	}
}
