package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"configgen/config"
	"configgen/internal/infrastructure/storage"
	"configgen/internal/presentation"
	"configgen/internal/usecase"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", "", "server address")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	_ = config.LoadDotEnv(".env")
	listenAddr := config.GetListenAddr(addr)

	dsn := os.Getenv("CONFIGGEN_DB")
	if dsn == "" {
		dsn = "db/configgen.db"
	}
	store, err := storage.NewSQLiteStore(dsn)
	if err != nil {
		slog.Error("failed to create store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	server := presentation.NewServer(usecase.NewGeneratorService(store))
	httpServer := &http.Server{
		Addr:              listenAddr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server exited", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
