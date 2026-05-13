package middleware

import (
	"net/http"
	"strings"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/pkg/utils"
	"github.com/gin-gonic/gin"
)

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := tokenFromRequest(c)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "Authorization header is required",
				},
			})
			return
		}

		claims, err := utils.ValidateToken(raw, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "INVALID_TOKEN",
					Message: "Invalid or expired token",
				},
			})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// tokenFromRequest extracts the Bearer token from the Authorization header,
// falling back to the ?token= query param for SSE connections where browsers
// cannot set custom headers.
func tokenFromRequest(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
		return ""
	}
	return c.Query("token")
}
