package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/version"
)

func (h *Handler) Health(c *gin.Context) {
	err := h.Svc.Pool.Ping(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"db":      err.Error(),
			"version": version.Info(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": version.Info(),
	})
}

// Version returns build metadata only (no DB).
func (h *Handler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": version.Info()})
}
