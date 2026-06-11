package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/genai"
	"koran-ai-backend/internal/ai/prompt"
	summaryPrompt "koran-ai-backend/internal/summary/prompt"
)

// ClassifyResult represents the outcome of article classification.
type ClassifyResult struct {
	Category   string
	Confidence float64
}

// SummaryResult represents the outcome of cluster summarization.
type SummaryResult struct {
	Headline      string
	SummaryShort  string
	SummaryMedium string
	KeyPoints     []string
	Confidence    float64
}

// GeminiClient defines the client interface for article and cluster AI tasks.
type GeminiClient interface {
	ClassifyArticle(ctx context.Context, title string, content string) (*ClassifyResult, error)
	SummarizeCluster(ctx context.Context, content string) (*SummaryResult, error)
}

type geminiClient struct {
	client    *genai.Client
	modelName string
}

// NewGeminiClient instantiates a concrete GeminiClient.
func NewGeminiClient(apiKey string, modelName string) (GeminiClient, error) {
	return NewGeminiClientWithConfig(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	}, modelName)
}

// NewGeminiClientWithConfig allows creating a GeminiClient with a custom genai.ClientConfig (e.g. for mock HTTPClient in tests).
func NewGeminiClientWithConfig(ctx context.Context, cfg *genai.ClientConfig, modelName string) (GeminiClient, error) {
	c, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	return &geminiClient{
		client:    c,
		modelName: modelName,
	}, nil
}

// ClassifyArticle makes a 30s-timeout request to the Gemini API and parses the response.
// Real API failures return an error, whereas invalid formats/categories are handled using the "Nasional" fallback.
func (c *geminiClient) ClassifyArticle(ctx context.Context, title string, content string) (*ClassifyResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	promptStr := prompt.BuildClassifyPrompt(title, content)

	resp, err := c.client.Models.GenerateContent(
		timeoutCtx,
		c.modelName,
		genai.Text(promptStr),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
	}

	category, confidence, parseErr := prompt.ParseClassifyResponse(resp.Text())
	if parseErr != nil {
		// As per instructions, if the response is invalid (bad JSON / unlisted category),
		// we fallback to "Nasional" and 0.50 without stopping the process.
		// We still return nil error so that the worker stores this result.
		return &ClassifyResult{
			Category:   category,
			Confidence: confidence,
		}, nil
	}

	return &ClassifyResult{
		Category:   category,
		Confidence: confidence,
	}, nil
}

// SummarizeCluster makes a timeout-bound Gemini request and parses a strict JSON summary response.
func (c *geminiClient) SummarizeCluster(ctx context.Context, content string) (*SummaryResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	promptStr := summaryPrompt.BuildSummaryPrompt(content)
	resp, err := c.client.Models.GenerateContent(
		timeoutCtx,
		c.modelName,
		genai.Text(promptStr),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
	}

	parsed, err := summaryPrompt.ParseSummaryResponse(resp.Text())
	if err != nil {
		return nil, err
	}
	return &SummaryResult{
		Headline:      parsed.Headline,
		SummaryShort:  parsed.SummaryShort,
		SummaryMedium: parsed.SummaryMedium,
		KeyPoints:     parsed.KeyPoints,
		Confidence:    parsed.Confidence,
	}, nil
}
