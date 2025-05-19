package utils

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// FetchURLContent 获取指定URL的内容
func FetchURLContent(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// FetchURLContentWithRetry fetches content from a URL with retry logic.
// It attempts to fetch the content up to maxRetries times with a delay between attempts.
func FetchURLContentWithRetry(url string, maxRetries int, delay time.Duration) ([]byte, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", i+1, err)
			LogWarning(fmt.Sprintf("Failed to fetch URL %s (attempt %d/%d): %v", url, i+1, maxRetries, err), nil)
			time.Sleep(delay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("attempt %d failed with status code %d", i+1, resp.StatusCode)
			LogWarning(fmt.Sprintf("Failed to fetch URL %s (attempt %d/%d) with status code: %d", url, i+1, maxRetries, resp.StatusCode), nil)
			time.Sleep(delay)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to read body: %w", i+1, err)
			LogError(fmt.Sprintf("Failed to read response body for URL %s (attempt %d/%d): %v", url, i+1, maxRetries, err), err)
			time.Sleep(delay)
			continue
		}

		LogInfo(fmt.Sprintf("Successfully fetched URL %s after %d attempts", url, i+1))
		return body, nil
	}

	return nil, fmt.Errorf("failed to fetch URL %s after %d attempts: %w", url, maxRetries, lastErr)
}
