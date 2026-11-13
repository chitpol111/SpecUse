package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		log.Fatal("PORT must be an integer between 1 and 65535")
	}
	origins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	appEnv := os.Getenv("APP_ENV")
	fixturePath := os.Getenv("LOCAL_CATALOG_FILE")
	app, err := loadApplication()
	if err != nil {
		log.Fatal(err)
	}
	address := ":" + port
	if appEnv == "local" {
		// Local development data must not be served on a public interface.
		address = "127.0.0.1:" + port
	}
	server := &http.Server{
		Addr:              address,
		Handler:           handler.WithCORS(handler.WithMembers(handler.NewAPI(catalog.NewService(app.catalog), build.NewService(app.builds)), app.members), origins, "GET, POST, PATCH, DELETE"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatal(err)
	}
	if fixturePath != "" {
		log.Print("LOCAL DEVELOPMENT: using synthetic catalog fixture; not Supabase data")
	}
	log.Printf("API listening on %s", server.Addr)
	if err := serve(ctx, server, listener, 15*time.Second); err != nil {
		log.Fatal(err)
	}
}
