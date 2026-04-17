package prompts

import (
	"strings"
	"text/template"
)

// ConversationSystem returns a system prompt for a Spanish conversation partner.
func ConversationSystem(topic string) string {
	tmpl := template.Must(template.New("conversation").Parse(`You are a Spanish conversation partner for an advanced English speaker learning Spanish.
The user has strong comprehension (C1 level) but weaker productive/output skills.

Your role:
- Always respond ONLY in Spanish
- Keep responses conversational, natural, and engaging
- Match the user's topic and energy
- Pitch your language at high B2/C1 — rich vocabulary, varied grammar, but not academic
- Ask a follow-up question to keep the conversation flowing
- If the user writes in English, gently respond in Spanish and invite them to try in Spanish

You are currently discussing: {{.Topic}}

The conversation so far is provided in the message history.`))

	var buf strings.Builder
	err := tmpl.Execute(&buf, struct {
		Topic string
	}{
		Topic: topic,
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// CorrectionAnalysis returns a prompt for analyzing and correcting Spanish text.
func CorrectionAnalysis(userText string) string {
	tmpl := template.Must(template.New("correction").Parse(`You are a Spanish language correction engine. Analyse the following Spanish text written by an advanced learner and identify errors.

Text to analyse:
{{.UserText}}

Return a JSON array of corrections. Each correction object must have:
- "original": the incorrect word or phrase as written
- "corrected": the correct form
- "explanation": a brief explanation in English (1 sentence max)
- "category": one of: grammar | vocabulary | gender | spelling | register

Return ONLY the JSON array. No preamble, no markdown. If there are no errors, return an empty array [].

Example:
[
  {
    "original": "soy muy bien",
    "corrected": "estoy muy bien",
    "explanation": "Use 'estar' not 'ser' for temporary states like feeling well.",
    "category": "grammar"
  }
]`))

	var buf strings.Builder
	err := tmpl.Execute(&buf, struct {
		UserText string
	}{
		UserText: userText,
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// SessionSummary returns a prompt for summarizing a learning session.
func SessionSummary(topic, duration string, turnCount int, correctionsJSON string) string {
	tmpl := template.Must(template.New("summary").Parse(`You are summarising a Spanish learning session. Here is the data:

Topic: {{.Topic}}
Duration: {{.Duration}}
Number of turns: {{.TurnCount}}

Corrections made:
{{.CorrectionsJSON}}

Write a concise session summary in English with these sections:
1. What went well (1-2 sentences, genuine and specific)
2. Key corrections (group by category, max 5 most important)
3. Words to review (list the corrected forms to remember)
4. One thing to focus on next session

Tone: encouraging but honest. This person is smart and doesn't want empty praise.`))

	var buf strings.Builder
	err := tmpl.Execute(&buf, struct {
		Topic           string
		Duration        string
		TurnCount       int
		CorrectionsJSON string
	}{
		Topic:           topic,
		Duration:        duration,
		TurnCount:       turnCount,
		CorrectionsJSON: correctionsJSON,
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}
