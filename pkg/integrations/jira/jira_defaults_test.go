package jira

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDefaultIssueType(t *testing.T) {
	provider := New(Config{BaseURL: "https://jira.example.com", Email: "a@b.com", APIToken: "x", ProjectKey: "OPS"})
	require.Equal(t, "Task", provider.cfg.IssueType)
}
