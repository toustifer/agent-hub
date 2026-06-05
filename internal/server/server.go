// Package server 组装 gin engine。
package server

import (
	"github.com/gin-gonic/gin"
)

// New 返回配置好的 gin.Engine。
// Phase 3 完成：注册路由 + 中间件。
func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	return r
}
