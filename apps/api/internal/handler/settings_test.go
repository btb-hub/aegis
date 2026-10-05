package handler

import (
	"bytes"
	"context"
	"errors"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type settingsHandlerFixture struct{ err error }

func (f *settingsHandlerFixture) GetPublicationSettings(context.Context) (db.PublicationSettings, error) {
	return db.PublicationSettings{Time: "03:00", Timezone: "UTC"}, f.err
}
func (f *settingsHandlerFixture) UpdatePublicationSettings(_ context.Context, clock, zone string, _ time.Time) (db.PublicationSettings, error) {
	return db.PublicationSettings{Time: clock, Timezone: zone}, f.err
}
func TestPublicationSettingsAdministratorAPI(t *testing.T) {
	r, repo := setupPhase2Router(t)
	auth := service.NewAuthService(&config.Config{SessionTTL: time.Hour}, repo, repo, &authMockOIDC{})
	f := &settingsHandlerFixture{}
	NewSettingsHandler(service.NewSettingsService(f), auth).Register(r)
	admin := seedAdmin(t, r, repo)
	for _, spec := range []struct {
		method, body string
		want         int
		cookie       bool
		fail         bool
	}{{"GET", "", 200, true, false}, {"PATCH", `{"time":"08:00","timezone":"Europe/Moscow"}`, 200, true, false}, {"PATCH", `{`, 400, true, false}, {"PATCH", `{"time":"bad","timezone":"UTC"}`, 400, true, false}, {"GET", "", 401, false, false}, {"GET", "", 500, true, true}, {"PATCH", `{"time":"08:00","timezone":"UTC"}`, 500, true, true}} {
		f.err = nil
		if spec.fail {
			f.err = errors.New("db")
		}
		req := httptest.NewRequest(spec.method, "/api/v1/settings/oncall-publication", bytes.NewBufferString(spec.body))
		req.Header.Set("Content-Type", "application/json")
		if spec.cookie {
			req.AddCookie(admin)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, spec.want, w.Code, w.Body.String())
	}
	for id, user := range repo.users {
		user.Role = "member"
		repo.users[id] = user
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/oncall-publication", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}
