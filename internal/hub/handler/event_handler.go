package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type appendEventReq struct {
	Actor     string                 `json:"actor"`
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
}

func (h *Handler) AppendEvent(c *gin.Context) {
	var req appendEventReq
if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	bizCode := c.GetString("business_code")
	ev, err := h.Svc.AppendEvent(c.Request.Context(), bizCode, req.Actor, req.EventType, req.Payload)
if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ev})
}

func (h *Handler) ListEvents(c *gin.Context) {
	business := c.Query("business")
	eventType := c.Query("type")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code":400,"message":"invalid limit"})
		return
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.Svc.ListEvents(c.Request.Context(), business, eventType, limit, offset)
if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) StreamEvents(c *gin.Context) {
	business := c.Query("business")
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
			c.JSON(http.StatusInternalServerError, gin.H{"code":500,"message":"streaming not supported"})
			return
		}
	for ev := range ch {
		fmt.Fprintf(c.Writer, "data: {\"id\":%d,\"actor\":\"%s\",\"event_type\":\"%s\",\"created_at\":\"%s\"}\n\n",
			ev.ID, ev.Actor, ev.EventType, ev.CreatedAt.Format(time.RFC3339))
		flusher.Flush()
	}
}
