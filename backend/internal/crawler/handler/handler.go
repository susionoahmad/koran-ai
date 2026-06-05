package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/crawler/service"
	"koran-ai-backend/internal/shared/response"
)

// Handler handles internal HTTP requests for the crawler engine.
type Handler struct {
	svc service.Service
}

// NewHandler creates a new crawler Handler.
func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// RunSource handles POST /internal/crawler/run/:id
// Triggers a full crawl cycle for a single source.
func (h *Handler) RunSource(c fiber.Ctx) error {
	sourceID := c.Params("id")
	if sourceID == "" {
		return response.Error(c, fiber.StatusBadRequest, "source id is required")
	}

	result, err := h.svc.RunSource(c.Context(), sourceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSourceInactive):
			return response.Error(c, fiber.StatusBadRequest, "source is inactive")
		case errors.Is(err, service.ErrSourceNoRSS):
			return response.Error(c, fiber.StatusBadRequest, "source has no rss_url configured")
		default:
			return response.Error(c, fiber.StatusInternalServerError, "crawl failed: "+err.Error())
		}
	}

	return response.JSON(c, fiber.StatusOK, "crawl completed", fiber.Map{
		"source_id":      result.SourceID,
		"articles_found": result.ArticlesFound,
		"articles_saved": result.ArticlesSaved,
	})
}

// RunAll handles POST /internal/crawler/run-all
// Triggers a crawl for all active sources.
func (h *Handler) RunAll(c fiber.Ctx) error {
	results, err := h.svc.RunAllSources(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to start crawl: "+err.Error())
	}

	type sourceResult struct {
		SourceID      string `json:"source_id"`
		ArticlesFound int    `json:"articles_found"`
		ArticlesSaved int    `json:"articles_saved"`
		Error         string `json:"error,omitempty"`
	}

	out := make([]sourceResult, 0, len(results))
	for _, r := range results {
		sr := sourceResult{
			SourceID:      r.SourceID,
			ArticlesFound: r.ArticlesFound,
			ArticlesSaved: r.ArticlesSaved,
		}
		if r.Error != nil {
			sr.Error = r.Error.Error()
		}
		out = append(out, sr)
	}

	return response.JSON(c, fiber.StatusOK, "crawl-all completed", fiber.Map{
		"total_sources": len(out),
		"results":       out,
	})
}

// GetStats handles GET /internal/crawler/stats
// Retrieves crawler run stats and metrics.
func (h *Handler) GetStats(c fiber.Ctx) error {
	stats, err := h.svc.GetStats(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve stats: "+err.Error())
	}

	return response.JSON(c, fiber.StatusOK, "crawler stats retrieved", stats)
}
