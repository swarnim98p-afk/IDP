package main

import (
	"context"
	"internal-developer-portal/portal-api/internal/projects"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var store projects.Store
	if mode := env("PROJECT_STORE", "file"); mode != "file" && mode != "dynamodb" {
		log.Fatal("PROJECT_STORE must be file or dynamodb")
	}
	if env("PROJECT_STORE", "file") == "dynamodb" {
		db, err := projects.NewDynamoStore(ctx, os.Getenv("DYNAMODB_TABLE"), os.Getenv("AWS_REGION"), os.Getenv("DYNAMODB_ENDPOINT"))
		if err != nil {
			log.Fatal(err)
		}
		if os.Getenv("DYNAMODB_ENDPOINT") != "" && os.Getenv("DYNAMODB_CREATE_TABLE") == "true" {
			if err = db.EnsureLocalTable(ctx); err != nil {
				log.Fatal(err)
			}
		}
		store = db
	} else {
		file, err := projects.NewFileStore(env("PROJECT_DATA_FILE", "../../.run/projects.json"))
		if err != nil {
			log.Fatal(err)
		}
		store = file
	}
	cancel()
	router, err := projects.NewRouter(store)
	if err != nil {
		log.Fatal(err)
	}
	server := projects.Server(env("PROJECT_LISTEN_ADDR", "127.0.0.1:8082"), router)
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
	log.Printf("Project service listening on %s (storage: %s)", server.Addr, store.Mode())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
