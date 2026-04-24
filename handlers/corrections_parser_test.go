package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCorrectionsPayload_ValidJSON(t *testing.T) {
	raw := `[
  {
    "original": "mi gusta",
    "corrected": "me gusta",
    "explanation": "Use me with gustar.",
    "category": "grammar"
  }
]`

	corrections, err := parseCorrectionsPayload(raw)
	require.NoError(t, err)
	require.Len(t, corrections, 1)
	require.Equal(t, "me gusta", corrections[0].Corrected)
}

func TestParseCorrectionsPayload_FencedAlternatives(t *testing.T) {
	raw := "```json\n[\n  {\n    \"original\": \"no he trabajado\",\n    \"corrected\": \"he estado trabajando\" or \"no trabajo\" or \"he tenido que trabajar\",\n    \"explanation\": \"Clarify intended tense.\",\n    \"category\": \"grammar\"\n  },\n  {\n    \"original\": \"problemente\",\n    \"corrected\": \"probablemente\",\n    \"explanation\": \"Misspelling.\",\n    \"category\": \"spelling\"\n  }\n]\n```"

	corrections, err := parseCorrectionsPayload(raw)
	require.NoError(t, err)
	require.Len(t, corrections, 2)
	require.Equal(t, "he estado trabajando", corrections[0].Corrected)
	require.Equal(t, "probablemente", corrections[1].Corrected)
}

func TestParseCorrectionsPayload_InvalidJSON(t *testing.T) {
	raw := `[{"original":"x","corrected":"y","explanation":"z","category":"grammar"`

	_, err := parseCorrectionsPayload(raw)
	require.Error(t, err)
}
