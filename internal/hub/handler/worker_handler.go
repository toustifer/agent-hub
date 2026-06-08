package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type heartbeatReq struct {
	WorkerID string `json:"worker_id"`
	Version  string `json:"version"`
	Host     string `json:"host"`
	PID      int    `json:"pid"`
}

func (h *Handler) Heartbeat(c *gin.Context) {
	var req heartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	bizID := c.GetInt64("business_id")
	var bizCode string
	if err := h.Svc.Pool.QueryRow(c.Request.Context(), "SELECT code FROM hub.hub_businesses WHERE id=$1", bizID).Scan(&bizCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "business not found"})
		return
	}

	w, err := h.Svc.Heartbeat(c.Request.Context(), bizCode, req.WorkerID, req.Version, req.Host, req.PID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w})
}

func (h *Handler) ListWorkers(c *gin.Context) {
	business := c.Query("business")
	status := c.Query("status")
	list, err := h.Svc.ListWorkersByBusiness(c.Request.Context(), business, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
