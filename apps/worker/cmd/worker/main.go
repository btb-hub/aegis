package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aegis/aegis/apps/worker/internal/processor"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	adapter := &storeAdapter{store: store}
	worker := processor.NewWorker(nil, adapter, processor.NewAlertProcessor(nil))

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if err := worker.RunOnce(ctx); err != nil {
				log.Printf("worker error: %v", err)
			}
		}
	}
}

type storeAdapter struct {
	store *db.Store
}

func (s *storeAdapter) ClaimNextJob(ctx context.Context) (bool, processor.Job, error) {
	job, err := s.store.ClaimNextJob(ctx)
	if err == pgx.ErrNoRows {
		return false, processor.Job{}, nil
	}
	if err != nil {
		return false, processor.Job{}, err
	}
	return true, processor.Job{ID: job.ID.String(), Kind: job.Kind, Payload: json.RawMessage(job.Payload)}, nil
}

func (s *storeAdapter) CompleteJob(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.store.CompleteJob(ctx, uid)
}

func (s *storeAdapter) FailJob(ctx context.Context, id, message string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.store.FailJob(ctx, uid, message)
}
