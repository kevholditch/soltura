package handlers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"soltura/models"
)

type rawCorrection struct {
	Original    string `json:"original"`
	Corrected   string `json:"corrected"`
	Explanation string `json:"explanation"`
	Category    string `json:"category"`
}

var correctedAlternativesRE = regexp.MustCompile(`(?m)("corrected"\s*:\s*)"([^"\n]*)"(?:\s+or\s+"[^"\n]*")+`)

func parseCorrectionsPayload(raw string) ([]models.Correction, error) {
	cleaned := normaliseCorrectionsPayload(raw)
	rawCorrections, err := unmarshalRawCorrections(cleaned)
	if err == nil {
		return toCorrections(rawCorrections), nil
	}

	repaired := correctedAlternativesRE.ReplaceAllString(cleaned, `$1"$2"`)
	if repaired != cleaned {
		rawCorrections, repairedErr := unmarshalRawCorrections(repaired)
		if repairedErr == nil {
			return toCorrections(rawCorrections), nil
		}
		return nil, fmt.Errorf("initial parse failed: %v; repaired parse failed: %w", err, repairedErr)
	}

	return nil, err
}

func normaliseCorrectionsPayload(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		if len(lines) > 0 {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		cleaned = strings.Join(lines, "\n")
		cleaned = strings.TrimSpace(cleaned)
	}

	start := strings.Index(cleaned, "[")
	end := strings.LastIndex(cleaned, "]")
	if start >= 0 && end >= start {
		cleaned = cleaned[start : end+1]
	}

	return strings.TrimSpace(cleaned)
}

func unmarshalRawCorrections(cleaned string) ([]rawCorrection, error) {
	var rawCorrections []rawCorrection
	if err := json.Unmarshal([]byte(cleaned), &rawCorrections); err != nil {
		return nil, err
	}
	return rawCorrections, nil
}

func toCorrections(rawCorrections []rawCorrection) []models.Correction {
	corrections := make([]models.Correction, 0, len(rawCorrections))
	for _, rc := range rawCorrections {
		corrections = append(corrections, models.Correction{
			Original:    rc.Original,
			Corrected:   rc.Corrected,
			Explanation: rc.Explanation,
			Category:    rc.Category,
		})
	}
	return corrections
}
