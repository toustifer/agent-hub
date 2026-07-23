package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func (m *Middleware) JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		tokenStr := ""
		if auth != "" && strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}
		// SSE EventSource can't set headers — fallback to query param
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			if m.tryAPIKey(c) {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing token"})
			return
		}
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			if m.tryAPIKey(c) {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid claims"})
			return
		}
		var userID int64
		if uid, ok := claims["uid"]; ok {
			switch v := uid.(type) {
			case float64:
				userID = int64(v)
			case int64:
				userID = v
			case int:
				userID = int64(v)
			}
		}
		// Reject device-placeholder / unbound tokens (uid=0)
		if userID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid user token (re-login required)"})
			return
		}
		c.Set("user_id", userID)
		if sub, ok := claims["sub"]; ok {
			c.Set("email", sub)
		}
		if role, ok := claims["role"]; ok {
			c.Set("role", role)
		}
		c.Next()
	}
}

func (m *Middleware) tryAPIKey(c *gin.Context) bool {
	key := c.GetHeader("X-API-Key")
	businessCode := c.GetHeader("X-Business-Code")
	if key == "" || businessCode == "" {
		return false
	}
	// Resolve short code / alias / slug-code path
	keyCode := businessCode
	if i := lastShortCode(businessCode); i != "" {
		keyCode = i
	}
	hash := sha256Hex(key)
	var bizID int64
	var canonical string
	err := m.Pool.QueryRow(c.Request.Context(),
		`SELECT k.business_id, b.code FROM hub.hub_api_keys k
		 JOIN hub.hub_businesses b ON b.id = k.business_id
		 WHERE b.code = $1 AND k.key_hash = $2 AND k.revoked_at IS NULL`,
		keyCode, hash,
	).Scan(&bizID, &canonical)
	if err != nil {
		// alias grace
		err = m.Pool.QueryRow(c.Request.Context(),
			`SELECT k.business_id, b.code FROM hub.hub_api_keys k
			 JOIN hub.hub_businesses b ON b.id = k.business_id
			 JOIN hub.hub_business_code_aliases a ON a.business_id = b.id
			 WHERE a.old_code = $1 AND a.expires_at > now() AND k.key_hash = $2 AND k.revoked_at IS NULL`,
			keyCode, hash,
		).Scan(&bizID, &canonical)
		if err != nil {
			// fallback without revoked_at / aliases for older schemas
			err = m.Pool.QueryRow(c.Request.Context(),
				`SELECT k.business_id, b.code FROM hub.hub_api_keys k
				 JOIN hub.hub_businesses b ON b.id = k.business_id
				 WHERE b.code = $1 AND k.key_hash = $2`,
				keyCode, hash,
			).Scan(&bizID, &canonical)
			if err != nil {
				return false
			}
		}
	}
	c.Set("business_id", bizID)
	c.Set("business_code", canonical)
	c.Header("X-Business-Code-Canonical", canonical)
	// API key path does NOT set user_id — worker scope only
	return true
}

// lastShortCode returns trailing 4-char token if present (slug-code form).
func lastShortCode(input string) string {
	input = strings.TrimSpace(input)
	parts := strings.Split(input, "-")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if len(last) == 4 {
			ok := true
			for _, r := range last {
				if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
					ok = false
					break
				}
			}
			if ok {
				return last
			}
		}
	}
	return input
}
