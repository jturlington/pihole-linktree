package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pihole-linktree/pkg/cache"
	"pihole-linktree/pkg/config"
	"pihole-linktree/pkg/pihole"
	"pihole-linktree/pkg/server"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Starting DNS record viewer service")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	piholeClient := pihole.NewClient(cfg.PiholeHost, cfg.AuthToken)
	cache := cache.New(piholeClient, cfg.CacheRefreshInterval, cfg.BaseDomain)
	server := server.New(cache, cfg)

	// Start cache updater
	cache.Start(ctx)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		sig := <-sigChan

		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server listening on :8080")
	if err := server.Start(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Printf("Server shutdown complete")
}
