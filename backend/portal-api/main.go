package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"internal-developer-portal/portal-api/internal/gateway"
	"internal-developer-portal/portal-api/internal/projects"
)

// setupRouter provides the project API for in-process regression tests.
func setupRouter() (*gin.Engine, error) { return projects.NewRouter(projects.NewMemoryStore()) }

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func main() {
	router, err := gateway.NewRouter(env("PROJECT_SERVICE_URL", "http://127.0.0.1:8082"), env("FRONTEND_DIR", "../../frontend/dist"))
	if err != nil {
		log.Fatal(err)
	}
	server := projects.Server(env("PORTAL_LISTEN_ADDR", "127.0.0.1:8080"), router)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	log.Printf("Portal API listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
