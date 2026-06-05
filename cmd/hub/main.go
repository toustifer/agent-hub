// Command hub 是 agent-hub 的入口。
//
// 启动流程：
//  1. 加载配置（viper + .env）
//  2. 初始化 logger（zap）
//  3. 连接 PostgreSQL（schema=hub）
//  4. 连接 Redis
//  5. 跑 migration
//  6. 启动后台任务（锁清理、Worker 离线检测）
//  7. 启动 HTTP 服务（gin）
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("agent-hub starting...")

	// 占位实现：完整启动流程在 Phase 3 接入
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown signal received")
		cancel()
	}()

	<-ctx.Done()
	log.Println("agent-hub stopped")
}
