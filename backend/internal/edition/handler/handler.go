package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	appLogger "koran-ai-backend/internal/shared/logger"
	"koran-ai-backend/internal/shared/response"
	edRepo "koran-ai-backend/internal/edition/repository"
	"koran-ai-backend/internal/edition/service"
)

type GenerateRequest struct {
	Date string `json:"date" validate:"required"`
}

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

// Generate handles POST /internal/editions/generate
func (h *Handler) Generate(c fiber.Ctx) error {
	var req GenerateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body: "+err.Error())
	}
	if req.Date == "" {
		return response.Error(c, fiber.StatusBadRequest, "date parameter is required")
	}

	// Validate date format YYYY-MM-DD
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid date format, must be YYYY-MM-DD")
	}

	edition, err := h.svc.GenerateEdition(c.Context(), req.Date)
	if err != nil {
		if errors.Is(err, service.ErrWorkerAlreadyRunning) {
			return response.Error(c, fiber.StatusConflict, "edition generation worker is already running")
		}
		if errors.Is(err, edRepo.ErrAlreadyExists) {
			return response.Error(c, fiber.StatusConflict, "edition already exists for this date")
		}
		if errors.Is(err, service.ErrNoSummariesFound) {
			return response.Error(c, fiber.StatusBadRequest, "no summaries found for this date")
		}
		if h.logger != nil {
			h.logger.Error("failed to generate edition", zap.Error(err))
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"edition_id": edition.ID.String(),
	})
}

// Stats handles GET /internal/editions/stats
func (h *Handler) Stats(c fiber.Ctx) error {
	total, latest, err := h.svc.GetStats(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to get edition stats: "+err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"total_editions": total,
		"latest_edition": latest,
	})
}

// List handles GET /api/v1/editions
func (h *Handler) List(c fiber.Ctx) error {
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	editions, total, err := h.svc.ListEditions(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to list editions: "+err.Error())
	}

	return response.Paginated(c, fiber.StatusOK, editions, fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// Detail handles GET /api/v1/editions/:id
func (h *Handler) Detail(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, fiber.StatusBadRequest, "edition id is required")
	}

	details, err := h.svc.GetEditionByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, edRepo.ErrNotFound) {
			return response.Error(c, fiber.StatusNotFound, "edition not found")
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve edition: "+err.Error())
	}

	return response.JSON(c, fiber.StatusOK, "edition retrieved", details)
}

// Latest handles GET /api/v1/editions/latest
func (h *Handler) Latest(c fiber.Ctx) error {
	editions, _, err := h.svc.ListEditions(c.Context(), 1, 1)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to check latest edition: "+err.Error())
	}

	if len(editions) == 0 {
		return response.Error(c, fiber.StatusNotFound, "no editions found")
	}

	details, err := h.svc.GetEditionByID(c.Context(), editions[0].ID.String())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve latest edition details: "+err.Error())
	}

	return response.JSON(c, fiber.StatusOK, "latest edition retrieved", details)
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
