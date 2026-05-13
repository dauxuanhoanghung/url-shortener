package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type URLHandler struct {
	urlService service.URLService
	baseURL    string
}

func NewURLHandler(urlService service.URLService, baseURL string) *URLHandler {
	return &URLHandler{urlService: urlService, baseURL: baseURL}
}

func (h *URLHandler) Create(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := h.urlService.Create(c.Request.Context(), userID, req, h.baseURL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			respondError(c, http.StatusBadRequest, "INVALID_URL", "The provided URL is not valid")
		case errors.Is(err, service.ErrPlanLimitReached):
			respondError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the maximum number of URLs for your plan")
		case errors.Is(err, service.ErrShortCodeRetry):
			respondError(c, http.StatusInternalServerError, "SHORT_CODE_UNAVAILABLE", "Unable to generate a unique short code, please try again")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *URLHandler) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	resp, err := h.urlService.List(c.Request.Context(), userID, limit, offset, h.baseURL)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *URLHandler) Delete(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid URL id")
		return
	}

	if err := h.urlService.Delete(c.Request.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrURLNotFound):
			respondError(c, http.StatusNotFound, "SHORT_CODE_NOT_FOUND", "URL not found")
		case errors.Is(err, service.ErrURLForbidden):
			respondError(c, http.StatusForbidden, "FORBIDDEN", "You do not have access to this URL")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func getUserID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("userID")
	if !exists {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user context")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user context")
		return uuid.Nil, false
	}
	return id, true
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.ErrorResponse{
		Success: false,
		Error: dto.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
