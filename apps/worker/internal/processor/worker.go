package processor

import (
	"context"
	"fmt"
	"log/slog"
)

type Handler interface {
	Handle(ctx context.Context, job Job) error
}

type Worker struct {
	log      *slog.Logger
	store    JobStore
	handlers map[string]Handler
}

func NewWorker(log *slog.Logger, store JobStore, alert *AlertProcessor) *Worker {
	if log == nil {
		log = slog.Default()
	}
	return &Worker{
		log:   log,
		store: store,
		handlers: map[string]Handler{
			"process_alert": alert,
		},
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	claimed, job, err := w.store.ClaimNextJob(ctx)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	handler, ok := w.handlers[job.Kind]
	if !ok {
		return w.store.FailJob(ctx, job.ID, fmt.Sprintf("unknown job kind: %s", job.Kind))
	}
	if err := handler.Handle(ctx, job); err != nil {
		return w.store.FailJob(ctx, job.ID, err.Error())
	}
	return w.store.CompleteJob(ctx, job.ID)
}
