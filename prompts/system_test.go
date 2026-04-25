package prompts

import (
	"strings"
	"testing"
)

func TestDrillStartRequiresShortBlankAnswerQuestions(t *testing.T) {
	prompt := DrillStart(`[{"id":"vocab-1","original":"a el","corrected":"al","seen_count":3}]`)

	for _, want := range []string{
		"exactly one blank",
		"one or two words",
		"Do not ask the learner to rewrite or transform a full sentence",
		"The blank must replace the target token",
		"Do not use quotation marks or dialogue fragments around the blank",
		"The sentence around ___ must be understandable on its own",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected DrillStart prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}

func TestDrillEvaluateRequiresShortBlankAnswerNextQuestions(t *testing.T) {
	prompt := DrillEvaluate("a el -> al", "Use al.", "Completa: Voy ___ parque.", "a el", "[]")

	for _, want := range []string{
		"exactly one blank",
		"one or two words",
		"Do not ask the learner to rewrite or transform a full sentence",
		"The blank must replace the target token",
		"Do not use quotation marks or dialogue fragments around the blank",
		"The sentence around ___ must be understandable on its own",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected DrillEvaluate prompt to contain %q, got:\n%s", want, prompt)
		}
	}
}
