package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithPurposeAndPurposeFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithPurpose(ctx, PurposeDrillStart)

	purpose := PurposeFromContext(ctx, "")
	require.Equal(t, PurposeDrillStart, purpose)
}

func TestPurposeFromContextFallback(t *testing.T) {
	purpose := PurposeFromContext(context.Background(), PurposeSessionSeed)
	require.Equal(t, PurposeSessionSeed, purpose)
}

func TestWithPurposeEmptyNoop(t *testing.T) {
	ctx := context.Background()
	ctx = WithPurpose(ctx, "")

	purpose := PurposeFromContext(ctx, PurposeSessionSummary)
	require.Equal(t, PurposeSessionSummary, purpose)
}
