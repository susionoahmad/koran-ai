package prompt

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Categories is the official list of 15 allowed categories.
var Categories = []string{
	"Nasional",
	"Politik",
	"Ekonomi",
	"Bisnis",
	"Teknologi",
	"Internasional",
	"Olahraga",
	"Hiburan",
	"Pendidikan",
	"Kesehatan",
	"Otomotif",
	"Properti",
	"Travel",
	"Hukum",
	"Lingkungan",
}

// ClassifyResponseJSON represents the expected JSON response from Gemini.
type ClassifyResponseJSON struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

// BuildClassifyPrompt constructs the classification prompt for Gemini.
func BuildClassifyPrompt(title, content string) string {
	categoriesStr := strings.Join(Categories, ", ")
	return fmt.Sprintf(`Identify the category of the following news article.
Choose exactly one category from this list: [%s].

Return the output strictly in the following JSON format:
{
  "category": "CategoryName",
  "confidence": 0.95
}

Rules:
1. "category" must match exactly one of the allowed categories listed above.
2. "confidence" must be a float between 0.0 and 1.0 representing your confidence in this classification.
3. Output MUST be valid JSON. Do not write any explanations or markdown formatting (e.g. do not wrap the JSON in code block ticks).

Article Title: %s
Article Content:
%s`, categoriesStr, title, content)
}

// ParseClassifyResponse parses and validates Gemini's raw string response.
// If the category is invalid or parsing fails, it returns a descriptive error, but the caller should fallback to "Nasional" and 0.50.
func ParseClassifyResponse(raw string) (string, float64, error) {
	cleaned := strings.TrimSpace(raw)
	// Strip markdown code block wrappers if the model returned them defensively
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		var contentLines []string
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, "```") {
				continue
			}
			contentLines = append(contentLines, line)
		}
		cleaned = strings.TrimSpace(strings.Join(contentLines, "\n"))
	}

	var resp ClassifyResponseJSON
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return "Nasional", 0.50, fmt.Errorf("failed to unmarshal JSON: %w (raw response: %q)", err, raw)
	}

	// Validate the category (case-insensitive, but return the exact match from our list)
	isValid := false
	matchedCategory := ""
	for _, cat := range Categories {
		if strings.EqualFold(strings.TrimSpace(resp.Category), cat) {
			isValid = true
			matchedCategory = cat
			break
		}
	}

	if !isValid {
		return "Nasional", 0.50, fmt.Errorf("invalid category returned: %q", resp.Category)
	}

	if resp.Confidence < 0.0 || resp.Confidence > 1.0 {
		resp.Confidence = 0.50
	}

	return matchedCategory, resp.Confidence, nil
}
