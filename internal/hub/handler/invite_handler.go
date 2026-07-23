package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func genInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// acceptInviteToken validates a raw invite token, creates membership, marks invite accepted.
func (h *Handler) acceptInviteToken(c *gin.Context, userID int64, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return errors.New("token required")
	}
	th := hashToken(rawToken)

	var inviteID, businessID int64
	var email, role, status string
	var expiresAt time.Time
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT id, business_id, email, role, status, expires_at
		 FROM hub.hub_invites WHERE token_hash=$1`, th,
	).Scan(&inviteID, &businessID, &email, &role, &status, &expiresAt)
	if err != nil {
		return errors.New("invalid invite token")
	}
	if status != "pending" {
		return fmt.Errorf("invite is %s", status)
	}
	if time.Now().After(expiresAt) {
		_, _ = h.Svc.Pool.Exec(c.Request.Context(),
			`UPDATE hub.hub_invites SET status='expired', updated_at=now() WHERE id=$1`, inviteID)
		return errors.New("invite expired")
	}

	var userEmail string
	if err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT email FROM hub.hub_users WHERE id=$1", userID).Scan(&userEmail); err != nil {
		return errors.New("user not found")
	}
	if !strings.EqualFold(userEmail, email) {
		return errors.New("invite email does not match current user")
	}

	if role == "" {
		role = "member"
	}
	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at)
		 VALUES ($1, $2, $3, now()) ON CONFLICT DO NOTHING`,
		userID, businessID, role)
	if err != nil {
		return err
	}

	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		`UPDATE hub.hub_invites
		 SET status='accepted', accepted_by=$1, accepted_at=now(), updated_at=now()
		 WHERE id=$2`, userID, inviteID)
	if err != nil {
		return err
	}
	return nil
}

type acceptInviteReq struct {
	Token string `json:"token" binding:"required"`
}

func (h *Handler) AcceptInvite(c *gin.Context) {
	userIDRaw, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return
	}
	userID, ok := userIDRaw.(int64)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid user"})
		return
	}
	var req acceptInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.acceptInviteToken(c, userID, req.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "invite accepted"}})
}

func (h *Handler) ListInvites(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

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
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only admins can list invites"})
		return
	}

	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT id, email, role, status, expires_at, created_at
		 FROM hub.hub_invites
		 WHERE business_id=$1 AND status='pending'
		 ORDER BY created_at DESC`, biz.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()

	type item struct {
		ID        int64  `json:"id"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expires_at"`
		CreatedAt string `json:"created_at"`
	}
	var list []item
	for rows.Next() {
		var it item
		var exp, created time.Time
		if err := rows.Scan(&it.ID, &it.Email, &it.Role, &it.Status, &exp, &created); err != nil {
			continue
		}
		it.ExpiresAt = exp.Format(time.RFC3339)
		it.CreatedAt = created.Format(time.RFC3339)
		list = append(list, it)
	}
	if list == nil {
		list = []item{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) RevokeInvite(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")
	inviteID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid invite id"})
		return
	}

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

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
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only admins can revoke invites"})
		return
	}

	tag, err := h.Svc.Pool.Exec(c.Request.Context(),
		`UPDATE hub.hub_invites SET status='revoked', updated_at=now()
		 WHERE id=$1 AND business_id=$2 AND status='pending'`,
		inviteID, biz.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "pending invite not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "invite revoked"}})
}

func (h *Handler) ListMembers(c *gin.Context) {
	code := c.Param("code")
	userID, _ := c.Get("user_id")

	biz, err := h.Svc.GetBusinessByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	var role string
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		userID, biz.ID,
	).Scan(&role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "you are not a member of this business"})
		return
	}

	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT u.id, u.email, COALESCE(u.name,''), m.role, m.created_at
		 FROM hub.hub_memberships m
		 JOIN hub.hub_users u ON u.id = m.user_id
		 WHERE m.business_id=$1
		 ORDER BY m.created_at ASC`, biz.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()

	type member struct {
		UserID    int64  `json:"user_id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Role      string `json:"role"`
		JoinedAt  string `json:"joined_at"`
	}
	var list []member
	for rows.Next() {
		var m member
		var joined time.Time
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.Role, &joined); err != nil {
			continue
		}
		m.JoinedAt = joined.Format(time.RFC3339)
		list = append(list, m)
	}
	if list == nil {
		list = []member{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
