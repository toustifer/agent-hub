package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/hub/service"
)

type appendEventReq struct {
	BusinessCode  string                 `json:"business_code"`
	Actor         string                 `json:"actor"`
	EventType     string                 `json:"event_type" binding:"required"`
	Payload       map[string]interface{} `json:"payload"`
	ActorRole     string                 `json:"actor_role"`
	ActorWorkerID string                 `json:"actor_worker_id"`
}

func (h *Handler) AppendEvent(c *gin.Context) {
	var req appendEventReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	bizCode := req.BusinessCode
	if bizCode == "" {
		bizCode = c.GetString("business_code")
	}
	if bizCode == "" {
		bizCode = c.Query("business")
	}
	if bizCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "business_code required"})
		return
	}
	if _, _, ok := h.RequireMembership(c, bizCode); !ok {
		return
	}

	st := stampActorFromGin(c, req.ActorRole, req.ActorWorkerID, req.Actor)
	id, err := h.Svc.AppendEventFull(c.Request.Context(), service.AppendEventInput{
		BusinessCode:  bizCode,
		Actor:         st.LegacyActor,
		EventType:     req.EventType,
		Payload:       req.Payload,
		ActorUserID:   userIDPtr(st.UserID),
		ActorEmail:    st.Email,
		ActorRole:     st.Role,
		ActorWorkerID: st.WorkerID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":              id,
		"actor":           st.LegacyActor,
		"actor_user_id":   st.UserID,
		"actor_email":     st.Email,
		"actor_role":      st.Role,
		"actor_worker_id": st.WorkerID,
		"event_type":      req.EventType,
	}})
}

func (h *Handler) ListEvents(c *gin.Context) {
	business := c.Query("business")
	if business == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "business query required"})
		return
	}
	if _, _, ok := h.RequireMembership(c, business); !ok {
		return
	}
	eventType := c.Query("type")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid limit"})
		return
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.Svc.ListEventRows(c.Request.Context(), business, eventType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) StreamEvents(c *gin.Context) {
	business := c.Query("business")
	if business == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "business query required"})
		return
	}
	if _, _, ok := h.RequireMembership(c, business); !ok {
		return
	}
	sinceStr := c.DefaultQuery("since", "")
	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid since format"})
			return
		}
	}
	ch, err := h.Svc.StreamEvents(c.Request.Context(), business, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "streaming not supported"})
		return
	}
	for ev := range ch {
		fmt.Fprintf(c.Writer, "data: {\"id\":%d,\"actor\":\"%s\",\"event_type\":\"%s\",\"created_at\":\"%s\"}\n\n",
			ev.ID, ev.Actor, ev.EventType, ev.CreatedAt.Format(time.RFC3339))
		flusher.Flush()
	}
}
