package handler

import (
	"errors"
	"net/http"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TagHandler struct {
	tagService service.TagService
}

func NewTagHandler(tagService service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

func (h *TagHandler) UpdateTags(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	urlID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid URL id")
		return
	}

	var req dto.UpdateURLTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	tags, err := h.tagService.UpdateTags(c.Request.Context(), urlID, userID, req.Tags)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrURLNotFound):
			respondError(c, http.StatusNotFound, "URL_NOT_FOUND", "URL not found")
		case errors.Is(err, service.ErrURLForbidden):
			respondError(c, http.StatusForbidden, "URL_FORBIDDEN", "You do not have access to this URL")
		case errors.Is(err, service.ErrTagLimitExceeded):
			respondError(c, http.StatusBadRequest, "TAG_LIMIT_EXCEEDED", "Maximum 20 tags allowed per URL")
		case errors.Is(err, service.ErrInvalidTag):
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Tags must be 1-50 characters")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: dto.URLTagsResponse{Tags: tags}})
}
