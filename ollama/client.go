package ollama

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
	Messages  []llm.Message `json:"messages"`
	Stream    bool          `json:"stream"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

// streamChunk is the SSE payload shape from the OpenAI-compatible endpoint.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// completeResponse is the non-streaming response shape.
type completeResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/") + "/v1/chat/completions",
		model:   model,
		http:    &http.Client{Timeout: 0}, // no timeout on client, use ctx
	}
}

// buildMessages prepends a system message when system is non-empty, then appends
// the conversation messages. Ollama uses role "system" rather than a top-level field.
func buildMessages(system string, messages []llm.Message) []llm.Message {
	if system == "" {
		return messages
	}
	out := make([]llm.Message, 0, len(messages)+1)
	out = append(out, llm.Message{Role: "system", Content: system})
	out = append(out, messages...)
	return out
}

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
	resp, err := c.post(ctx, request{
		Model:     c.model,
		Messages:  buildMessages(system, messages),
		Stream:    true,
		MaxTokens: 4096,
	})
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
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("failed to unmarshal stream chunk: %v", err)
			continue
		}

		if len(chunk.Choices) > 0 {
			text := chunk.Choices[0].Delta.Content
			if text != "" {
				onChunk(text)
				accumulated.WriteString(text)
			}
		}
	}

	return accumulated.String(), nil
}

func (c *Client) Complete(ctx context.Context, system string, messages []llm.Message) (string, error) {
	resp, err := c.post(ctx, request{
		Model:     c.model,
		Messages:  buildMessages(system, messages),
		Stream:    false,
		MaxTokens: 1000,
	})
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	var result completeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty choices in response")
	}

	return result.Choices[0].Message.Content, nil
}
