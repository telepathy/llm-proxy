package backend

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenRouterBackend struct {
	name         string
	baseURL      string
	apiKey       string
	extraHeaders map[string]string
	client       *http.Client
}

func NewOpenRouterBackend(name, baseURL, apiKey string, extraHeaders map[string]string) *OpenRouterBackend {
	return &OpenRouterBackend{
		name:         name,
		baseURL:      baseURL,
		apiKey:       apiKey,
		extraHeaders: extraHeaders,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (b *OpenRouterBackend) Name() string {
	return b.name
}

func (b *OpenRouterBackend) ChatCompletion(ctx context.Context, req *Request) (*Response, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	b.setHeaders(httpReq)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (b *OpenRouterBackend) ChatCompletionStream(ctx context.Context, req *Request, writer io.Writer) error {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	b.setHeaders(httpReq)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	flusher, canFlush := writer.(http.Flusher)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if _, err := fmt.Fprintf(writer, "%s\n", line); err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		if canFlush {
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	return nil
}

func (b *OpenRouterBackend) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)

	for key, value := range b.extraHeaders {
		req.Header.Set(key, value)
	}
}
