package prompt

import (
	"strings"
	"testing"
)

func TestBuildSummaryPrompt(t *testing.T) {
	promptStr := BuildSummaryPrompt("Artikel tentang ekonomi.")
	if !strings.Contains(promptStr, "professional news editor") {
		t.Fatal("expected editor instruction")
	}
	if !strings.Contains(promptStr, "formal Indonesian language") {
		t.Fatal("expected Indonesian language rule")
	}
	if !strings.Contains(promptStr, "Artikel tentang ekonomi.") {
		t.Fatal("expected article content")
	}
}

func TestParseSummaryResponse(t *testing.T) {
	raw := "```json\n{\"headline\":\"Judul\",\"summary_short\":\"Ringkas\",\"summary_medium\":\"Sedang\",\"key_points\":[\"Poin 1\", \"\"],\"confidence\":0.91}\n```"
	got, err := ParseSummaryResponse(raw)
	if err != nil {
		t.Fatalf("ParseSummaryResponse returned error: %v", err)
	}
	if got.Headline != "Judul" || got.SummaryShort != "Ringkas" || got.Confidence != 0.91 {
		t.Fatalf("unexpected parsed response: %+v", got)
	}
	if len(got.KeyPoints) != 1 || got.KeyPoints[0] != "Poin 1" {
		t.Fatalf("expected compact key points, got %+v", got.KeyPoints)
	}
}

func TestParseSummaryResponse_Invalid(t *testing.T) {
	tests := []string{
		`not json`,
		`{"headline":"","summary_short":"x","key_points":["p"],"confidence":0.5}`,
		`{"headline":"h","summary_short":"","key_points":["p"],"confidence":0.5}`,
		`{"headline":"h","summary_short":"s","key_points":[],"confidence":0.5}`,
		`{"headline":"h","summary_short":"s","key_points":["p"],"confidence":1.5}`,
	}
	for _, raw := range tests {
		if _, err := ParseSummaryResponse(raw); err == nil {
			t.Fatalf("expected error for %s", raw)
		}
	}
}
