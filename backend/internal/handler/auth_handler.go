package handler

import (
	"errors"
	"net/http"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			respondError(c, http.StatusConflict, "EMAIL_EXISTS", "An account with this email already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
		case errors.Is(err, service.ErrUserDisabled):
			respondError(c, http.StatusForbidden, "USER_DISABLED", "This account is disabled")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Data: resp})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if err := h.authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		if errors.Is(err, service.ErrTokenInvalid) {
			respondError(c, http.StatusGone, "TOKEN_INVALID", "Verification link is invalid or has expired")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.authService.ResendVerification(c.Request.Context(), userID); err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyVerified):
			respondError(c, http.StatusConflict, "ALREADY_VERIFIED", "Your email is already verified")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		}
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Even on bind failure we return 200 to preserve the uniform-response
		// anti-enumeration contract — only a real server error leaks.
		c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
		return
	}
	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrTokenInvalid) {
			respondError(c, http.StatusGone, "TOKEN_INVALID", "Reset link is invalid or has expired")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

// getUserID and respondError live in url_handler.go; shared across handlers.
