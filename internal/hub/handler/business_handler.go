package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type createBusinessReq struct {
	Code        string `json:"code" binding:"required,min=2,max=64"`
	Name        string `json:"name" binding:"required,min=1,max=128"`
	RepoURL     string `json:"repo_url"`
	Description string `json:"description"`
}

func genAPIKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *Handler) CreateBusiness(c *gin.Context) {
	var req createBusinessReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	biz, err := h.Svc.CreateBusiness(c.Request.Context(), req.Code, req.Name, req.RepoURL, 0, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	if uid, ok := userID.(int64); ok && uid > 0 {
		_, err := h.Svc.Pool.Exec(c.Request.Context(),
			"INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at) VALUES ($1, $2, 'admin', now()) ON CONFLICT DO NOTHING",
			uid, biz.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to add membership"})
			return
		}
	}

	apiKey, err := genAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate api key"})
		return
	}
	hash := sha256.Sum256([]byte(apiKey))
	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_api_keys (business_id, key_hash, created_at) VALUES ($1, $2, now()) ON CONFLICT DO NOTHING",
		biz.ID, hex.EncodeToString(hash[:]))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to create api key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"business": biz, "api_key": apiKey}})
}

func (h *Handler) ListBusinesses(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.Svc.ListBusinesses(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

type updateBusinessReq struct{ Status string `json:"status"` }

func (h *Handler) UpdateBusiness(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}
	var req updateBusinessReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.Svc.UpdateBusinessStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func (h *Handler) GetBusinessByCode(c *gin.Context) {
	code := c.Param("code")
	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": biz})
}

type inviteMemberReq struct {
	Email string `json:"email" binding:"required"`
	Role  string `json:"role"`
}

func (h *Handler) InviteMember(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return
	}
	var req inviteMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	// Check if the inviter is a member
	var role string
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		userID, biz.ID,
	).Scan(&role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "you are not a member of this business"})
		return
	}

	// Find the target user by email
	var targetUserID int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id FROM hub.hub_users WHERE email=$1", req.Email,
	).Scan(&targetUserID)
	if err != nil {
		// User doesn't exist yet — they'll be added when they register and accept
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"message":    "Invitation recorded. User will be added when they register and join.",
			"invite_url": "https://hub.stifer.xyz/team/" + code,
			"email":      req.Email,
		}})
		return
	}

	// Add membership
	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at) VALUES ($1, $2, $3, now()) ON CONFLICT DO NOTHING",
		targetUserID, biz.ID, req.Role,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message":    "Member invited successfully",
		"email":      req.Email,
		"role":       req.Role,
		"invite_url": "https://hub.stifer.xyz/team/" + code,
	}})
}

type generateCodeReq struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) GenerateBusinessCode(c *gin.Context) {
	var req generateCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Sanitize name to a valid code: lowercase, replace spaces/special chars with -
	code := sanitizeCode(req.Name)
	if code == "" {
		code = "project"
	}
	base := code

	// Try the base code first, then append suffix if taken
	for i := 0; i < 100; i++ {
		var exists bool
		err := h.Svc.Pool.QueryRow(c.Request.Context(),
			"SELECT EXISTS(SELECT 1 FROM hub.hub_businesses WHERE code=$1)", code,
		).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if !exists {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"business_code": code, "display_name": req.Name}})
			return
		}
		// Append random 4-char suffix
		code = fmt.Sprintf("%s-%s", base, randomCode(4))
	}

	c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "unable to generate unique code after 100 attempts"})
}

func sanitizeCode(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'A' && c <= 'Z' {
				c += 32 // lowercase
			}
			result = append(result, c)
		} else if c == ' ' || c == '-' || c == '_' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	// Trim trailing dashes
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return string(result)
}
