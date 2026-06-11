package prompt

import (
	"encoding/json"
	"fmt"
	"strings"
)

const maxPromptContentRunes = 30000

// SummaryResponseJSON is the expected JSON payload returned by Gemini.
type SummaryResponseJSON struct {
	Headline      string   `json:"headline"`
	SummaryShort  string   `json:"summary_short"`
	SummaryMedium string   `json:"summary_medium"`
	KeyPoints     []string `json:"key_points"`
	Confidence    float64  `json:"confidence"`
}

// BuildSummaryPrompt constructs the Gemini prompt for cluster summarization.
func BuildSummaryPrompt(content string) string {
	content = strings.TrimSpace(content)
	if len([]rune(content)) > maxPromptContentRunes {
		content = string([]rune(content)[:maxPromptContentRunes])
	}

	return fmt.Sprintf(`You are a professional news editor.

Your task is to create a factual and neutral news summary.

Rules:
* Never invent facts.
* Never speculate.
* Never use clickbait.
* Use formal Indonesian language.
* Merge duplicate information.
* Focus on the most important facts.

Return ONLY valid JSON:

{
"headline": "",
"summary_short": "",
"summary_medium": "",
"key_points": [
""
],
"confidence": 0.95
}

Articles:
%s`, content)
}

// ParseSummaryResponse parses and validates Gemini's JSON response.
func ParseSummaryResponse(raw string) (*SummaryResponseJSON, error) {
	cleaned := cleanJSON(raw)
	var resp SummaryResponseJSON
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal summary JSON: %w", err)
	}

	resp.Headline = strings.TrimSpace(resp.Headline)
	resp.SummaryShort = strings.TrimSpace(resp.SummaryShort)
	resp.SummaryMedium = strings.TrimSpace(resp.SummaryMedium)
	for i := range resp.KeyPoints {
		resp.KeyPoints[i] = strings.TrimSpace(resp.KeyPoints[i])
	}
	resp.KeyPoints = compactStrings(resp.KeyPoints)

	if resp.Headline == "" {
		return nil, fmt.Errorf("headline is required")
	}
	if resp.SummaryShort == "" {
		return nil, fmt.Errorf("summary_short is required")
	}
	if len(resp.KeyPoints) == 0 {
		return nil, fmt.Errorf("key_points is required")
	}
	if resp.Confidence < 0 || resp.Confidence > 1 {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
	}

	return &resp, nil
}

func cleanJSON(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		var contentLines []string
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				continue
			}
			contentLines = append(contentLines, line)
		}
		cleaned = strings.TrimSpace(strings.Join(contentLines, "\n"))
	}
	return cleaned
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
