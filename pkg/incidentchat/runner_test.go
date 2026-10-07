package incidentchat

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type timeoutStore struct {
	Store
	status  string
	outcome []byte
	ready   bool
}

func (s *timeoutStore) ClaimChannelDelivery(context.Context) (db.ChannelDelivery, error) {
	return db.ChannelDelivery{Attempts: 5}, nil
}
func (s *timeoutStore) ClaimChatAckFeedback(context.Context) (db.ChatAckRequest, error) {
	return db.ChatAckRequest{Attempts: 5, Outcome: json.RawMessage(`{"code":"acknowledged"}`)}, nil
}
func (s *timeoutStore) GetIntegration(ctx context.Context, _ uuid.UUID) (db.Integration, error) {
	<-ctx.Done()
	return db.Integration{}, ctx.Err()
}
func (s *timeoutStore) FinishChannelDelivery(ctx context.Context, _ uuid.UUID, status, _, _ string, _ time.Time) error {
	s.status = status
	return ctx.Err()
}
func (s *timeoutStore) FinishChatAck(ctx context.Context, _ uuid.UUID, status, _ string, _ time.Time) error {
	s.status = status
	return ctx.Err()
}
func (s *timeoutStore) ClaimChatAckAction(context.Context) (db.ChatAckRequest, error) {
	return db.ChatAckRequest{Attempts: 5, Locale: "en", Provider: "slack"}, nil
}
func (s *timeoutStore) ProcessChatAck(context.Context, db.ChatAckRequest) (json.RawMessage, error) {
	return nil, errors.New("transaction failed")
}
func (s *timeoutStore) SaveChatAckOutcome(ctx context.Context, _ uuid.UUID, outcome []byte) error {
	s.outcome = outcome
	return ctx.Err()
}
func (s *timeoutStore) ReadyChatAckFeedback(ctx context.Context, _ uuid.UUID) error {
	s.ready = true
	return ctx.Err()
}
func (s *timeoutStore) GetUserBySlackID(context.Context, string) (db.User, error) {
	return db.User{Locale: "ru"}, nil
}

func TestTimeoutPersistsFinalAttemptWithFreshContext(t *testing.T) {
	for _, feedback := range []bool{false, true} {
		s := &timeoutStore{}
		r := New(s, "https://aegis.local")
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		var err error
		if feedback {
			err = r.DeliverFeedback(ctx)
		} else {
			err = r.DeliverChannel(ctx)
		}
		cancel()
		require.NoError(t, err)
		require.Equal(t, "failed", s.status)
	}
}

func TestExhaustedAcknowledgementQueuesHonestFailureFeedback(t *testing.T) {
	s := &timeoutStore{}
	r := New(s, "https://aegis.local")
	require.NoError(t, r.ProcessAck(context.Background()))
	require.True(t, s.ready)
	require.Contains(t, string(s.outcome), `"code":"not_confirmed"`)
	require.Contains(t, string(s.outcome), `"locale":"ru"`)
}

func TestProviderRetryAfterExtendsBackoffAndPermanentErrorsStop(t *testing.T) {
	before := time.Now()
	status, next := RetryDisposition(1, &integrations.HTTPError{Status: 429, RetryAfter: time.Minute})
	require.Equal(t, "pending", status)
	require.WithinDuration(t, before.Add(time.Minute), next, 100*time.Millisecond)
	status, _ = RetryDisposition(1, &integrations.HTTPError{Status: 403})
	require.Equal(t, "failed", status)
}

type exhaustedStore struct{ timeoutStore }

func (s *exhaustedStore) ClaimChannelDelivery(context.Context) (db.ChannelDelivery, error) {
	return db.ChannelDelivery{Attempts: 6}, nil
}
func (s *exhaustedStore) ClaimChatAckFeedback(context.Context) (db.ChatAckRequest, error) {
	return db.ChatAckRequest{Attempts: 6}, nil
}
func (s *exhaustedStore) ClaimChatAckAction(context.Context) (db.ChatAckRequest, error) {
	return db.ChatAckRequest{Attempts: 6, Provider: "slack"}, nil
}
func (s *exhaustedStore) GetIntegration(context.Context, uuid.UUID) (db.Integration, error) {
	panic("must not send a sixth time")
}
func (s *exhaustedStore) ProcessChatAck(context.Context, db.ChatAckRequest) (json.RawMessage, error) {
	panic("must not process a sixth time")
}
func TestExpiredFifthLeaseCannotSendOrProcessAgain(t *testing.T) {
	s := &exhaustedStore{}
	r := New(s, "")
	require.NoError(t, r.DeliverChannel(t.Context()))
	require.Equal(t, "failed", s.status)
	require.NoError(t, r.DeliverFeedback(t.Context()))
	require.Equal(t, "failed", s.status)
	require.NoError(t, r.ProcessAck(t.Context()))
	require.True(t, s.ready)
	require.Contains(t, string(s.outcome), "not_confirmed")
}
