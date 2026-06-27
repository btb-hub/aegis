package routing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMatchLabelsInvalidJSON(t *testing.T) {
	_, err := ParseMatchLabels([]byte(`{`))
	require.Error(t, err)
}

func TestParseMatchLabelsEmpty(t *testing.T) {
	labels, err := ParseMatchLabels(nil)
	require.NoError(t, err)
	require.Empty(t, labels)
}
