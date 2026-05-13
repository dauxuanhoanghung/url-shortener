package router

import (
	"github.com/dauxuanhoanghung/url-shortener/internal/handler"
	"github.com/dauxuanhoanghung/url-shortener/internal/middleware"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	URL      *handler.URLHandler
	Redirect *handler.RedirectHandler
}

func Setup(mode string, jwtSecret string, userRepo repository.UserRepository, h Handlers) *gin.Engine {
	gin.SetMode(mode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/r/:short_code", h.Redirect.Redirect)

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.POST("/verify-email", h.Auth.VerifyEmail)
			auth.POST("/forgot-password", h.Auth.ForgotPassword)
			auth.POST("/reset-password", h.Auth.ResetPassword)
		}

		authed := api.Group("")
		authed.Use(middleware.AuthRequired(jwtSecret))
		{
			authed.POST("/auth/resend-verification", h.Auth.ResendVerification)

			urls := authed.Group("/urls")
			{
				// URL creation requires verified email once the grace period ends.
				urls.POST("", middleware.VerifiedEmailRequired(userRepo), h.URL.Create)
				urls.GET("", h.URL.List)
				urls.DELETE("/:id", h.URL.Delete)
			}
		}
	}

	return r
}
