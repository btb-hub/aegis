package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type expressLinkMockRepo struct {
	code string
	user db.User
}

func (m *expressLinkMockRepo) CreateExpressLinkCode(_ context.Context, userID uuid.UUID, _ time.Duration) (string, error) {
	m.code = "ABC123"
	return m.code, nil
}

func (m *expressLinkMockRepo) RedeemExpressLinkCode(_ context.Context, code string, expressHuid uuid.UUID) (db.User, error) {
	if code != m.code {
		return db.User{}, fmt.Errorf("link code invalid or expired")
	}
	m.user.ExpressUserHuid = db.ExpressHuidToPg(expressHuid)
	return m.user, nil
}

func (m *expressLinkMockRepo) UpdateUserExpressHuid(_ context.Context, userID, expressHuid uuid.UUID) (db.User, error) {
	m.user.ID = userID
	m.user.ExpressUserHuid = db.ExpressHuidToPg(expressHuid)
	return m.user, nil
}


func TestExpressLinkServiceCreateCode(t *testing.T) {
	svc := NewExpressLinkService(&expressLinkMockRepo{})
	code, err := svc.CreateLinkCode(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, "ABC123", code)
}

func TestExpressLinkServiceBindExpressHuid(t *testing.T) {
	repo := &expressLinkMockRepo{}
	svc := NewExpressLinkService(repo)
	huid := uuid.MustParse("6fafda2c-6505-57a5-a088-25ea5d1d0364")
	user, err := svc.BindExpressHuid(context.Background(), uuid.New(), huid.String())
	require.NoError(t, err)
	require.True(t, user.ExpressUserHuid.Valid)
}
