package prompt

import (
	"strings"
	"testing"
)

func TestBuildClassifyPrompt(t *testing.T) {
	title := "Gebrakan AI Terbaru"
	content := "Gemini 2.5 Flash dirilis dengan performa tinggi."
	promptStr := BuildClassifyPrompt(title, content)

	if !strings.Contains(promptStr, title) {
		t.Errorf("expected prompt to contain title, but it did not")
	}
	if !strings.Contains(promptStr, content) {
		t.Errorf("expected prompt to contain content, but it did not")
	}
	if !strings.Contains(promptStr, "Teknologi") {
		t.Errorf("expected prompt to list allowed categories, but it did not")
	}
}

func TestParseClassifyResponse(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantCat    string
		wantConf   float64
		wantErr    bool
		errSubstr  string
	}{
		{
			name:     "Valid JSON",
			raw:      `{"category": "Teknologi", "confidence": 0.95}`,
			wantCat:  "Teknologi",
			wantConf: 0.95,
			wantErr:  false,
		},
		{
			name:     "Markdown Wrapped JSON",
			raw:      "```json\n{\n  \"category\": \"Otomotif\",\n  \"confidence\": 0.88\n}\n```",
			wantCat:  "Otomotif",
			wantConf: 0.88,
			wantErr:  false,
		},
		{
			name:     "Case Insensitive Category",
			raw:      `{"category": "bisnis", "confidence": 0.75}`,
			wantCat:  "Bisnis",
			wantConf: 0.75,
			wantErr:  false,
		},
		{
			name:      "Invalid Category Fallback",
			raw:       `{"category": "Gossip", "confidence": 0.99}`,
			wantCat:   "Nasional",
			wantConf:  0.50,
			wantErr:   true,
			errSubstr: "invalid category",
		},
		{
			name:      "Malformed JSON",
			raw:       `{"category": "Politik", `,
			wantCat:   "Nasional",
			wantConf:  0.50,
			wantErr:   true,
			errSubstr: "failed to unmarshal JSON",
		},
		{
			name:     "Confidence Out of Bounds High",
			raw:      `{"category": "Teknologi", "confidence": 1.5}`,
			wantCat:  "Teknologi",
			wantConf: 0.50,
			wantErr:  false,
		},
		{
			name:     "Confidence Out of Bounds Low",
			raw:      `{"category": "Teknologi", "confidence": -0.2}`,
			wantCat:  "Teknologi",
			wantConf: 0.50,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, conf, err := ParseClassifyResponse(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseClassifyResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
			}
			if cat != tt.wantCat {
				t.Errorf("ParseClassifyResponse() cat = %v, want %v", cat, tt.wantCat)
			}
			if conf != tt.wantConf {
				t.Errorf("ParseClassifyResponse() conf = %v, want %v", conf, tt.wantConf)
			}
		})
	}
}
