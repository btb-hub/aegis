package main

import (
	"context"
	"encoding/json"
	"github.com/aegis/aegis/pkg/integrations"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aegis/aegis/apps/worker/internal/processor"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
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

	if err := loadI18n(); err != nil {
		log.Fatalf("i18n: %v", err)
	}

	store := db.NewStore(pool)
	adapter := &storeAdapter{store: store}
	materialise := processor.NewMaterialiseProcessor(nil, store)
	alert := processor.NewAlertProcessor(nil, store, cfg.IncidentDedupWindow, cfg.EscalationTimeout)
	escalate := processor.NewEscalateProcessor(nil, store, cfg.PublicURL)
	handoffNotify := processor.NewHandoffNotifyProcessor(nil, store, cfg.PublicURL)
	notifyIncident := processor.NewNotifyIncidentProcessor(nil, store, cfg.PublicURL)
	publishOnCall := processor.NewPublishOnCallProcessor(nil, store, cfg.PublicURL)
	worker := processor.NewWorker(nil, adapter, alert, materialise, escalate, handoffNotify, notifyIncident, publishOnCall)

	worker.Register("sync_jira", processor.NewJiraSyncProcessor(store, cfg.PublicURL))
	worker.Register("express_reply", processor.NewExpressReplyProcessor(store))

	go enqueueNightlyMaterialise(ctx, store)
	go enqueueDailyPublishOnCall(ctx, store)
	go enqueueOnCallRotationTicker(ctx, store)

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
	return true, processor.Job{ID: job.ID.String(), Kind: job.Kind, Attempt: job.Attempts, Payload: json.RawMessage(job.Payload)}, nil
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

func enqueueNightlyMaterialise(ctx context.Context, store *db.Store) {
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, time.UTC)
		time.Sleep(time.Until(next))
		_, _ = store.EnqueueJob(ctx, "materialise_oncall", []byte(`{}`), time.Now())
	}
}

func enqueueDailyPublishOnCall(ctx context.Context, store *db.Store) {
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, time.UTC)
		time.Sleep(time.Until(next))
		_, _ = store.EnqueueJob(ctx, "publish_oncall", []byte(`{}`), time.Now())
	}
}

func enqueueOnCallRotationTicker(ctx context.Context, store *db.Store) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := processor.EnqueueOnCallRotationPublishes(ctx, store, time.Now().UTC()); err != nil {
				log.Printf("publish_oncall rotation enqueue: %v", err)
			}
		}
	}
}

func loadI18n() error {
	dir := filepath.Join("pkg", "i18n", "messages")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = filepath.Join("..", "..", "pkg", "i18n", "messages")
	}
	return i18n.LoadMessages(dir)
}

func (s *storeAdapter) RetryJob(ctx context.Context, id string, cause error) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	if !integrations.RetryableError(cause) {
		return s.store.FailJob(ctx, uid, cause.Error())
	}
	return s.store.RescheduleJob(ctx, uid, cause.Error())
}

func (s *storeAdapter) BeginJob(ctx context.Context, job processor.Job) (context.Context, func()) {
	child, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-child.Done():
				return
			case <-ticker.C:
				id, err := uuid.Parse(job.ID)
				if err != nil {
					cancel()
					return
				}
				alive, err := s.store.RenewClaimedJob(child, id, job.Attempt)
				if err != nil || !alive {
					cancel()
					return
				}
			}
		}
	}()
	return child, func() { cancel(); <-done }
}
func (s *storeAdapter) FinishClaimedJob(ctx context.Context, job processor.Job, result error) error {
	id, err := uuid.Parse(job.ID)
	if err != nil {
		return err
	}
	retry := result != nil && integrations.RetryableError(result) && (job.Kind == "sync_jira" || job.Kind == "express_reply" || job.Kind == "notify_incident" || job.Kind == "notify_handoff" || job.Kind == "escalate_incident" || job.Kind == "publish_oncall")
	message := ""
	if result != nil {
		message = result.Error()
	}
	return s.store.FinishClaimedJob(ctx, id, job.Attempt, message, result == nil, retry)
}
