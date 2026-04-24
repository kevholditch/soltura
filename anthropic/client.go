package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"soltura/llm"
)

type request struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system"`
	Messages  []llm.Message `json:"messages"`
	Stream    bool          `json:"stream"`
}

type streamEvent struct {
	Type  string `json:"type"`
	Delta *delta `json:"delta,omitempty"`
}

type delta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Client struct {
	apiKey      string
	baseURL     string
	strongModel string
	fastModel   string
	http        *http.Client
}

const (
	DefaultStrongModel = "claude-sonnet-4-6"
	DefaultFastModel   = "claude-haiku-4-5"
)

// ResolveModels applies Anthropic model defaults for strong/fast lanes.
func ResolveModels(strongModel, fastModel string) (string, string) {
	if strongModel == "" {
		strongModel = DefaultStrongModel
	}
	if fastModel == "" {
		fastModel = DefaultFastModel
	}
	return strongModel, fastModel
}

func NewClient(apiKey, strongModel, fastModel string) *Client {
	strongModel, fastModel = ResolveModels(strongModel, fastModel)

	return &Client{
		apiKey:      apiKey,
		baseURL:     "https://api.anthropic.com/v1/messages",
		strongModel: strongModel,
		fastModel:   fastModel,
		http:        &http.Client{Timeout: 0}, // no timeout on client, use ctx
	}
}

func (c *Client) modelFor(ctx context.Context) string {
	profile := llm.ModelProfileFromContext(ctx, llm.ModelProfileStrong)
	if profile == llm.ModelProfileFast && c.fastModel != "" {
		return c.fastModel
	}
	return c.strongModel
}

func (c *Client) shouldRetryWithStrong(requestedModel string) bool {
	return requestedModel != "" && requestedModel == c.fastModel && c.strongModel != "" && c.strongModel != requestedModel
}

// post marshals req, sets auth headers, executes the request, and returns the
// response on 200. On non-200 the body is read, closed, and returned as an error.
func (c *Client) post(ctx context.Context, req request) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("request failed with status %d (could not read body: %v)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

func (c *Client) StreamCompletion(ctx context.Context, system string, messages []llm.Message, onChunk func(string)) (string, error) {
	maxTokens := llm.MaxTokensFromContext(ctx, 4096)
	requestedModel := c.modelFor(ctx)
	req := request{
		Model:     requestedModel,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Stream:    true,
	}

	resp, err := c.post(ctx, req)
	if err != nil && c.shouldRetryWithStrong(requestedModel) {
		log.Printf("anthropic fast model request failed (%s), retrying with strong model (%s): %v", requestedModel, c.strongModel, err)
		req.Model = c.strongModel
		resp, err = c.post(ctx, req)
	}
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	var accumulated strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.Printf("failed to unmarshal stream event: %v", err)
			continue
		}

		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Type == "text_delta" {
			onChunk(event.Delta.Text)
			accumulated.WriteString(event.Delta.Text)
		}
	}

	return accumulated.String(), nil
}

func (c *Client) Complete(ctx context.Context, system string, messages []llm.Message) (string, error) {
	maxTokens := llm.MaxTokensFromContext(ctx, 1000)
	requestedModel := c.modelFor(ctx)
	req := request{
		Model:     requestedModel,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Stream:    false,
	}

	resp, err := c.post(ctx, req)
	if err != nil && c.shouldRetryWithStrong(requestedModel) {
		log.Printf("anthropic fast model request failed (%s), retrying with strong model (%s): %v", requestedModel, c.strongModel, err)
		req.Model = c.strongModel
		resp, err = c.post(ctx, req)
	}
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty content in response")
	}

	return result.Content[0].Text, nil
}
