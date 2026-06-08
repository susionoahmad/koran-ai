package service

import "testing"

func TestNormalizeTitle(t *testing.T) {
	got := NormalizeTitle("Bank Indonesia Pertahankan BI Rate!")
	want := "bank indonesia pertahankan bi rate"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCombinedSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		min  float64
		max  float64
	}{
		{
			name: "same topic crosses threshold",
			a:    "Bank Indonesia Pertahankan BI Rate",
			b:    "Bank Indonesia Pertahankan BI Rate",
			min:  SimilarityThreshold,
			max:  1.0,
		},
		{
			name: "different topic below threshold",
			a:    "Bank Indonesia Pertahankan BI Rate",
			b:    "Pemerintah Bangun Sekolah Baru",
			min:  0,
			max:  SimilarityThreshold - 0.01,
		},
		{
			name: "empty title has zero score",
			a:    "",
			b:    "Bank Indonesia Pertahankan BI Rate",
			min:  0,
			max:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CombinedSimilarity(tt.a, tt.b)
			if score < tt.min || score > tt.max {
				t.Fatalf("score %.4f outside range %.4f..%.4f", score, tt.min, tt.max)
			}
		})
	}
}

func TestCombinedSimilarity_ThresholdEdge(t *testing.T) {
	score := CombinedSimilarity("a b c d", "a b c d e")
	if score < SimilarityThreshold {
		t.Fatalf("expected edge case to meet threshold, got %.4f", score)
	}
}
