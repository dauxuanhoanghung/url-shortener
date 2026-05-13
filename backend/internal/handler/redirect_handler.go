package handler

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9]{1,20}$`)

type RedirectHandler struct {
	redirectService service.RedirectService
}

func NewRedirectHandler(redirectService service.RedirectService) *RedirectHandler {
	return &RedirectHandler{redirectService: redirectService}
}

func (h *RedirectHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("short_code")

	if !shortCodePattern.MatchString(shortCode) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Error: dto.ErrorDetail{
				Code:    "SHORT_CODE_NOT_FOUND",
				Message: "Short URL not found",
			},
		})
		return
	}

	originalURL, err := h.redirectService.Resolve(c.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, service.ErrURLNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "SHORT_CODE_NOT_FOUND",
					Message: "Short URL not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Error: dto.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "An unexpected error occurred",
			},
		})
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}
