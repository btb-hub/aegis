package service

import "github.com/aegis/aegis/pkg/apperrors"

func ErrInvalidOAuthState() *apperrors.Error {
	return apperrors.Validation("invalid oauth state", nil)
}

func ErrInvalidBody() *apperrors.Error {
	return apperrors.Validation("invalid request body", nil)
}
