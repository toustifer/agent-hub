package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/hub/service"
)

type acquireReq struct {
	ResourceKey string `json:"resource_key"`
	WorkerID    string `json:"worker_id"`
	TTLSeconds  int    `json:"ttl_seconds"`
}
type renewReq   struct{ HolderToken string `json:"holder_token"`; TTLSeconds int `json:"ttl_seconds"` }
type releaseReq struct{ HolderToken string `json:"holder_token"` }

func (h *Handler) AcquireLock(c *gin.Context) {
	var req acquireReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.TTLSeconds <= 0 { req.TTLSeconds = 300 }
	bizCode := c.GetString("business_code")
	token, expiresAt, err := h.Svc.AcquireLock(c.Request.Context(), bizCode, req.ResourceKey, req.WorkerID, req.TTLSeconds)
	if err != nil {
		var held *service.LockHeldError
		if errors.As(err, &held) {
			c.JSON(http.StatusConflict, gin.H{"code": 409, "message": held.Error(), "data": gin.H{"holder_worker_id": held.HolderWorkerID}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"holder_token": token, "expires_at": expiresAt}})
}

func (h *Handler) RenewLock(c *gin.Context) {
	var req renewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.Svc.RenewLock(c.Request.Context(), req.HolderToken, req.TTLSeconds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func (h *Handler) ReleaseLock(c *gin.Context) {
	var req releaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.Svc.ReleaseLock(c.Request.Context(), req.HolderToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func (h *Handler) ListActiveLocks(c *gin.Context) {
	business := c.Query("business")
	list, err := h.Svc.ListActiveLocks(c.Request.Context(), business)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
