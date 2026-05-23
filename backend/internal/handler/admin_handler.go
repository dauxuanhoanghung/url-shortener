package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	svc service.AdminService
}

func NewAdminHandler(svc service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit, offset := paginationFromQuery(c)
	resp, err := h.svc.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid user ID")
		return
	}
	resp, err := h.svc.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch user")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *AdminHandler) DisableUser(c *gin.Context) { h.setUserDisabled(c, true) }
func (h *AdminHandler) EnableUser(c *gin.Context)  { h.setUserDisabled(c, false) }

func (h *AdminHandler) setUserDisabled(c *gin.Context, disabled bool) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid user ID")
		return
	}
	actorID, ok := actorIDFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing actor context")
		return
	}
	resp, err := h.svc.SetUserDisabled(c.Request.Context(), actorID, targetID, disabled)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCannotDisableSelf):
			respondError(c, http.StatusBadRequest, "CANNOT_DISABLE_SELF", "You cannot disable your own account")
		case errors.Is(err, repository.ErrUserNotFound):
			respondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update user")
		}
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *AdminHandler) UpdatePlanFeatures(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Plan code is required")
		return
	}
	var req dto.UpdatePlanFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	actorID, ok := actorIDFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing actor context")
		return
	}
	plan, err := h.svc.UpdatePlanFeatures(c.Request.Context(), actorID, code, req.Features)
	if err != nil {
		if errors.Is(err, repository.ErrPlanNotFound) {
			respondError(c, http.StatusNotFound, "PLAN_NOT_FOUND", "Plan not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update plan features")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: plan})
}

func (h *AdminHandler) ListAudit(c *gin.Context) {
	limit, offset := paginationFromQuery(c)
	resp, err := h.svc.ListAudit(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list audit log")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: resp})
}

// --- helpers ---------------------------------------------------------------

func paginationFromQuery(c *gin.Context) (limit int32, offset int32) {
	if v, err := strconv.Atoi(c.Query("limit")); err == nil {
		limit = int32(v)
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil {
		offset = int32(v)
	}
	return limit, offset
}

func actorIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
