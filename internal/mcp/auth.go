package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Identity is the authenticated principal for an MCP tool call.
type Identity struct {
	UserID       int64
	Email        string
	Role         string
	BusinessID   int64
	BusinessCode string
	IsAPIKey     bool
	// BearerToken is the raw JWT from Authorization (soft-sync export).
	BearerToken  string
}

func (id *Identity) HasUser() bool { return id != nil && id.UserID > 0 }

func (h *Hub) identityFromRequest(ctx context.Context, req *mcpsdk.CallToolRequest) (*Identity, error) {
	var hdr http.Header
	if req != nil && req.Extra != nil {
		hdr = req.Extra.Header
	}
	if hdr == nil {
		hdr = http.Header{}
	}

	auth := hdr.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		id, err := h.parseJWT(tokenStr)
		if err == nil {
			id.BearerToken = tokenStr
			return id, nil
		}
		// fall through to API key
	}

	if id, ok := h.tryAPIKey(ctx, hdr); ok {
		return id, nil
	}

	if strings.HasPrefix(auth, "Bearer ") {
		return nil, errors.New("invalid or expired token; re-authenticate via OAuth or hub_login")
	}
	return nil, errors.New("missing Authorization Bearer token (or X-API-Key + X-Business-Code)")
}

func (h *Hub) parseJWT(tokenStr string) (*Identity, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	var userID int64
	switch v := claims["uid"].(type) {
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	case int:
		userID = int64(v)
	}
	if userID <= 0 {
		return nil, errors.New("invalid user token (re-login required)")
	}
	id := &Identity{UserID: userID}
	if sub, ok := claims["sub"].(string); ok {
		id.Email = sub
	}
	if role, ok := claims["role"].(string); ok {
		id.Role = role
	}
	return id, nil
}

func (h *Hub) tryAPIKey(ctx context.Context, hdr http.Header) (*Identity, bool) {
	key := hdr.Get("X-API-Key")
	businessCode := hdr.Get("X-Business-Code")
	if key == "" || businessCode == "" {
		return nil, false
	}
	// Prefer resolved short code / alias so keys keep working after migration
	bizIDResolved, canonical, _, rerr := h.Svc.ResolveBusinessCode(ctx, businessCode)
	lookupCode := businessCode
	if rerr == nil {
		lookupCode = canonical
	}
	hash := sha256Hex(key)
	var bizID int64
	err := h.Svc.Pool.QueryRow(ctx,
		`SELECT k.business_id FROM hub.hub_api_keys k
		 JOIN hub.hub_businesses b ON b.id = k.business_id
		 WHERE b.code = $1 AND k.key_hash = $2 AND k.revoked_at IS NULL`,
		lookupCode, hash,
	).Scan(&bizID)
	if err != nil && bizIDResolved > 0 {
		err = h.Svc.Pool.QueryRow(ctx,
			`SELECT k.business_id FROM hub.hub_api_keys k
			 WHERE k.business_id = $1 AND k.key_hash = $2 AND k.revoked_at IS NULL`,
			bizIDResolved, hash,
		).Scan(&bizID)
	}
	if err != nil {
		err = h.Svc.Pool.QueryRow(ctx,
			`SELECT k.business_id FROM hub.hub_api_keys k
			 JOIN hub.hub_businesses b ON b.id = k.business_id
			 WHERE b.code = $1 AND k.key_hash = $2`,
			lookupCode, hash,
		).Scan(&bizID)
		if err != nil {
			return nil, false
		}
	}
	if canonical == "" {
		canonical = lookupCode
	}
	return &Identity{
		BusinessID:   bizID,
		BusinessCode: canonical,
		IsAPIKey:     true,
		Role:         "api_key",
	}, true
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (h *Hub) requireUser(ctx context.Context, req *mcpsdk.CallToolRequest) (*Identity, error) {
	id, err := h.identityFromRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if !id.HasUser() {
		return nil, errors.New("login required (JWT user token)")
	}
	return id, nil
}

// requireMembership ensures the caller may access business_code (JWT member or matching API key).
// business_code may be short code, 90d alias, or "{slug}-{code}".
func (h *Hub) requireMembership(ctx context.Context, req *mcpsdk.CallToolRequest, code string) (bizID int64, role string, id *Identity, err error) {
	if code == "" {
		return 0, "", nil, errors.New("business_code required")
	}
	id, err = h.identityFromRequest(ctx, req)
	if err != nil {
		return 0, "", nil, err
	}

	keyBizID := id.BusinessID // preserve API-key binding before resolve
	var canonical string
	bizID, canonical, _, err = h.Svc.ResolveBusinessCode(ctx, code)
	if err != nil {
		return 0, "", nil, err
	}

	if id.IsAPIKey {
		if keyBizID != bizID {
			return 0, "", nil, errors.New("api key is not for this business")
		}
		id.BusinessID = bizID
		id.BusinessCode = canonical
		return bizID, "api_key", id, nil
	}

	if !id.HasUser() {
		return 0, "", nil, errors.New("login required")
	}
	err = h.Svc.Pool.QueryRow(ctx,
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		id.UserID, bizID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", nil, errors.New("you are not a member of this business")
		}
		return 0, "", nil, err
	}
	id.BusinessID = bizID
	id.BusinessCode = canonical
	return bizID, role, id, nil
}

func (h *Hub) requireAdmin(ctx context.Context, req *mcpsdk.CallToolRequest, code string) (bizID int64, id *Identity, err error) {
	bizID, role, id, err := h.requireMembership(ctx, req, code)
	if err != nil {
		return 0, nil, err
	}
	if role != "admin" && role != "owner" {
		return 0, nil, errors.New("only admins can perform this action")
	}
	return bizID, id, nil
}
