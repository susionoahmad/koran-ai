package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	clusterRepo "koran-ai-backend/internal/clustering/repository"
	"koran-ai-backend/internal/clustering/service"
	"koran-ai-backend/internal/shared/response"
)

// Handler coordinates HTTP requests for the news clustering engine.
type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// Run handles POST /internal/clustering/run.
func (h *Handler) Run(c fiber.Ctx) error {
	result, err := h.svc.RunClustering(c.Context())
	if err != nil {
		if errors.Is(err, service.ErrWorkerAlreadyRunning) {
			return response.Error(c, fiber.StatusConflict, "clustering worker is already running")
		}
		return response.Error(c, fiber.StatusInternalServerError, "clustering failed: "+err.Error())
	}
	return response.JSON(c, fiber.StatusOK, "clustering completed", result)
}

// Stats handles GET /internal/clustering/stats.
func (h *Handler) Stats(c fiber.Ctx) error {
	stats, err := h.svc.GetStats(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve clustering stats: "+err.Error())
	}
	return response.JSON(c, fiber.StatusOK, "clustering stats retrieved", stats)
}

// List handles GET /api/v1/clusters.
func (h *Handler) List(c fiber.Ctx) error {
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	clusters, total, err := h.svc.ListClusters(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to list clusters: "+err.Error())
	}

	return response.Paginated(c, fiber.StatusOK, clusters, fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// Detail handles GET /api/v1/clusters/:id.
func (h *Handler) Detail(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, fiber.StatusBadRequest, "cluster id is required")
	}

	cluster, err := h.svc.GetClusterByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, clusterRepo.ErrNotFound) {
			return response.Error(c, fiber.StatusNotFound, "cluster not found")
		}
		return response.Error(c, fiber.StatusInternalServerError, "failed to retrieve cluster: "+err.Error())
	}
	return response.JSON(c, fiber.StatusOK, "cluster retrieved", cluster)
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
