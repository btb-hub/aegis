package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_URL")), "/")
	if err := config.SeedDevAllowed(publicURL); err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	teamID, err := store.EnsureDemoRouting(ctx)
	if err != nil {
		return fmt.Errorf("ensure demo routing: %w", err)
	}
	log.Printf("routing rule team=platform → team %s (Platform)", teamID)

	replayed, err := store.ReplayFailedProcessAlertJobs(ctx)
	if err != nil {
		return fmt.Errorf("replay failed alert jobs: %w", err)
	}
	if replayed > 0 {
		log.Printf("re-queued %d failed process_alert job(s) — ensure make dev-worker is running", replayed)
	}

	alertCount, err := store.CountAllAlerts(ctx)
	if err != nil {
		return err
	}
	log.Printf("alerts in database: %d (view at /alerts when signed in)", alertCount)
	log.Printf("incidents appear at /incidents after the worker processes re-queued jobs")
	return nil
}
