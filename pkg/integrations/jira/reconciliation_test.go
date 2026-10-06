package jira

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindTicketVersionAndAmbiguity(t *testing.T) {
	for _, deployment := range []string{"server_dc", "cloud"} {
		t.Run(deployment, func(t *testing.T) {
			count := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := "/rest/api/2/search"
				if deployment == "cloud" {
					path = "/rest/api/3/search/jql"
				}
				require.Equal(t, path, r.URL.Path)
				require.Contains(t, r.URL.Query().Get("jql"), "aegis-incident-identifier")
				count++
				switch count {
				case 1:
					_, _ = w.Write([]byte(`{"issues":[]}`))
				case 2:
					_, _ = w.Write([]byte(`{"issues":[{"key":"OPS-1"}]}`))
				case 3:
					_, _ = w.Write([]byte(`{"issues":[{"key":"OPS-1"},{"key":"OPS-2"}]}`))
				case 4:
					_, _ = w.Write([]byte(`bad`))
				default:
					w.WriteHeader(403)
				}
			}))
			defer s.Close()
			p := New(Config{BaseURL: s.URL, Deployment: deployment, ProjectKey: "OPS"})
			key, err := p.FindTicket(t.Context(), "identifier")
			require.NoError(t, err)
			require.Empty(t, key)
			key, err = p.FindTicket(t.Context(), "identifier")
			require.NoError(t, err)
			require.Equal(t, "OPS-1", key)
			_, err = p.FindTicket(t.Context(), "identifier")
			require.ErrorContains(t, err, "multiple")
			_, err = p.FindTicket(t.Context(), "identifier")
			require.Error(t, err)
			_, err = p.FindTicket(t.Context(), "identifier")
			require.Error(t, err)
		})
	}
}
func TestFindCommentPagination(t *testing.T) {
	count := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count == 1 {
			require.Equal(t, "0", r.URL.Query().Get("startAt"))
			_ = json.NewEncoder(w).Encode(map[string]any{"total": 2, "comments": []any{map[string]string{"id": "1", "body": "other"}}})
		} else {
			require.Equal(t, "1", r.URL.Query().Get("startAt"))
			_, _ = w.Write([]byte(`{"total":2,"comments":[{"id":"2","body":{"text":"Aegis event: marker"}}]}`))
		}
	}))
	defer s.Close()
	key, err := New(Config{BaseURL: s.URL}).FindComment(t.Context(), "OPS-1", "Aegis event: marker")
	require.NoError(t, err)
	require.Equal(t, "2", key)
}
func TestCommentsInvalidResponses(t *testing.T) {
	for _, response := range []string{`bad`, `{"id":""}`} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
		p := New(Config{BaseURL: s.URL})
		_, err := p.AddComment(t.Context(), "OPS-1", "note")
		require.Error(t, err)
		_, err = p.FindComment(t.Context(), "OPS-1", "marker")
		if response == "bad" {
			require.Error(t, err)
		}
		s.Close()
	}
}
