package handler

import (
	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/ai/service"
	"koran-ai-backend/internal/shared/response"
)

// ProcessRequest defines the payload for triggering AI processing.
type ProcessRequest struct {
	Limit int `json:"limit"`
}

// Handler coordinates HTTP requests for AI categorization.
type Handler struct {
	svc service.Service
}

// NewHandler instantiates a new Handler with the given AI service.
func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// Process handles POST /internal/ai/process.
// Initiates classification for a batch of unprocessed articles.
//
// Body (optional JSON):
//
//	{ "limit": 100 }   // default: 100, max: 500
func (h *Handler) Process(c fiber.Ctx) error {
	const defaultLimit = 100
	const maxLimit = 500

	var req ProcessRequest
	req.Limit = defaultLimit

	// Parse JSON request body if present
	if c.Request().Body() != nil && len(c.Request().Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "Invalid request body: "+err.Error())
		}
	}

	if req.Limit <= 0 {
		req.Limit = defaultLimit
	}
	if req.Limit > maxLimit {
		req.Limit = maxLimit
	}

	res, err := h.svc.Process(c.Context(), req.Limit)
	if err != nil {
		if err.Error() == "worker already running" {
			return response.Error(c, fiber.StatusConflict, "AI Worker process is already running")
		}
		return response.Error(c, fiber.StatusInternalServerError, "AI processing failed: "+err.Error())
	}

	return response.JSON(c, fiber.StatusOK, "AI processing batch completed", res)
}

// GetStats handles GET /internal/ai/stats.
// Retrieves summary metrics of the AI engine categorization tasks.
func (h *Handler) GetStats(c fiber.Ctx) error {
	stats, err := h.svc.GetStats(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve AI stats: "+err.Error())
	}

	return response.JSON(c, fiber.StatusOK, "AI stats retrieved", stats)
}
