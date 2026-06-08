package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type createPlaybookReq struct {
	Category string   `json:"category"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
	WorkerID string   `json:"worker_id"`
}

func (h *Handler) CreatePlaybook(c *gin.Context) {
	var req createPlaybookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	bizCode := c.GetString("business_code")
	pb, err := h.Svc.CreatePlaybook(c.Request.Context(), bizCode, req.Category, req.Title, req.Content, req.Tags, req.WorkerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pb})
}

func (h *Handler) SearchPlaybooks(c *gin.Context) {
	q := c.Query("q")
	business := c.Query("business")
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.Svc.SearchPlaybooks(c.Request.Context(), q, business, category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) GetPlaybookByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}
	pb, err := h.Svc.GetPlaybookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pb})
}
