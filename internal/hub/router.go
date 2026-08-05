package hub

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/config"
	"github.com/stifer/agent-hub/internal/hub/handler"
	hubmcp "github.com/stifer/agent-hub/internal/mcp"
	"github.com/stifer/agent-hub/internal/middleware"
)

func publicBaseURL() string {
	base := strings.TrimRight(os.Getenv("HUB_PUBLIC_URL"), "/")
	if base == "" {
		base = "https://hub.stifer.xyz"
	}
	return base
}

func RegisterRoutes(r *gin.Engine, mw *middleware.Middleware, h *handler.Handler, cfg *config.Config) {
	base := publicBaseURL()

	r.GET("/setup", func(c *gin.Context) { c.File("setup.html") })
	// Generate unique business code — returns "siruoning" or "siruoning-a3f8" if taken
	r.POST("/v1/hub/businesses/generate-code", h.GenerateBusinessCode)

	// MCP OAuth metadata — tells Claude Code how to authenticate (issuer from env)
	r.GET("/.well-known/oauth-authorization-server", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"issuer":                                 base,
			"authorization_endpoint":                 base + "/v1/hub/oauth/authorize",
			"token_endpoint":                         base + "/v1/hub/oauth/device/token",
			"registration_endpoint":                  base + "/v1/hub/oauth/register",
			"device_authorization_endpoint":          base + "/v1/hub/oauth/device/authorize",
			"response_types_supported":               []string{"code"},
			"grant_types_supported":                  []string{"authorization_code", "urn:ietf:params:oauth:grant-type:device_code"},
			"token_endpoint_auth_methods_supported":  []string{"none"},
			"code_challenge_methods_supported":       []string{"S256", "plain"},
			"scopes_supported":                       []string{"openid", "profile", "mcp"},
		})
	})
	// OAuth dynamic client registration (RFC 7591) — persist client when possible
	r.POST("/v1/hub/oauth/register", h.OAuthRegister)

	r.POST("/v1/hub/oauth/device/authorize", h.OAuthDeviceAuthorize)
	r.POST("/v1/hub/oauth/device/token", h.OAuthDeviceToken)
	// Auth code → device flow bridge: redirect browser to device approval page with generated code
	r.GET("/v1/hub/oauth/authorize", h.OAuthAuthorizeRedirect)

	// In-process Streamable HTTP MCP (official go-sdk). Replaces reverse-proxy to :9001.
	mcpHub := hubmcp.New(h.Svc, cfg.JWTSecret)
	mcpHandler := mcpHub.HTTPHandler()
	r.Any("/mcp", gin.WrapH(mcpHandler))
	r.Any("/mcp/*path", gin.WrapH(mcpHandler))

	r.POST("/v1/hub/auth/register", h.Register)
	r.POST("/v1/hub/auth/login", h.Login)
	r.GET("/v1/hub/auth/verify-email", h.VerifyEmail)
	r.POST("/v1/hub/auth/verify-email", h.VerifyEmail)
	r.POST("/v1/hub/auth/resend-verification", h.ResendVerification)
	r.POST("/v1/hub/auth/device", h.DeviceAuth)
	r.GET("/healthz", h.Health)
	r.GET("/health", h.Health)
	r.GET("/version", h.Version)
	r.GET("/v1/hub/auth/device/token", h.DeviceToken)

	userAuth := r.Group("/v1/hub")
	userAuth.Use(mw.JWT(cfg.JWTSecret))
	{
		// Device confirm requires a real logged-in user; JWT re-signed on confirm
		userAuth.GET("/auth/device/confirm", h.DeviceConfirm)
		userAuth.POST("/auth/device/confirm", h.DeviceConfirm)
		userAuth.GET("/me/businesses", h.GetMyBusinesses)
		userAuth.POST("/businesses/:code/join", h.JoinBusiness)
		userAuth.POST("/businesses/:code/link-requests", h.CreateLinkRequest)
		userAuth.POST("/invites/accept", h.AcceptInvite)
	}

	admin := r.Group("/v1/hub")
	admin.Use(mw.JWT(cfg.JWTSecret))
	{
		admin.POST("/businesses", h.CreateBusiness)
		admin.GET("/businesses", h.ListBusinesses)
		admin.PUT("/businesses/:id", h.UpdateBusiness)
		admin.PATCH("/businesses/:code/profile", h.PatchBusinessProfile)
		admin.GET("/workers", h.ListWorkers)
		admin.GET("/locks", h.ListActiveLocks)
		admin.GET("/events", h.ListEvents)
		admin.POST("/events", h.AppendEvent)
		admin.GET("/events/stream", h.StreamEvents)
		admin.GET("/playbooks/:id", h.GetPlaybookByID)
		admin.GET("/playbooks/search", h.SearchPlaybooks)
		// Team docs + worker templates (collab sync v0.2)
		admin.POST("/businesses/:code/docs", h.UpsertTeamDoc)
		admin.GET("/businesses/:code/docs", h.ListTeamDocs)
		admin.GET("/businesses/:code/docs/:key", h.GetTeamDoc)
		admin.POST("/businesses/:code/worker-templates", h.PublishWorkerTemplate)
		admin.GET("/businesses/:code/worker-templates", h.ListWorkerTemplates)
		admin.GET("/businesses/:code/worker-templates/:worker_id", h.GetWorkerTemplate)
		admin.POST("/repos/:code", h.AddRepo)
		admin.GET("/repos/:code", h.ListRepos)
		admin.DELETE("/repos/:id", h.DeleteRepo)
		admin.GET("/dag/:code", h.GetDAG)
		admin.POST("/dag/:code", h.SyncDAG)
		// Requirements for non-technical collaboration
		admin.POST("/requirements/:code", h.CreateRequirement)
		admin.GET("/requirements/:code", h.ListRequirements)
		admin.GET("/requirements/:code/:id", h.GetRequirement)
		admin.PATCH("/requirements/:code/:id", h.UpdateRequirement)
		admin.POST("/requirements/:code/:id/submit", h.SubmitRequirement)
		admin.POST("/requirements/:code/:id/accept", h.AcceptRequirement)
		admin.POST("/requirements/:code/:id/reject", h.RejectRequirement)
		admin.POST("/requirements/:code/:id/cancel", h.CancelRequirement)
		admin.POST("/requirements/:code/:id/comments", h.CreateComment)
		admin.GET("/requirements/:code/:id/comments", h.ListComments)
		admin.POST("/requirements/:code/:id/link", h.LinkRequirementTask)
		admin.DELETE("/requirements/:code/:id/link/:task_id", h.UnlinkRequirementTask)
		admin.POST("/community/workers", h.PublishWorker)
		admin.GET("/community/workers", h.ListCommunityWorkers)
		admin.GET("/community/workers/:id", h.GetCommunityWorker)
		admin.POST("/community/workers/:id/install", h.InstallWorker)
		admin.GET("/community/workers/:id/reviews", h.ListCommunityWorkerReviews)
		admin.POST("/community/workers/:id/reviews", h.AddCommunityWorkerReview)
		admin.POST("/businesses/:code/invite", h.InviteMember)
		admin.GET("/businesses/:code/invites", h.ListInvites)
		admin.POST("/businesses/:code/invites/:id/revoke", h.RevokeInvite)
		admin.GET("/businesses/:code/members", h.ListMembers)
		admin.GET("/businesses/:code/link-requests", h.ListLinkRequests)
		admin.POST("/businesses/:code/link-requests/:id/review", h.ReviewLinkRequest)
		// Branch index under JWT group only (middleware accepts API key via tryAPIKey).
		// Do not also register under worker group — gin panics on duplicate paths.
		admin.GET("/repos/:code/branches", h.ListBranches)
		admin.POST("/repos/:code/branches/report", h.ReportBranches)
		admin.POST("/repos/:code/branches/bind", h.BindBranch)
		admin.POST("/repos/:code/branches/unbind", h.UnbindBranch)
		admin.POST("/repos/:code/branches/refresh", h.RefreshBranches)
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
		// POST /events only on JWT group (API key via tryAPIKey in mw.JWT)
		worker.POST("/sync/workers", h.SyncWorkers)
	}
}
