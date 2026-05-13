package middleware

import (
	"net/http"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VerificationGracePeriod is how long a new user can keep using mutation
// endpoints before their email must be verified. See
// docs/24-user-account-lifecycle.md §2.1.
const VerificationGracePeriod = 7 * 24 * time.Hour

// VerifiedEmailRequired blocks the request if the caller is past the grace
// window and still has no email_verified_at. Must run after AuthRequired.
func VerifiedEmailRequired(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get("userID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   dto.ErrorDetail{Code: "UNAUTHORIZED", Message: "Missing user context"},
			})
			return
		}
		id, err := uuid.Parse(raw.(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   dto.ErrorDetail{Code: "UNAUTHORIZED", Message: "Invalid user context"},
			})
			return
		}
		user, err := userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Error:   dto.ErrorDetail{Code: "UNAUTHORIZED", Message: "User not found"},
			})
			return
		}
		if !user.IsEmailVerified() && time.Since(user.CreatedAt) > VerificationGracePeriod {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Error: dto.ErrorDetail{
					Code:    "EMAIL_VERIFICATION_REQUIRED",
					Message: "Please verify your email to continue.",
				},
			})
			return
		}
		c.Next()
	}
}
