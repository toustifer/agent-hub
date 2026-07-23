package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/config"
	hub "github.com/stifer/agent-hub/internal/hub"
	"github.com/stifer/agent-hub/internal/hub/handler"
	"github.com/stifer/agent-hub/internal/middleware"
)

func New(mw *middleware.Middleware, h *handler.Handler, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(mw.CORS(cfg.CORSOrigins))
	r.Use(mw.Logging())
	r.Use(gin.Recovery())
	// /health, /healthz, /version registered in hub.RegisterRoutes (includes build version)

	hub.RegisterRoutes(r, mw, h, cfg)

	staticDir := os.Getenv("HUB_STATIC_DIR")
	if staticDir == "" {
		staticDir = "static"
	}

	// AI-oriented public docs (no auth). Prefer static/, then repo docs/, then CWD.
	serveDoc := func(c *gin.Context, contentType string, names ...string) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Access-Control-Allow-Origin", "*")
		bases := []string{staticDir, "docs", ".", "frontend/public"}
		for _, base := range bases {
			for _, name := range names {
				p := filepath.Join(base, name)
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					if contentType != "" {
						c.Header("Content-Type", contentType)
					}
					c.File(p)
					return
				}
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "doc not found", "tried": names})
	}
	r.GET("/mcp.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "mcp.md", "MCP_FOR_AI.md")
	})
	r.GET("/MCP_FOR_AI.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "MCP_FOR_AI.md", "mcp.md")
	})
	r.GET("/agent-setup.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "agent-setup.md")
	})
	r.GET("/agentflow-setup.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "agentflow-setup.md", "HUB_SOFT_SYNC.md")
	})
	r.GET("/llms.txt", func(c *gin.Context) {
		serveDoc(c, "text/plain; charset=utf-8", "llms.txt")
	})
	r.GET("/agentflow-alignment.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "agentflow-alignment.md", "AGENTFLOW_ALIGNMENT.md")
	})
	r.GET("/AGENTFLOW_ALIGNMENT.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "AGENTFLOW_ALIGNMENT.md", "agentflow-alignment.md")
	})
	r.GET("/sync-contract.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "sync-contract.md", "SYNC_CONTRACT.md")
	})
	r.GET("/SYNC_CONTRACT.md", func(c *gin.Context) {
		serveDoc(c, "text/markdown; charset=utf-8", "SYNC_CONTRACT.md", "sync-contract.md")
	})
	// /setup is registered in hub.RegisterRoutes → setup.html at WorkingDirectory

	if _, err := os.Stat(staticDir); err == nil {
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if len(path) >= 4 && path[:4] == "/v1/" {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
				return
			}
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			// strip leading slash for join
			rel := path
			if len(rel) > 0 && rel[0] == '/' {
				rel = rel[1:]
			}
			filePath := filepath.Join(staticDir, rel)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				c.File(filepath.Join(staticDir, "index.html"))
				return
			}
			c.File(filePath)
		})
	}
	return r
}
