package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/shared/response"
	"koran-ai-backend/internal/shared/validator"
	"koran-ai-backend/internal/source/dto"
	"koran-ai-backend/internal/source/service"
)

type Handler struct {
	svc service.Service
	val validator.Validator
}

func NewHandler(svc service.Service, val validator.Validator) *Handler {
	return &Handler{svc: svc, val: val}
}

// Create handles creating a new source.
// @Summary Create a new source
// @Description Create a new news source with unique name and base URL.
// @Tags Sources
// @Accept json
// @Produce json
// @Param request body dto.CreateSourceRequest true "Create Source Request"
// @Success 201 {object} response.SuccessEnvelope{data=dto.SourceResponse}
// @Failure 400 {object} response.ErrorEnvelope
// @Failure 409 {object} response.ErrorEnvelope
// @Failure 500 {object} response.ErrorEnvelope
// @Router /api/v1/sources [post]
func (h *Handler) Create(c fiber.Ctx) error {
	var req dto.CreateSourceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if errs := h.val.Validate(req); errs != nil {
		return response.Error(c, http.StatusBadRequest, "Validation failed", errs)
	}

	res, err := h.svc.Create(c.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateName) || errors.Is(err, service.ErrDuplicateBaseURL) {
			return response.Error(c, http.StatusConflict, err.Error())
		}
		return response.Error(c, http.StatusInternalServerError, "Internal Server Error")
	}

	return response.JSON(c, http.StatusCreated, "Source created successfully", res)
}

// GetByID handles retrieving a single source by ID.
// @Summary Get source by ID
// @Description Get detailed information of a source by its UUID.
// @Tags Sources
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} response.SuccessEnvelope{data=dto.SourceResponse}
// @Failure 404 {object} response.ErrorEnvelope
// @Failure 500 {object} response.ErrorEnvelope
// @Router /api/v1/sources/{id} [get]
func (h *Handler) GetByID(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, "Missing source ID")
	}

	res, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return response.Error(c, http.StatusNotFound, "Source not found")
		}
		return response.Error(c, http.StatusInternalServerError, "Internal Server Error")
	}

	return response.JSON(c, http.StatusOK, "Success", res)
}

// List handles listing sources with pagination.
// @Summary List sources
// @Description Retrieve news sources with page and limit pagination parameters.
// @Tags Sources
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Results limit (default: 20, max: 100)"
// @Success 200 {object} response.PaginatedEnvelope{data=[]dto.SourceResponse}
// @Failure 500 {object} response.ErrorEnvelope
// @Router /api/v1/sources [get]
func (h *Handler) List(c fiber.Ctx) error {
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	res, err := h.svc.List(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Internal Server Error")
	}

	meta := fiber.Map{
		"total": res.Total,
		"page":  res.Page,
		"limit": res.Limit,
	}

	return response.Paginated(c, http.StatusOK, res.Sources, meta)
}

// Update handles updating an existing source.
// @Summary Update source
// @Description Update name, base URL, RSS URL, source type, and active status of a source.
// @Tags Sources
// @Accept json
// @Produce json
// @Param id path string true "Source ID"
// @Param request body dto.UpdateSourceRequest true "Update Source Request"
// @Success 200 {object} response.SuccessEnvelope{data=dto.SourceResponse}
// @Failure 400 {object} response.ErrorEnvelope
// @Failure 404 {object} response.ErrorEnvelope
// @Failure 409 {object} response.ErrorEnvelope
// @Failure 500 {object} response.ErrorEnvelope
// @Router /api/v1/sources/{id} [put]
func (h *Handler) Update(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, "Missing source ID")
	}

	var req dto.UpdateSourceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Invalid request body")
	}

	if errs := h.val.Validate(req); errs != nil {
		return response.Error(c, http.StatusBadRequest, "Validation failed", errs)
	}

	res, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return response.Error(c, http.StatusNotFound, "Source not found")
		}
		if errors.Is(err, service.ErrDuplicateName) || errors.Is(err, service.ErrDuplicateBaseURL) {
			return response.Error(c, http.StatusConflict, err.Error())
		}
		return response.Error(c, http.StatusInternalServerError, "Internal Server Error")
	}

	return response.JSON(c, http.StatusOK, "Source updated successfully", res)
}

// Delete handles soft-deleting a source.
// @Summary Soft-delete source
// @Description Deactivates a news source by setting is_active=false.
// @Tags Sources
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} response.SuccessEnvelope
// @Failure 404 {object} response.ErrorEnvelope
// @Failure 500 {object} response.ErrorEnvelope
// @Router /api/v1/sources/{id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, "Missing source ID")
	}

	err := h.svc.Delete(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return response.Error(c, http.StatusNotFound, "Source not found")
		}
		return response.Error(c, http.StatusInternalServerError, "Internal Server Error")
	}

	return response.JSON(c, http.StatusOK, "Source deleted successfully", nil)
}
