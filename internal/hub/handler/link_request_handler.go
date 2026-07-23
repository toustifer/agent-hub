package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type createLinkReq struct {
	DeviceInfo string `json:"device_info"`
}

type reviewLinkReq struct {
	Action string `json:"action" binding:"required"`
}

func (h *Handler) CreateLinkRequest(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return
	}

	var req createLinkReq
	c.ShouldBindJSON(&req)

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	// Check if already a member
	var role string
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		userID, biz.ID,
	).Scan(&role)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "already a member"})
		return
	}

	// Check for existing pending request
	var existingID int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id FROM hub.hub_link_requests WHERE business_id=$1 AND user_id=$2 AND status='pending'",
		biz.ID, userID,
	).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "link request already pending"})
		return
	}

	// Create the link request
	var id int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO hub.hub_link_requests (business_id, user_id, device_info, status)
		 VALUES ($1, $2, $3, 'pending') RETURNING id`,
		biz.ID, userID, req.DeviceInfo,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"id":            id,
		"business_code": code,
		"status":        "pending",
		"message":       "Link request created. An admin must approve it.",
	}})
}

func (h *Handler) ListLinkRequests(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	// Only admin|owner may list pending link requests
	var role string
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		userID, biz.ID,
	).Scan(&role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "you are not a member of this business"})
		return
	}
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only admins can list link requests"})
		return
	}

	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT r.id, r.user_id, u.email, u.name, r.device_info, r.status, r.created_at
		 FROM hub.hub_link_requests r
		 JOIN hub.hub_users u ON u.id = r.user_id
		 WHERE r.business_id = $1 AND r.status = 'pending'
		 ORDER BY r.created_at DESC`, biz.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()

	type item struct {
		ID         int64  `json:"id"`
		UserID     int64  `json:"user_id"`
		Email      string `json:"email"`
		Name       string `json:"name"`
		DeviceInfo string `json:"device_info"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
	}

	var list []item
	for rows.Next() {
		var it item
		var deviceInfo, name *string
		var createdAt time.Time
		if err := rows.Scan(&it.ID, &it.UserID, &it.Email, &name, &deviceInfo, &it.Status, &createdAt); err != nil {
			continue
		}
		if name != nil {
			it.Name = *name
		}
		if deviceInfo != nil {
			it.DeviceInfo = *deviceInfo
		}
		it.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		list = append(list, it)
	}
	if list == nil {
		list = []item{}
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) ReviewLinkRequest(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")
	reqID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request id"})
		return
	}

	var req reviewLinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.Action != "approve" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "action must be 'approve' or 'reject'"})
		return
	}

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	// Verify caller is admin
	var role string
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		userID, biz.ID,
	).Scan(&role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "you are not a member of this business"})
		return
	}
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only admins can review link requests"})
		return
	}

	// Fetch the link request
	var requestUserID int64
	var status string
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT user_id, status FROM hub.hub_link_requests WHERE id=$1 AND business_id=$2",
		reqID, biz.ID,
	).Scan(&requestUserID, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "link request not found"})
		return
	}
	if status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "link request is already " + status})
		return
	}

	uid := userID.(int64)
	if requestUserID == uid {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "cannot review your own link request"})
		return
	}

	if req.Action == "approve" {
		// Add to memberships
		_, err = h.Svc.Pool.Exec(c.Request.Context(),
			"INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at) VALUES ($1, $2, 'member', now()) ON CONFLICT DO NOTHING",
			requestUserID, biz.ID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	// Update link request status
	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		"UPDATE hub.hub_link_requests SET status=$1, reviewed_by=$2, reviewed_at=now(), updated_at=now() WHERE id=$3",
		req.Action+"d", uid, reqID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "link request " + req.Action + "d"}})
}
