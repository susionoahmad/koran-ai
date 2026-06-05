package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/genai"
)

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClassifyArticle_Success(t *testing.T) {
	mockResponseJSON := `{
		"candidates": [{
			"content": {
				"parts": [{
					"text": "{\n  \"category\": \"Teknologi\",\n  \"confidence\": 0.92\n}"
				}]
			}
		}]
	}`

	httpClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseJSON)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := &genai.ClientConfig{
		APIKey:     "fake-api-key",
		HTTPClient: httpClient,
	}

	c, err := NewGeminiClientWithConfig(context.Background(), cfg, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	res, err := c.ClassifyArticle(context.Background(), "Title", "Content")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if res.Category != "Teknologi" {
		t.Errorf("expected category 'Teknologi', got %q", res.Category)
	}
	if res.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %f", res.Confidence)
	}
}

func TestClassifyArticle_APIError(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": {"message": "invalid request"}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := &genai.ClientConfig{
		APIKey:     "fake-api-key",
		HTTPClient: httpClient,
	}

	c, err := NewGeminiClientWithConfig(context.Background(), cfg, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = c.ClassifyArticle(context.Background(), "Title", "Content")
	if err == nil {
		t.Fatal("expected API error, got nil")
	}
	if !strings.Contains(err.Error(), "API call failed") {
		t.Errorf("expected error containing 'API call failed', got %q", err.Error())
	}
}

func TestClassifyArticle_Fallback(t *testing.T) {
	// Let's return malformed JSON from the API
	mockResponseJSON := `{
		"candidates": [{
			"content": {
				"parts": [{
					"text": "this is not json"
				}]
			}
		}]
	}`

	httpClient := &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseJSON)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := &genai.ClientConfig{
		APIKey:     "fake-api-key",
		HTTPClient: httpClient,
	}

	c, err := NewGeminiClientWithConfig(context.Background(), cfg, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	res, err := c.ClassifyArticle(context.Background(), "Title", "Content")
	if err != nil {
		t.Fatalf("expected nil error (fallback path), got %v", err)
	}

	if res.Category != "Nasional" {
		t.Errorf("expected category 'Nasional' (fallback), got %q", res.Category)
	}
	if res.Confidence != 0.50 {
		t.Errorf("expected confidence 0.50 (fallback), got %f", res.Confidence)
	}
}

func TestNewGeminiClient(t *testing.T) {
	c, err := NewGeminiClient("fake-api-key", "")
	if err != nil {
		t.Fatalf("expected nil error on NewGeminiClient creation, got %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

