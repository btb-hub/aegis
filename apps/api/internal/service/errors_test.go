package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrInvalidBody(t *testing.T) {
	err := ErrInvalidBody()
	require.Equal(t, "VALIDATION_ERROR", err.Code)
}

func TestErrInvalidOAuthState(t *testing.T) {
	err := ErrInvalidOAuthState()
	require.Equal(t, "VALIDATION_ERROR", err.Code)
}
