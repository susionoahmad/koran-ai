package service

import (
	"regexp"
	"strings"
)

const SimilarityThreshold = 0.70

var (
	punctuationPattern = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	spacePattern       = regexp.MustCompile(`\s+`)
	stopwords          = map[string]struct{}{
		"yang": {}, "dan": {}, "di": {}, "ke": {}, "dari": {}, "untuk": {},
		"dengan": {}, "pada": {}, "dalam": {}, "atas": {}, "atau": {},
		"ini": {}, "itu": {}, "sebagai": {}, "karena": {}, "oleh": {},
		"akan": {}, "telah": {}, "sudah": {}, "adalah": {}, "para": {},
		"sebuah": {}, "satu": {}, "soal": {}, "terkait": {},
	}
)

// NormalizeTitle lowercases, removes punctuation, strips stopwords, and collapses spaces.
func NormalizeTitle(title string) string {
	lowered := strings.ToLower(title)
	noPunctuation := punctuationPattern.ReplaceAllString(lowered, " ")
	parts := strings.Fields(spacePattern.ReplaceAllString(noPunctuation, " "))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if _, ok := stopwords[part]; ok {
			continue
		}
		tokens = append(tokens, part)
	}
	return strings.Join(tokens, " ")
}

// Tokenize returns normalized title tokens.
func Tokenize(title string) []string {
	normalized := NormalizeTitle(title)
	if normalized == "" {
		return nil
	}
	return strings.Fields(normalized)
}

// CombinedSimilarity computes weighted Jaccard and token-overlap similarity.
func CombinedSimilarity(a string, b string) float64 {
	tokensA := Tokenize(a)
	tokensB := Tokenize(b)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	intersection, union := tokenSetStats(tokensA, tokensB)
	jaccard := float64(intersection) / float64(union)

	shorter := len(uniqueTokens(tokensA))
	if bUnique := len(uniqueTokens(tokensB)); bUnique < shorter {
		shorter = bUnique
	}
	overlap := float64(intersection) / float64(shorter)

	return (jaccard * 0.7) + (overlap * 0.3)
}

func tokenSetStats(a []string, b []string) (int, int) {
	setA := uniqueTokens(a)
	setB := uniqueTokens(b)
	intersection := 0
	for token := range setA {
		if _, ok := setB[token]; ok {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	return intersection, union
}

func uniqueTokens(tokens []string) map[string]struct{} {
	out := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		out[token] = struct{}{}
	}
	return out
}
