package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

// RateLimit returns a middleware that allows at most limit requests per window.
// Identity is the authenticated user ID when available (set by AuthRequired),
// falling back to client IP for public endpoints.
func RateLimit(limiter *ratelimit.Limiter, limit int64, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity := ctx.ClientIP()
		if userID, exists := ctx.Get("userID"); exists {
			identity = fmt.Sprintf("user:%v", userID)
		}
		key := ratelimit.Key(ctx.FullPath(), identity)

		allowed, err := limiter.Allow(ctx.Request.Context(), key, limit, window)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "INTERNAL_ERROR",
					Message: "Rate limiter unavailable",
				},
			})
			return
		}
		if !allowed {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Too many requests. Please slow down and try again later.",
				},
			})
			return
		}

		ctx.Next()
	}
}
