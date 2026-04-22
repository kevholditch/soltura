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
- NEVER correct the user's grammar or spelling in your reply — corrections are handled separately by another system

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

// DrillStart returns a prompt for the LLM to analyse vocab errors and pick a pattern to drill.
func DrillStart(vocabJSON string) string {
	tmpl := template.Must(template.New("drillstart").Parse(`You are a Spanish language drill coach.

Below is a JSON list of mistakes made by a Spanish learner. Each entry has:
original, corrected, explanation, category, seen_count.

{{.VocabJSON}}

Task:
1. Identify the single most common mistake PATTERN (group related errors).
   Prioritise by total seen_count across grouped errors.
2. Write a 2-3 sentence explanation of the pattern in Spanish (B2 level, warm tone).
3. Write one practice question in Spanish that requires applying the correct rule.
   Question types: fill-in-the-blank, translate a short phrase, or correct an error.

Return ONLY valid JSON — no markdown, no preamble:
{
  "pattern_name": "short English label for the pattern",
  "explanation": "explanation in Spanish",
  "question": "practice question in Spanish",
  "vocab_ids": ["id1", "id2"]
}

vocab_ids must contain the IDs of all errors that belong to this pattern.`))

	var buf strings.Builder
	err := tmpl.Execute(&buf, struct {
		VocabJSON string
	}{VocabJSON: vocabJSON})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// DrillEvaluate returns a prompt for structured evaluation of a drill answer.
func DrillEvaluate(patternName, explanation, question, answer, historyJSON string) string {
	tmpl := template.Must(template.New("drillevaluate").Parse(`You are evaluating a Spanish drill answer.

Pattern: {{.PatternName}}
Explanation given: {{.Explanation}}
Conversation so far: {{.HistoryJSON}}
Most recent question: {{.Question}}
Learner's answer: {{.Answer}}

Decide:
1. correct — did the learner apply the rule correctly? (true/false)
2. mastered — has the learner demonstrated clear understanding across this conversation? (true/false)
   Set mastered=true only after at least one correct answer and no persistent confusion.
3. next_question — if not mastered, provide a new question on the same pattern (different from previous ones).
   Leave empty string if mastered.

Return ONLY valid JSON:
{"correct": true, "mastered": false, "next_question": "new question here"}`))

	var buf strings.Builder
	err := tmpl.Execute(&buf, struct {
		PatternName string
		Explanation string
		Question    string
		Answer      string
		HistoryJSON string
	}{
		PatternName: patternName,
		Explanation: explanation,
		Question:    question,
		Answer:      answer,
		HistoryJSON: historyJSON,
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// DrillFeedback returns a system prompt for streaming warm feedback on a drill answer.
func DrillFeedback(patternName, question, answer string) string {
	tmpl := template.Must(template.New("drillfeedback").Parse(`You are an encouraging Spanish language drill coach giving feedback on a learner's answer.

Pattern being drilled: {{.PatternName}}
Question asked: {{.Question}}
Learner's answer: {{.Answer}}

Give 1-3 sentences of warm, specific feedback in Spanish:
- If correct: confirm what they did right, reinforce the rule briefly.
- If wrong: gently point out the error and restate the rule.
Do not ask a new question. Do not use English.`))

	var buf strings.Builder
	err := tmpl.Execute(&buf, struct {
		PatternName string
		Question    string
		Answer      string
	}{
		PatternName: patternName,
		Question:    question,
		Answer:      answer,
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
