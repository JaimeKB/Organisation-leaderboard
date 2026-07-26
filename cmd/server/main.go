// Command server starts the local web UI for browsing GitHub org commit
// leaderboards. It exits with a graceful HTTP shutdown on SIGINT/SIGTERM,
// which the Docker/Makefile flow relies on for clean teardown.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaimekershawbrown/organisation-leaderboard/internal/config"
	"github.com/jaimekershawbrown/organisation-leaderboard/internal/githubapi"
	"github.com/jaimekershawbrown/organisation-leaderboard/internal/server"
)

const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	if cfg.GitHubPAT == "" {
		log.Print("warning: " + config.MissingPATMessage)
	}

	gh := githubapi.New(ctx, cfg.GitHubPAT)
	srv, err := server.New(":"+cfg.Port, gh, cfg.GitHubPAT != "")
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
