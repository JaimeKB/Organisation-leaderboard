// Command server runs the GitHub org contributor leaderboard web app.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaimekb/org-leaderboard/internal/config"
	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/httpserver"
	"github.com/jaimekb/org-leaderboard/internal/wizard"
)

const labelConfigPath = "configs/labels.yaml"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	labels, err := config.LoadLabels(labelConfigPath)
	if err != nil {
		return err
	}

	store := wizard.NewMemoryStore()

	sweepCtx, stopSweep := context.WithCancel(context.Background())
	defer stopSweep()
	store.StartSweeper(sweepCtx, 10*time.Minute, 2*time.Hour)

	server, err := httpserver.New(httpserver.Options{
		Client:       githubapi.NewClient(cfg.GitHubPAT),
		Store:        store,
		LabelConfig:  labels,
		Concurrency:  6,
		WindowMonths: 3,
	})
	if err != nil {
		return err
	}

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on http://localhost:%s", cfg.Port)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		log.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
	return nil
}
