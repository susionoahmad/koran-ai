package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	appLogger "koran-ai-backend/internal/shared/logger"
	"koran-ai-backend/internal/shared/response"
	summaryRepo "koran-ai-backend/internal/summary/repository"
	"koran-ai-backend/internal/summary/service"
)

// GenerateRequest defines the payload for triggering summary generation.
type GenerateRequest struct {
	Limit int `json:"limit"`
}

// Handler coordinates HTTP requests for the AI summary engine.
type Handler struct {
	svc    service.Service
	logger appLogger.Logger
}

func NewHandler(svc service.Service, logger ...appLogger.Logger) *Handler {
	h := &Handler{svc: svc}
	if len(logger) > 0 {
		h.logger = logger[0]
	}
	return h
}

// Generate handles POST /internal/summaries/generate.
func (h *Handler) Generate(c fiber.Ctx) error {
	req := GenerateRequest{Limit: 50}
	if c.Request().Body() != nil && len(c.Request().Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "Invalid request body: "+err.Error())
		}
	}

	result, err := h.svc.GenerateBatch(c.Context(), req.Limit)
	if err != nil {
		if errors.Is(err, service.ErrWorkerAlreadyRunning) {
			return response.Error(c, fiber.StatusConflict, "summary worker is already running")
		}
		if h.logger != nil {
			h.logger.Error("summary generation failed", zap.Error(err))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}
	return response.JSON(c, fiber.StatusOK, "summary generation completed", result)
}

// Stats handles GET /internal/summaries/stats.
func (h *Handler) Stats(c fiber.Ctx) error {
	stats, err := h.svc.GetStats(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve summary stats: "+err.Error())
	}
	return response.JSON(c, fiber.StatusOK, "summary stats retrieved", stats)
}

// List handles GET /api/v1/summaries.
func (h *Handler) List(c fiber.Ctx) error {
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	summaries, total, err := h.svc.ListSummaries(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to list summaries: "+err.Error())
	}
	return response.Paginated(c, fiber.StatusOK, summaries, fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// Detail handles GET /api/v1/summaries/:id.
func (h *Handler) Detail(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, fiber.StatusBadRequest, "summary id is required")
	}

	summary, err := h.svc.GetSummaryByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, summaryRepo.ErrNotFound) {
			return response.Error(c, fiber.StatusNotFound, "summary not found")
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve summary: "+err.Error())
	}
	return response.JSON(c, fiber.StatusOK, "summary retrieved", summary)
}

// ByCluster handles GET /api/v1/clusters/:id/summary.
func (h *Handler) ByCluster(c fiber.Ctx) error {
	clusterID := c.Params("id")
	if clusterID == "" {
		return response.Error(c, fiber.StatusBadRequest, "cluster id is required")
	}

	summary, err := h.svc.GetSummaryByClusterID(c.Context(), clusterID)
	if err != nil {
		if errors.Is(err, summaryRepo.ErrNotFound) {
			return response.Error(c, fiber.StatusNotFound, "summary not found")
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve summary: "+err.Error())
	}
	return response.JSON(c, fiber.StatusOK, "summary retrieved", summary)
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
