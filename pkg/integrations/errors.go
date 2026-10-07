package integrations

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type HTTPError struct {
	Provider, Operation, Message string
	Status                       int
	RetryAfter                   time.Duration
}

func ParseRetryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && time.Until(at) > 0 {
		return time.Until(at)
	}
	return 0
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s %s: status %d: %s", e.Provider, e.Operation, e.Status, e.Message)
}
func (e *HTTPError) Retryable() bool { return e.Status == 408 || e.Status == 429 || e.Status >= 500 }

// RetryableError preserves transient failures when independent destinations fail together.
func RetryableError(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			if RetryableError(child) {
				return true
			}
		}
		return false
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return RetryableError(wrapped.Unwrap())
	}
	if e, ok := err.(interface{ Retryable() bool }); ok {
		return e.Retryable()
	}
	return true
}
