package logger

import (
	"fmt"
	"strings"
)

var currentLevel = "info"

// SetLevel configures the current log verbosity.
func SetLevel(level string) {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "silent", "info", "warn", "debug":
		currentLevel = level
	default:
		currentLevel = "info"
	}
}

// Debug prints a debug-level message.
func Debug(message string) {
	if shouldLog("debug") {
		fmt.Printf("[D] %s\n", message)
	}
}

// Info prints an info-level message.
func Info(message string) {
	if shouldLog("info") {
		fmt.Printf("[+] %s\n", message)
	}
}

// Warning prints a warning-level message.
func Warning(message string) {
	if shouldLog("warn") {
		fmt.Printf("[!] %s\n", message)
	}
}

// Error prints an error-level message.
func Error(message string) {
	if currentLevel != "silent" {
		fmt.Printf("[-] %s\n", message)
	}
}

func shouldLog(level string) bool {
	order := map[string]int{
		"silent": 0,
		"warn":   1,
		"info":   2,
		"debug":  3,
	}
	return order[currentLevel] >= order[level]
}
