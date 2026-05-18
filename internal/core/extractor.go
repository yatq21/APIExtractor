package core

import (
	"regexp"
	"strings"

	"apiextractor/internal/model"
)

var apiPattern = regexp.MustCompile("(?i)(?:[\"'`])((?:https?:)?//[^\"'`\\s]+|/[A-Za-z0-9._~!$&'()*+,;=:@%/-]*(?:api|graphql|rest)[A-Za-z0-9._~!$&'()*+,;=:@%/-]*)(?:[\"'`])")

// ExtractFromText 从 HTML 或 JavaScript 文本中提取疑似 API 路径或 URL 字面量。
func ExtractFromText(text string) []string {
	matches := apiPattern.FindAllStringSubmatch(text, -1)
	results := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		results = append(results, strings.TrimSpace(match[1]))
	}

	return results
}

// ExtractAll 汇总页面 HTML 和已下载 JavaScript 文件中的 API 候选。
func ExtractAll(html string, jsFiles []model.SourceFile) []string {
	seen := make(map[string]struct{})
	all := make([]string, 0)

	merge := func(items []string) {
		for _, item := range items {
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			all = append(all, item)
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
