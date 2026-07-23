package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// RequireMembership ensures the caller may access the business identified by code.
//
// Paths:
//   - JWT user (user_id set): must have a hub_memberships row
//   - API key (business_id set by middleware, no user_id): key must match this business
//
// `code` may be a short code, old alias (90d grace), or "{slug}-{code}" path segment.
func (h *Handler) RequireMembership(c *gin.Context, code string) (bizID int64, role string, ok bool) {
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "business code required"})
		return 0, "", false
	}

	bizID, canonical, _, err := h.Svc.ResolveBusinessCode(c.Request.Context(), code)
	if err != nil {
		if strings.HasPrefix(err.Error(), "business not found") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
			return 0, "", false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return 0, "", false
	}
	c.Header("X-Business-Code-Canonical", canonical)

	// Machine path: API key middleware set business_id without user_id
	if uid, hasUser := c.Get("user_id"); !hasUser || uid == nil {
		if mid, hasBiz := c.Get("business_id"); hasBiz {
			if id, castOK := mid.(int64); castOK && id == bizID {
				return bizID, "api_key", true
			}
		}
		if bc, has := c.Get("business_code"); has {
			if s, castOK := bc.(string); castOK {
				_, can2, _, e2 := h.Svc.ResolveBusinessCode(c.Request.Context(), s)
				if e2 == nil && can2 == canonical {
					return bizID, "api_key", true
				}
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return 0, "", false
	}

	userID, okUID := c.Get("user_id")
	if !okUID {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return 0, "", false
	}

	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		userID, bizID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "you are not a member of this business"})
			return 0, "", false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return 0, "", false
	}
	return bizID, role, true
}
