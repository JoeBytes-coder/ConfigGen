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

	"github.com/getsentry/sentry-go"

	"configgen/internal/infrastructure/generators"
	"configgen/internal/infrastructure/storage"
	"configgen/internal/presentation"
	"configgen/internal/usecase"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":8080", "server address")
	flag.Parse()

	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDSN,
			TracesSampleRate: 0.01,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
		log.Println("Sentry initialized")
	}

	dataSource := os.Getenv("CONFIGGEN_DB")
	if dataSource == "" {
		dataSource = "db/configgen.db"
	}

	store, err := storage.NewSQLiteStore(dataSource)
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	configGen := usecase.NewConfigGenerator(generators.NewAdapter(), store)

	srv := presentation.NewServer(configGen)

	// Run server in goroutine
	go func() {
		if err := srv.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server exited: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with 5s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
