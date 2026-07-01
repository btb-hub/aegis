package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type mockSavedViewRepo struct {
	views   map[uuid.UUID]db.SavedView
	nextErr error
}

func (m *mockSavedViewRepo) ListSavedViewsForUser(ctx context.Context, userID uuid.UUID) ([]db.SavedView, error) {
	if m.nextErr != nil {
		return nil, m.nextErr
	}
	items := make([]db.SavedView, 0)
	for _, view := range m.views {
		if view.OwnerID == userID || view.Shared {
			items = append(items, view)
		}
	}
	return items, nil
}

func (m *mockSavedViewRepo) GetSavedView(ctx context.Context, id uuid.UUID) (db.SavedView, error) {
	if m.nextErr != nil {
		return db.SavedView{}, m.nextErr
	}
	view, ok := m.views[id]
	if !ok {
		return db.SavedView{}, pgx.ErrNoRows
	}
	return view, nil
}

func (m *mockSavedViewRepo) CreateSavedView(ctx context.Context, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error) {
	if m.nextErr != nil {
		return db.SavedView{}, m.nextErr
	}
	view := db.SavedView{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		Filter:    filter,
		Shared:    shared,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if m.views == nil {
		m.views = map[uuid.UUID]db.SavedView{}
	}
	m.views[view.ID] = view
	return view, nil
}

func (m *mockSavedViewRepo) UpdateSavedView(ctx context.Context, id, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error) {
	if m.nextErr != nil {
		return db.SavedView{}, m.nextErr
	}
	view, ok := m.views[id]
	if !ok || view.OwnerID != ownerID {
		return db.SavedView{}, pgx.ErrNoRows
	}
	view.Name = name
	view.Filter = filter
	view.Shared = shared
	m.views[id] = view
	return view, nil
}

func (m *mockSavedViewRepo) DeleteSavedView(ctx context.Context, id, ownerID uuid.UUID) error {
	if m.nextErr != nil {
		return m.nextErr
	}
	view, ok := m.views[id]
	if !ok || view.OwnerID != ownerID {
		return pgx.ErrNoRows
	}
	delete(m.views, id)
	return nil
}

func TestSavedViewCreateAndList(t *testing.T) {
	ownerID := uuid.New()
	repo := &mockSavedViewRepo{views: map[uuid.UUID]db.SavedView{}}
	svc := NewSavedViewService(repo)

	filter, _ := json.Marshal(map[string]any{"severity": "critical"})
	view, err := svc.Create(context.Background(), ownerID, "Critical only", filter, true)
	require.NoError(t, err)
	require.Equal(t, "Critical only", view.Name)
	require.True(t, view.Shared)

	views, err := svc.List(context.Background(), ownerID)
	require.NoError(t, err)
	require.Len(t, views, 1)
}

func TestSavedViewGetShared(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	viewID := uuid.New()
	repo := &mockSavedViewRepo{views: map[uuid.UUID]db.SavedView{
		viewID: {ID: viewID, OwnerID: ownerID, Name: "Shared", Filter: []byte(`{}`), Shared: true},
	}}
	svc := NewSavedViewService(repo)

	view, err := svc.Get(context.Background(), otherID, viewID)
	require.NoError(t, err)
	require.Equal(t, viewID, view.ID)
}

func TestSavedViewGetPrivateDenied(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	viewID := uuid.New()
	repo := &mockSavedViewRepo{views: map[uuid.UUID]db.SavedView{
		viewID: {ID: viewID, OwnerID: ownerID, Name: "Private", Filter: []byte(`{}`), Shared: false},
	}}
	svc := NewSavedViewService(repo)

	_, err := svc.Get(context.Background(), otherID, viewID)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestSavedViewValidation(t *testing.T) {
	svc := NewSavedViewService(&mockSavedViewRepo{})
	_, err := svc.Create(context.Background(), uuid.New(), " ", []byte(`{}`), false)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestSavedViewDelete(t *testing.T) {
	ownerID := uuid.New()
	viewID := uuid.New()
	repo := &mockSavedViewRepo{views: map[uuid.UUID]db.SavedView{
		viewID: {ID: viewID, OwnerID: ownerID, Name: "Mine", Filter: []byte(`{}`)},
	}}
	svc := NewSavedViewService(repo)
	require.NoError(t, svc.Delete(context.Background(), ownerID, viewID))
	_, err := svc.Get(context.Background(), ownerID, viewID)
	require.Error(t, err)
}

func TestSavedViewRepoError(t *testing.T) {
	svc := NewSavedViewService(&mockSavedViewRepo{nextErr: errors.New("db down")})
	_, err := svc.List(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestSavedViewUpdate(t *testing.T) {
	ownerID := uuid.New()
	viewID := uuid.New()
	repo := &mockSavedViewRepo{views: map[uuid.UUID]db.SavedView{
		viewID: {ID: viewID, OwnerID: ownerID, Name: "Mine", Filter: []byte(`{}`)},
	}}
	svc := NewSavedViewService(repo)
	filter, _ := json.Marshal(map[string]any{"severity": "warning"})
	view, err := svc.Update(context.Background(), ownerID, viewID, "Updated", filter, true)
	require.NoError(t, err)
	require.Equal(t, "Updated", view.Name)
	require.True(t, view.Shared)
}

func TestSavedViewInvalidFilter(t *testing.T) {
	svc := NewSavedViewService(&mockSavedViewRepo{})
	_, err := svc.Create(context.Background(), uuid.New(), "Bad", []byte(`not-json`), false)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestMapSavedViewNotFound(t *testing.T) {
	err := mapSavedViewError(pgx.ErrNoRows)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestSavedViewJSONEmptyFilter(t *testing.T) {
	out := SavedViewJSON(db.SavedView{ID: uuid.New(), OwnerID: uuid.New(), Filter: []byte(`invalid`)})
	filter, ok := out["filter"].(map[string]any)
	require.True(t, ok)
	require.Empty(t, filter)
}

func TestAnalyticsJSONEmpty(t *testing.T) {
	out := AnalyticsJSON(db.AlertAnalytics{})
	require.NotNil(t, out["by_severity"])
	require.NotNil(t, out["by_status"])
	require.Empty(t, out["top_labels"])
}

func TestSavedViewCreateEmptyFilter(t *testing.T) {
	svc := NewSavedViewService(&mockSavedViewRepo{})
	_, err := svc.Create(context.Background(), uuid.New(), "Empty", nil, false)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestSavedViewJSON(t *testing.T) {
	view := db.SavedView{
		ID:      uuid.New(),
		OwnerID: uuid.New(),
		Name:    "View",
		Filter:  []byte(`{"severity":"critical"}`),
		Shared:  true,
	}
	out := SavedViewJSON(view)
	require.Equal(t, "View", out["name"])
	filter, ok := out["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "critical", filter["severity"])
}
