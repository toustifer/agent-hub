package hub

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/config"
	"github.com/stifer/agent-hub/internal/hub/handler"
	"github.com/stifer/agent-hub/internal/middleware"
)

func RegisterRoutes(r *gin.Engine, mw *middleware.Middleware, h *handler.Handler, cfg *config.Config) {
	r.GET("/setup", func(c *gin.Context) { c.File("setup.html") })
	// Generate unique business code — returns "siruoning" or "siruoning-a3f8" if taken
	r.POST("/v1/hub/businesses/generate-code", h.GenerateBusinessCode)
	// MCP OAuth metadata — tells Claude Code how to authenticate
	r.GET("/.well-known/oauth-authorization-server", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"issuer":                                "https://hub.stifer.xyz",
			"authorization_endpoint":                "https://hub.stifer.xyz/v1/hub/oauth/authorize",
			"token_endpoint":                        "https://hub.stifer.xyz/v1/hub/oauth/device/token",
			"registration_endpoint":                 "https://hub.stifer.xyz/v1/hub/oauth/register",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code"},
			"token_endpoint_auth_methods_supported": []string{"none"},
		})
	})
	// OAuth dynamic client registration (RFC 7591)
	r.POST("/v1/hub/oauth/register", func(c *gin.Context) {
		c.JSON(201, gin.H{
			"client_id":                  "agent-hub-mcp",
			"client_name":                "Agent Hub MCP Client",
			"redirect_uris":              []string{"http://localhost:0/callback"},
			"token_endpoint_auth_method": "none",
			"grant_types":                []string{"urn:ietf:params:oauth:grant-type:device_code"},
			"response_types":             []string{"code"},
			"client_id_issued_at":        0,
		})
	})
	r.POST("/v1/hub/oauth/device/authorize", h.OAuthDeviceAuthorize)
	r.POST("/v1/hub/oauth/device/token", h.OAuthDeviceToken)
	// Auth code → device flow bridge: redirect browser to device approval page with generated code
	r.GET("/v1/hub/oauth/authorize", h.OAuthAuthorizeRedirect)
	// OAuth device confirmation (existing, user clicks in browser)
	r.GET("/v1/hub/auth/device/confirm", h.DeviceConfirm)
	r.POST("/v1/hub/auth/device/confirm", h.DeviceConfirm)
	mcpTarget, _ := url.Parse("http://127.0.0.1:9001")
	r.Any("/mcp", func(c *gin.Context) { httputil.NewSingleHostReverseProxy(mcpTarget).ServeHTTP(c.Writer, c.Request) })
	r.Any("/mcp/*path", func(c *gin.Context) { c.Request.URL.Path = "/" + c.Param("path"); httputil.NewSingleHostReverseProxy(mcpTarget).ServeHTTP(c.Writer, c.Request) })
	r.POST("/v1/hub/auth/register", h.Register)
	r.POST("/v1/hub/auth/login", h.Login)
	r.POST("/v1/hub/auth/device", h.DeviceAuth)
	r.GET("/healthz", h.Health)
	r.GET("/v1/hub/auth/device/token", h.DeviceToken)
	userAuth := r.Group("/v1/hub")
	userAuth.Use(mw.JWT(cfg.JWTSecret))
	{
		userAuth.GET("/me/businesses", h.GetMyBusinesses)
		userAuth.POST("/businesses/:code/join", h.JoinBusiness)
	}

	admin := r.Group("/v1/hub")
	admin.Use(mw.JWT(cfg.JWTSecret))
	{
		admin.POST("/businesses", h.CreateBusiness)
		admin.GET("/businesses", h.ListBusinesses)
		admin.PUT("/businesses/:id", h.UpdateBusiness)
		admin.GET("/workers", h.ListWorkers)
		admin.GET("/locks", h.ListActiveLocks)
		admin.GET("/events", h.ListEvents)
		admin.GET("/events/stream", h.StreamEvents)
		admin.GET("/playbooks/:id", h.GetPlaybookByID)
		admin.GET("/playbooks/search", h.SearchPlaybooks)
		admin.POST("/repos/:code", h.AddRepo)
		admin.GET("/repos/:code", h.ListRepos)
		admin.DELETE("/repos/:id", h.DeleteRepo)
		admin.GET("/dag/:code", h.GetDAG)
		admin.POST("/dag/:code", h.SyncDAG)
		admin.POST("/community/workers", h.PublishWorker)
		admin.GET("/community/workers", h.ListCommunityWorkers)
		admin.GET("/community/workers/:id", h.GetCommunityWorker)
		admin.POST("/community/workers/:id/install", h.InstallWorker)
		admin.GET("/community/workers/:id/reviews", h.ListCommunityWorkerReviews)
		admin.POST("/community/workers/:id/reviews", h.AddCommunityWorkerReview)
		admin.POST("/businesses/:code/invite", h.InviteMember)
	}

	worker := r.Group("/v1/hub")
	worker.Use(mw.APIKey())
	{
		worker.GET("/businesses/:code", h.GetBusinessByCode)
		worker.POST("/workers/heartbeat", h.Heartbeat)
		worker.POST("/locks/acquire", h.AcquireLock)
		worker.POST("/locks/renew", h.RenewLock)
		worker.POST("/locks/release", h.ReleaseLock)
		worker.POST("/playbooks", h.CreatePlaybook)
		worker.POST("/events", h.AppendEvent)
		worker.POST("/sync/workers", h.SyncWorkers)
	}
}
