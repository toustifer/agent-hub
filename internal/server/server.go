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
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	hub.RegisterRoutes(r, mw, h, cfg)

	staticDir := os.Getenv("HUB_STATIC_DIR")
	if staticDir == "" {
		staticDir = "static"
	}
	if _, err := os.Stat(staticDir); err == nil {
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if len(path) >= 4 && path[:4] == "/v1/" {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
				return
			}
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			filePath := filepath.Join(staticDir, path)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				c.File(filepath.Join(staticDir, "index.html"))
				return
			}
			c.File(filePath)
		})
	}
	return r
}
