package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/hub/service"
)

type createBusinessReq struct {
	// Code is ignored when present — server always assigns a 4-char short code.
	Code        string `json:"code"`
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

	userIDRaw, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return
	}
	uid, ok := userIDRaw.(int64)
	if !ok || uid <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid user"})
		return
	}

	// Always auto-generate short code (ignore client-supplied code).
	biz, err := h.Svc.CreateBusiness(c.Request.Context(), "", req.Name, req.RepoURL, uid, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at) VALUES ($1, $2, 'owner', now()) ON CONFLICT DO NOTHING",
		uid, biz.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to add ownership"})
		return
	}

	// Default: no API key. Human path is JWT/MCP login + membership.
	// Opt-in machine key: HUB_CREATE_API_KEY=1
	out := gin.H{
		"business": biz,
		"role":     "owner",
		"note":     "use Web login or MCP hub_login (JWT) to manage this team; API key is optional for CI/workers",
	}
	if os.Getenv("HUB_CREATE_API_KEY") == "1" {
		apiKey, err := genAPIKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate api key"})
			return
		}
		hash := sha256.Sum256([]byte(apiKey))
		_, err = h.Svc.Pool.Exec(c.Request.Context(),
			"INSERT INTO hub.hub_api_keys (business_id, key_hash, label, created_at) VALUES ($1, $2, 'default', now()) ON CONFLICT DO NOTHING",
			biz.ID, hex.EncodeToString(hash[:]))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to create api key"})
			return
		}
		out["api_key"] = apiKey
		out["note"] = "store api_key securely; it is shown only once (HUB_CREATE_API_KEY=1)"
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
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

type patchBusinessProfileReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// PatchBusinessProfile renames a team (name/description only). Code is immutable.
// Route: PATCH /v1/hub/businesses/:code/profile
func (h *Handler) PatchBusinessProfile(c *gin.Context) {
	code := c.Param("code")
	bizID, role, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only owners/admins can rename the team"})
		return
	}
	var req patchBusinessProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.Name == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name or description required"})
		return
	}
	if err := h.Svc.UpdateBusinessProfile(c.Request.Context(), bizID, req.Name, req.Description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	biz, err := h.Svc.GetBusinessByID(c.Request.Context(), bizID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"business":  biz,
		"team_path": service.BuildTeamPath(biz.Name, biz.Code),
	}})
}

func (h *Handler) GetBusinessByCode(c *gin.Context) {
	code := c.Param("code")
	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.Header("X-Business-Code-Canonical", biz.Code)
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

	bizID, role, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	if role != "admin" && role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only admins can invite members"})
		return
	}

	rawToken, err := genInviteToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate invite token"})
		return
	}
	th := hashToken(rawToken)

	// Always persist invite (registered or not). Accept still required.
	var inviteID int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO hub.hub_invites
		   (business_id, email, role, token_hash, invited_by, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', now() + interval '7 days')
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		bizID, req.Email, req.Role, th, userID,
	).Scan(&inviteID)
	if err != nil {
		// pending unique conflict: revoke old pending and re-insert
		_, _ = h.Svc.Pool.Exec(c.Request.Context(),
			`UPDATE hub.hub_invites SET status='revoked', updated_at=now()
			 WHERE business_id=$1 AND lower(email)=lower($2) AND status='pending'`,
			bizID, req.Email)
		err = h.Svc.Pool.QueryRow(c.Request.Context(),
			`INSERT INTO hub.hub_invites
			   (business_id, email, role, token_hash, invited_by, status, expires_at)
			 VALUES ($1, $2, $3, $4, $5, 'pending', now() + interval '7 days')
			 RETURNING id`,
			bizID, req.Email, req.Role, th, userID,
		).Scan(&inviteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	inviteURL := publicBaseURL() + "/invite/accept?token=" + rawToken
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message":    "Invitation created. Share the invite_url (token shown only once).",
		"email":      req.Email,
		"role":       req.Role,
		"invite_id":  inviteID,
		"invite_url": inviteURL,
		"expires_in": "7d",
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
