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
			if !m.tryAPIKey(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing token"})
			}
			return
		}
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			if !m.tryAPIKey(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			}
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid claims"})
			return
		}
		if uid, ok := claims["uid"]; ok {
			c.Set("user_id", int64(uid.(float64)))
		}
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
	hash := sha256Hex(key)
	var bizID int64
	err := m.Pool.QueryRow(c.Request.Context(),
		`SELECT k.business_id FROM hub.hub_api_keys k
		 JOIN hub.hub_businesses b ON b.id = k.business_id
		 WHERE b.code = $1 AND k.key_hash = $2`,
		businessCode, hash,
	).Scan(&bizID)
	if err != nil {
		return false
	}
	c.Set("business_id", bizID)
	c.Set("business_code", businessCode)
	return true
}
