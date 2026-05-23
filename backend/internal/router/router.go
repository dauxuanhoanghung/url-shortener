package router

import (
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/handler"
	"github.com/dauxuanhoanghung/url-shortener/internal/middleware"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	URL      *handler.URLHandler
	Redirect *handler.RedirectHandler
	Plan     *handler.PlanHandler
	SSE      *handler.SSEHandler
	Admin    *handler.AdminHandler
}

func Setup(mode string, jwtSecret string, userRepo repository.UserRepository, appCache cache.Cache, h Handlers) *gin.Engine {
	gin.SetMode(mode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	limiter := ratelimit.New(appCache)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/r/:short_code", h.Redirect.Redirect)

	api := r.Group("/api/v1")
	{
		// Public — no auth required
		api.GET("/plans", h.Plan.List)

		auth := api.Group("/auth")
		{
			authRL := middleware.RateLimit(limiter, 10, time.Minute)
			auth.POST("/register", authRL, h.Auth.Register)
			auth.POST("/login", authRL, h.Auth.Login)
			auth.POST("/verify-email", h.Auth.VerifyEmail)
			auth.POST("/forgot-password", authRL, h.Auth.ForgotPassword)
			auth.POST("/reset-password", h.Auth.ResetPassword)
		}

		authed := api.Group("")
		authed.Use(middleware.AuthRequired(jwtSecret))
		{
			authed.POST("/auth/resend-verification", h.Auth.ResendVerification)
			authed.GET("/events", h.SSE.Stream)

			urls := authed.Group("/urls")
			{
				// URL creation requires verified email once the grace period ends.
				urls.POST("", middleware.VerifiedEmailRequired(userRepo), middleware.RateLimit(limiter, 30, time.Minute), h.URL.Create)
				urls.GET("", h.URL.List)
				urls.DELETE("/:id", h.URL.Delete)
			}

			admin := authed.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/users", h.Admin.ListUsers)
				admin.GET("/users/:id", h.Admin.GetUser)
				admin.POST("/users/:id/disable", h.Admin.DisableUser)
				admin.POST("/users/:id/enable", h.Admin.EnableUser)
				admin.PATCH("/plans/:code/features", h.Admin.UpdatePlanFeatures)
				admin.GET("/audit", h.Admin.ListAudit)
			}
		}
	}

	return r
}
