package handler

import (
	"net/http"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

type PlanHandler struct {
	planService service.PlanService
}

func NewPlanHandler(planService service.PlanService) *PlanHandler {
	return &PlanHandler{planService: planService}
}

func (h *PlanHandler) List(c *gin.Context) {
	plans, err := h.planService.List(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: plans})
}
