package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/llm-proxy/internal/auth"
	"github.com/llm-proxy/internal/backend"
	"github.com/llm-proxy/internal/config"
	"github.com/llm-proxy/internal/proxy"
	"github.com/llm-proxy/internal/router"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	factory, err := backend.NewFactory(cfg.Backends, cfg.APIKeys)
	if err != nil {
		log.Fatalf("Failed to create backend factory: %v", err)
	}

	r := router.New(cfg.Routes, cfg.DefaultRoute, factory)
	handler := proxy.NewHandler(r)

	authMiddleware := auth.NewMiddleware(cfg.Auth)
	authenticatedHandler := authMiddleware.Wrap(handler)

	server := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      authenticatedHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("Starting LLM Proxy on %s", cfg.Addr())
		log.Printf("Default backend: %s", cfg.DefaultRoute)
		log.Printf("Routes: %v", r.ListRoutes())

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
