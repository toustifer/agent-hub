package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type commentItem struct {
	ID          int64     `json:"id"`
	AuthorEmail string    `json:"author_email"`
	Body        string    `json:"body"`
	Decision    string    `json:"decision,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type createCommentReq struct {
	Body     string `json:"body" binding:"required"`
	Decision string `json:"decision"` // accept, reject, changes_requested, or empty for regular comment
}

func (h *Handler) CreateComment(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	reqID := c.Param("id")
	st := stampActorFromGin(c, "member", "", "")

	var req createCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Validate decision value if provided
	if req.Decision != "" {
		valid := map[string]bool{"accept": true, "reject": true, "changes_requested": true}
		if !valid[req.Decision] {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid decision: " + req.Decision})
			return
		}
	}
	var exists bool
	if err := h.Svc.Pool.QueryRow(c.Request.Context(), `SELECT EXISTS (SELECT 1 FROM hub.hub_requirements WHERE id=$1 AND business_id=$2)`, reqID, bizID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "requirement not found"})
		return
	}

	decision := strings.TrimSpace(req.Decision)
	var id int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO hub.hub_requirement_comments (business_id, requirement_id, author_user_id, author_email, body, decision, created_at)
		 VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),now()) RETURNING id`,
		bizID, reqID, userIDPtr(st.UserID), st.Email, req.Body, decision,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id}})
}

func (h *Handler) ListComments(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	reqID := c.Param("id")
	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT id, COALESCE(NULLIF(author_email,''),''),
		        body, COALESCE(NULLIF(decision,''),''), created_at
		 FROM hub.hub_requirement_comments
		 WHERE business_id=$1 AND requirement_id=$2
		 ORDER BY created_at ASC`,
		bizID, reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	var list []commentItem
	for rows.Next() {
		var item commentItem
		if err := rows.Scan(&item.ID, &item.AuthorEmail, &item.Body, &item.Decision, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	if list == nil {
		list = []commentItem{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
