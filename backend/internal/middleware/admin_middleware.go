package middleware

import (
	"net/http"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/gin-gonic/gin"
)

// RequireAdmin enforces that the caller's JWT carries role=admin.
// Must run after AuthRequired (which populates the "role" context key).
// Per docs/25-admin-accounts.md §1: the role claim is the authoritative check.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "ADMIN_REQUIRED",
					Message: "This endpoint requires an administrator account.",
				},
			})
			return
		}
		c.Next()
	}
}
