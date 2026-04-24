package testllm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"soltura/llm"
)

const DefaultFixturePath = "testdata/llm/default.json"

var vocabIDRE = regexp.MustCompile(`"id"\s*:\s*"([^"]+)"`)

type ScriptStep struct {
	CompleteText string   `json:"complete_text,omitempty"`
	StreamChunks []string `json:"stream_chunks,omitempty"`
}

type PurposeScript struct {
	CompleteText string       `json:"complete_text,omitempty"`
	StreamChunks []string     `json:"stream_chunks,omitempty"`
	Sequence     []ScriptStep `json:"sequence,omitempty"`
}

type Fixture struct {
	Purposes map[string]PurposeScript `json:"purposes"`
}

type Client struct {
	fixture *Fixture
	mu      sync.Mutex
	cursor  map[string]int
}

func NewClientFromFile(path string) (*Client, error) {
	fixture, err := LoadFixture(path)
	if err != nil {
		return nil, err
	}
	return NewClient(fixture), nil
}

func NewClientFromEnv() (*Client, error) {
	path := os.Getenv("TEST_FIXTURE_PATH")
	if strings.TrimSpace(path) == "" {
		path = DefaultFixturePath
	}
	return NewClientFromFile(path)
}

func NewClient(fixture *Fixture) *Client {
	return &Client{
		fixture: fixture,
		cursor:  map[string]int{},
	}
}

func LoadFixture(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read test fixture %q: %w", path, err)
	}

	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse test fixture %q: %w", path, err)
	}

	if err := validateFixture(&fixture); err != nil {
		return nil, err
	}

	return &fixture, nil
}

func validateFixture(fixture *Fixture) error {
	if fixture == nil {
		return fmt.Errorf("fixture is nil")
	}
	if len(fixture.Purposes) == 0 {
		return fmt.Errorf("fixture.purposes must not be empty")
	}

	required := []llm.Purpose{
		llm.PurposeSessionSeed,
		llm.PurposeConversationStream,
		llm.PurposeCorrectionAnalysis,
		llm.PurposeSessionSummary,
		llm.PurposeDrillStart,
		llm.PurposeDrillMark,
		llm.PurposeDrillFeedback,
		llm.PurposeDrillEvaluate,
		llm.PurposeDrillTransition,
	}

	for _, purpose := range required {
		script, ok := fixture.Purposes[string(purpose)]
		if !ok {
			return fmt.Errorf("fixture missing required purpose %q", purpose)
		}
		if err := validatePurposeScript(purpose, script); err != nil {
			return err
		}
	}

	return nil
}

func validatePurposeScript(purpose llm.Purpose, script PurposeScript) error {
	if len(script.Sequence) > 0 {
		for idx, step := range script.Sequence {
			if !hasScriptContent(step.CompleteText, step.StreamChunks) {
				return fmt.Errorf("purpose %q has empty sequence step at index %d", purpose, idx)
			}
		}
		return nil
	}

	if !hasScriptContent(script.CompleteText, script.StreamChunks) {
		return fmt.Errorf("purpose %q must define complete_text, stream_chunks, or sequence", purpose)
	}

	return nil
}

func hasScriptContent(completeText string, streamChunks []string) bool {
	return strings.TrimSpace(completeText) != "" || len(streamChunks) > 0
}

func (c *Client) StreamCompletion(ctx context.Context, _ string, _ []llm.Message, onChunk func(string)) (string, error) {
	step, purpose, err := c.nextStep(ctx)
	if err != nil {
		return "", err
	}

	chunks := step.StreamChunks
	if len(chunks) == 0 {
		if strings.TrimSpace(step.CompleteText) == "" {
			return "", fmt.Errorf("purpose %q has no stream_chunks or complete_text for StreamCompletion", purpose)
		}
		chunks = []string{step.CompleteText}
	}

	var b strings.Builder
	for _, chunk := range chunks {
		onChunk(chunk)
		b.WriteString(chunk)
	}

	return b.String(), nil
}

func (c *Client) Complete(ctx context.Context, _ string, messages []llm.Message) (string, error) {
	step, purpose, err := c.nextStep(ctx)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(step.CompleteText) != "" {
		return c.hydrateCompleteText(purpose, step.CompleteText, messages)
	}
	if len(step.StreamChunks) > 0 {
		return strings.Join(step.StreamChunks, ""), nil
	}

	return "", fmt.Errorf("purpose %q has no complete_text or stream_chunks for Complete", purpose)
}

func (c *Client) hydrateCompleteText(purpose llm.Purpose, text string, messages []llm.Message) (string, error) {
	if purpose != llm.PurposeDrillStart {
		return text, nil
	}

	const placeholder = "{{FIRST_VOCAB_ID}}"
	if !strings.Contains(text, placeholder) {
		return text, nil
	}

	firstID := firstVocabIDFromMessages(messages)
	if firstID == "" {
		return "", fmt.Errorf("purpose %q requested %s but no vocab id was found in prompt", purpose, placeholder)
	}

	return strings.ReplaceAll(text, placeholder, firstID), nil
}

func firstVocabIDFromMessages(messages []llm.Message) string {
	for _, msg := range messages {
		matches := vocabIDRE.FindStringSubmatch(msg.Content)
		if len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func (c *Client) nextStep(ctx context.Context) (ScriptStep, llm.Purpose, error) {
	purpose := llm.PurposeFromContext(ctx, "")
	if purpose == "" {
		return ScriptStep{}, purpose, fmt.Errorf("test backend requires llm purpose in context")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	script, ok := c.fixture.Purposes[string(purpose)]
	if !ok {
		return ScriptStep{}, purpose, fmt.Errorf("unknown test fixture purpose %q", purpose)
	}

	if len(script.Sequence) > 0 {
		idx := c.cursor[string(purpose)]
		if idx >= len(script.Sequence) {
			return ScriptStep{}, purpose, fmt.Errorf("fixture sequence exhausted for purpose %q at index %d", purpose, idx)
		}
		c.cursor[string(purpose)] = idx + 1
		return script.Sequence[idx], purpose, nil
	}

	return ScriptStep{
		CompleteText: script.CompleteText,
		StreamChunks: script.StreamChunks,
	}, purpose, nil
}
