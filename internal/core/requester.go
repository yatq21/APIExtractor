package core

import (
	"io"
	"net/http"
	"strings"
	"time"

	"apiextractor/internal/config"
	"apiextractor/internal/model"
)

// RequestAPI 对单个候选接口发送 GET 请求，并记录响应元信息。
func RequestAPI(targetURL string, cfg config.Config) model.RequestResult {
	start := time.Now()
	item := model.RequestResult{
		URL:    targetURL,
		Method: http.MethodGet,
	}

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		item.Error = err.Error()
		return item
	}

	for key, value := range cfg.DefaultHeaders {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		item.Error = err.Error()
		return item
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		item.Error = err.Error()
		return item
	}

	item.StatusCode = resp.StatusCode
	item.ContentType = resp.Header.Get("Content-Type")
	item.DurationMS = time.Since(start).Milliseconds()
	item.ResponseSize = len(body)
	item.BodyPreview = previewBody(string(body))

	return item
}

// RequestAll 顺序请求所有已规范化的候选接口。
func RequestAll(urls []string, cfg config.Config) []model.RequestResult {
	results := make([]model.RequestResult, 0, len(urls))
	for _, item := range urls {
		results = append(results, RequestAPI(item, cfg))
	}
	return results
}

// previewBody 返回去除首尾空白并限制长度后的响应体预览。
func previewBody(body string) string {
	compact := strings.TrimSpace(body)
	if len(compact) <= 200 {
		return compact
	}
	return compact[:200]
}
