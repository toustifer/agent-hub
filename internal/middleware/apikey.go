package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Middleware struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Middleware {
	return &Middleware{Pool: pool}
}

func (m *Middleware) APIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing api key"})
			return
		}
		businessCode := c.GetHeader("X-Business-Code")
		if businessCode == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing business code"})
			return
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid api key or business"})
			return
		}
		c.Set("business_id", bizID)
		c.Set("business_code", businessCode)
		c.Next()
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (m *Middleware) APIKeyOrNone() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		businessCode := c.GetHeader("X-Business-Code")
		if key == "" || businessCode == "" {
			c.Next()
			return
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
			c.Next()
			return
		}
		c.Set("business_id", bizID)
		c.Set("business_code", businessCode)
		c.Next()
	}
}
