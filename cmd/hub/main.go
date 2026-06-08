package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stifer/agent-hub/internal/config"
	"github.com/stifer/agent-hub/internal/hub/handler"
	"github.com/stifer/agent-hub/internal/hub/repository"
	"github.com/stifer/agent-hub/internal/hub/service"
	"github.com/stifer/agent-hub/internal/middleware"
	"github.com/stifer/agent-hub/internal/server"
)

func main() {
	fmt.Println("agent-hub starting...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, pool, err := repository.NewClient(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}
	defer client.Close()
	defer pool.Close()

	svc := service.New(client, pool)
	mw := middleware.New(pool)
	h := handler.New(svc, cfg.JWTSecret)
	srv := server.New(mw, h, cfg)

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	go func() {
		fmt.Printf("agent-hub listening on %s\n", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Background stale lock cleanup
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := svc.CleanupExpiredLocks(ctx)
				if err != nil {
					fmt.Printf("lock cleanup error: %v\n", err)
				} else if count > 0 {
					fmt.Printf("cleaned %d expired locks\n", count)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpSrv.Shutdown(shutdownCtx)
	fmt.Println("agent-hub stopped")
}
