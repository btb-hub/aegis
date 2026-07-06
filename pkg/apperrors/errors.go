package apperrors

import "net/http"

type Error struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	StatusCode int            `json:"-"`
}

func (e *Error) Error() string {
	return e.Message
}

func New(code, message string, status int) *Error {
	return &Error{Code: code, Message: message, StatusCode: status}
}

func Validation(message string, details map[string]any) *Error {
	return &Error{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		Details:    details,
		StatusCode: http.StatusBadRequest,
	}
}

func Unauthorized(message string) *Error {
	return New("UNAUTHORIZED", message, http.StatusUnauthorized)
}

func Forbidden(message string) *Error {
	return New("FORBIDDEN", message, http.StatusForbidden)
}

func InvalidLocale() *Error {
	return New("INVALID_LOCALE", "locale must be en or ru", http.StatusBadRequest)
}

func InvalidWebhookSecret() *Error {
	return New("INVALID_WEBHOOK_SECRET", "webhook secret is invalid", http.StatusUnauthorized)
}

func NotFound(resource string) *Error {
	return New("NOT_FOUND", resource+" not found", http.StatusNotFound)
}

func Conflict(message string) *Error {
	return New("CONFLICT", message, http.StatusConflict)
}

func ConflictWithDetails(message string, details map[string]any) *Error {
	return &Error{
		Code:       "CONFLICT",
		Message:    message,
		Details:    details,
		StatusCode: http.StatusConflict,
	}
}
