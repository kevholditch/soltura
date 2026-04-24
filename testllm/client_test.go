package testllm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"soltura/llm"
)

func TestLoadFixtureRejectsMissingRequiredPurpose(t *testing.T) {
	raw := `{
  "purposes": {
    "session_seed": {"complete_text": "hola"}
  }
}`
	path := writeFixtureFile(t, raw)

	_, err := LoadFixture(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required purpose")
}

func TestLoadFixtureRejectsEmptySequenceStep(t *testing.T) {
	fixture := validFixture()
	fixture.Purposes[string(llm.PurposeDrillMark)] = PurposeScript{
		Sequence: []ScriptStep{{}},
	}

	path := writeFixtureFile(t, mustJSON(t, fixture))
	_, err := LoadFixture(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty sequence step")
}

func TestClientCompleteAndStreamByPurpose(t *testing.T) {
	client := NewClient(validFixture())

	seedCtx := llm.WithPurpose(context.Background(), llm.PurposeSessionSeed)
	seedText, err := client.Complete(seedCtx, "", nil)
	require.NoError(t, err)
	require.Equal(t, "seed", seedText)

	streamCtx := llm.WithPurpose(context.Background(), llm.PurposeConversationStream)
	var chunks []string
	streamText, err := client.StreamCompletion(streamCtx, "", nil, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	require.NoError(t, err)
	require.Equal(t, []string{"convo-", "stream"}, chunks)
	require.Equal(t, "convo-stream", streamText)
}

func TestClientFailsOnUnknownPurpose(t *testing.T) {
	client := NewClient(validFixture())
	ctx := llm.WithPurpose(context.Background(), llm.Purpose("made_up_purpose"))

	_, err := client.Complete(ctx, "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown test fixture purpose")
}

func TestClientFailsWhenSequenceExhausted(t *testing.T) {
	fixture := validFixture()
	fixture.Purposes[string(llm.PurposeDrillMark)] = PurposeScript{
		Sequence: []ScriptStep{
			{CompleteText: `{"correct": false}`},
		},
	}
	client := NewClient(fixture)
	ctx := llm.WithPurpose(context.Background(), llm.PurposeDrillMark)

	_, err := client.Complete(ctx, "", nil)
	require.NoError(t, err)

	_, err = client.Complete(ctx, "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequence exhausted")
}

func TestClientHydratesDrillStartVocabID(t *testing.T) {
	fixture := validFixture()
	fixture.Purposes[string(llm.PurposeDrillStart)] = PurposeScript{
		CompleteText: `{"pattern_name":"p","explanation":"e","question":"q","vocab_ids":["{{FIRST_VOCAB_ID}}"]}`,
	}
	client := NewClient(fixture)
	ctx := llm.WithPurpose(context.Background(), llm.PurposeDrillStart)
	messages := []llm.Message{{Role: "user", Content: `{"id":"abc-123","original":"x"}`}}

	out, err := client.Complete(ctx, "", messages)
	require.NoError(t, err)
	require.Contains(t, out, `"vocab_ids":["abc-123"]`)
}

func writeFixtureFile(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.json")
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(raw)), 0o644))
	return path
}

func mustJSON(t *testing.T, fixture *Fixture) string {
	t.Helper()
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	return string(data)
}

func validFixture() *Fixture {
	return &Fixture{
		Purposes: map[string]PurposeScript{
			string(llm.PurposeSessionSeed): {
				CompleteText: "seed",
			},
			string(llm.PurposeConversationStream): {
				StreamChunks: []string{"convo-", "stream"},
			},
			string(llm.PurposeCorrectionAnalysis): {
				CompleteText: `[{"original":"x","corrected":"y","explanation":"z","category":"grammar"}]`,
			},
			string(llm.PurposeSessionSummary): {
				CompleteText: "summary",
			},
			string(llm.PurposeDrillStart): {
				CompleteText: `{"pattern_name":"p","explanation":"e","question":"q","vocab_ids":["v1"]}`,
			},
			string(llm.PurposeDrillMark): {
				Sequence: []ScriptStep{
					{CompleteText: `{"correct": false}`},
					{CompleteText: `{"correct": true}`},
				},
			},
			string(llm.PurposeDrillFeedback): {
				Sequence: []ScriptStep{
					{StreamChunks: []string{"feedback-", "one"}},
					{StreamChunks: []string{"feedback-", "two"}},
				},
			},
			string(llm.PurposeDrillEvaluate): {
				Sequence: []ScriptStep{
					{CompleteText: `{"correct": false, "mastered": false, "next_question":"next"}`},
					{CompleteText: `{"correct": true, "mastered": true, "next_question":""}`},
				},
			},
			string(llm.PurposeDrillTransition): {
				StreamChunks: []string{"transition"},
			},
		},
	}
}
