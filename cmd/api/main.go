package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cupcake_store/backend/internal/auth"
	"cupcake_store/backend/internal/config"
	"cupcake_store/backend/internal/db"
	"cupcake_store/backend/internal/handlers"
	"cupcake_store/backend/internal/httpx"
)

func main() {
	if err := run(); err != nil {
		slog.Error("encerrando", "erro", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	conn, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	h := handlers.New(db.NewStore(conn), auth.NewTokens(cfg.JWTSecret))
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpx.CORS(cfg.CORSOrigins, h.Routes()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("falha ao encerrar", "erro", err)
		}
	}()

	slog.Info("API ouvindo", "porta", cfg.Port)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-shutdownDone // espera as requisições em andamento terminarem
	return nil
}
