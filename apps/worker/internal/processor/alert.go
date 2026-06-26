package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

type Job struct {
	ID      string
	Kind    string
	Payload json.RawMessage
}

type JobStore interface {
	ClaimNextJob(ctx context.Context) (claimed bool, job Job, err error)
	CompleteJob(ctx context.Context, id string) error
	FailJob(ctx context.Context, id, message string) error
}

type AlertProcessor struct {
	log *slog.Logger
}

func NewAlertProcessor(log *slog.Logger) *AlertProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &AlertProcessor{log: log}
}

func (p *AlertProcessor) Handle(ctx context.Context, job Job) error {
	var payload map[string]any
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	alertID, _ := payload["alert_id"].(string)
	p.log.Info("process_alert stub", "job_id", job.ID, "alert_id", alertID)
	return nil
}

func KindFor(job Job) string {
	return job.Kind
}
