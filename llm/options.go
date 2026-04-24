package llm

import "context"

type contextKey string

const (
	maxTokensKey    contextKey = "llm_max_tokens"
	modelProfileKey contextKey = "llm_model_profile"
	purposeKey      contextKey = "llm_purpose"
)

type ModelProfile string

const (
	ModelProfileStrong ModelProfile = "strong"
	ModelProfileFast   ModelProfile = "fast"
)

type Purpose string

const (
	PurposeSessionSeed        Purpose = "session_seed"
	PurposeConversationStream Purpose = "conversation_stream"
	PurposeCorrectionAnalysis Purpose = "correction_analysis"
	PurposeSessionSummary     Purpose = "session_summary"
	PurposeDrillStart         Purpose = "drill_start"
	PurposeDrillMark          Purpose = "drill_mark"
	PurposeDrillFeedback      Purpose = "drill_feedback_stream"
	PurposeDrillEvaluate      Purpose = "drill_evaluate"
	PurposeDrillTransition    Purpose = "drill_transition"
)

// WithMaxTokens stores a max token budget in the context for a single model call.
func WithMaxTokens(ctx context.Context, maxTokens int) context.Context {
	if maxTokens <= 0 {
		return ctx
	}
	return context.WithValue(ctx, maxTokensKey, maxTokens)
}

// MaxTokensFromContext returns the configured max token budget, or fallback.
func MaxTokensFromContext(ctx context.Context, fallback int) int {
	if ctx == nil {
		return fallback
	}
	v := ctx.Value(maxTokensKey)
	n, ok := v.(int)
	if !ok || n <= 0 {
		return fallback
	}
	return n
}

// WithModelProfile stores a model profile hint for a single call.
func WithModelProfile(ctx context.Context, profile ModelProfile) context.Context {
	if profile == "" {
		return ctx
	}
	return context.WithValue(ctx, modelProfileKey, profile)
}

// ModelProfileFromContext returns the requested model profile, or fallback.
func ModelProfileFromContext(ctx context.Context, fallback ModelProfile) ModelProfile {
	if ctx == nil {
		return fallback
	}
	v := ctx.Value(modelProfileKey)
	p, ok := v.(ModelProfile)
	if !ok || p == "" {
		return fallback
	}
	return p
}

// WithPurpose stores a call purpose hint in the context for a single model call.
func WithPurpose(ctx context.Context, purpose Purpose) context.Context {
	if purpose == "" {
		return ctx
	}
	return context.WithValue(ctx, purposeKey, purpose)
}

// PurposeFromContext returns the requested call purpose, or fallback.
func PurposeFromContext(ctx context.Context, fallback Purpose) Purpose {
	if ctx == nil {
		return fallback
	}
	v := ctx.Value(purposeKey)
	p, ok := v.(Purpose)
	if !ok || p == "" {
		return fallback
	}
	return p
}
