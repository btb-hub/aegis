package jira

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFromJSONSuccess(t *testing.T) {
	provider, err := NewFromJSON([]byte(`{"base_url":"https://jira.example.com","email":"a@b.com","api_token":"x","project_key":"OPS"}`))
	require.NoError(t, err)
	require.Equal(t, "jira", provider.Kind())
}

func TestNewFromJSONInvalidJSON(t *testing.T) {
	_, err := NewFromJSON([]byte(`{`))
	require.Error(t, err)
}
