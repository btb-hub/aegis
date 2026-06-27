package processor

import (
	"context"
	"encoding/json"
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

func KindFor(job Job) string {
	return job.Kind
}
