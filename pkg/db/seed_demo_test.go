package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMatchLabels(t *testing.T) {
	labels, err := parseMatchLabels([]byte(`{"team":"platform"}`))
	require.NoError(t, err)
	require.Equal(t, "platform", labels["team"])
}

func TestParseMatchLabelsEmpty(t *testing.T) {
	labels, err := parseMatchLabels([]byte(`{}`))
	require.NoError(t, err)
	require.Empty(t, labels)
}
