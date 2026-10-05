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

func NewWorker(log *slog.Logger, store JobStore, alert *AlertProcessor, materialise *MaterialiseProcessor, escalate *EscalateProcessor, handoffNotify *HandoffNotifyProcessor, notifyIncident *NotifyIncidentProcessor, publishOnCall *PublishOnCallProcessor) *Worker {
	if log == nil {
		log = slog.Default()
	}
	handlers := map[string]Handler{
		"process_alert":      alert,
		"materialise_oncall": materialise,
		"escalate_incident":  escalate,
		"notify_handoff":     handoffNotify,
		"notify_incident":    notifyIncident,
	}
	if publishOnCall != nil {
		handlers["publish_oncall"] = publishOnCall
	}
	return &Worker{
		log:      log,
		store:    store,
		handlers: handlers,
	}
}

func (w *Worker) Register(kind string, handler Handler) { w.handlers[kind] = handler }

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
	if leased, ok := w.store.(interface {
		BeginJob(context.Context, Job) (context.Context, func())
	}); ok {
		var stop func()
		ctx, stop = leased.BeginJob(ctx, job)
		defer stop()
	}
	result := handler.Handle(ctx, job)
	if fenced, ok := w.store.(interface {
		FinishClaimedJob(context.Context, Job, error) error
	}); ok {
		return fenced.FinishClaimedJob(ctx, job, result)
	}
	if err := result; err != nil {
		if retry, ok := w.store.(interface {
			RetryJob(context.Context, string, error) error
		}); ok && (job.Kind == "sync_jira" || job.Kind == "express_reply" || job.Kind == "notify_incident" || job.Kind == "notify_handoff" || job.Kind == "escalate_incident" || job.Kind == "publish_oncall") {
			return retry.RetryJob(ctx, job.ID, err)
		}
		return w.store.FailJob(ctx, job.ID, err.Error())
	}
	return w.store.CompleteJob(ctx, job.ID)
}
