package integrations

import (
	"fmt"
)

type HTTPError struct {
	Provider, Operation, Message string
	Status                       int
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
