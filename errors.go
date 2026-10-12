package pidgely

import (
	"errors"
	"fmt"
	"net/http"
)

// Error names returned by the API, plus the SDK-side ErrApplication.
const (
	ErrValidation                   = "validation_error"
	ErrMissingRequiredField         = "missing_required_field"
	ErrNotFound                     = "not_found"
	ErrMissingAPIKey                = "missing_api_key"
	ErrInvalidAPIKey                = "invalid_api_key"
	ErrRestrictedAPIKey             = "restricted_api_key"
	ErrRateLimitExceeded            = "rate_limit_exceeded"
	ErrDailyQuotaExceeded           = "daily_quota_exceeded"
	ErrMonthlyQuotaExceeded         = "monthly_quota_exceeded"
	ErrInvalidIdempotencyKey        = "invalid_idempotency_key"
	ErrInvalidIdempotentRequest     = "invalid_idempotent_request"
	ErrConcurrentIdempotentRequests = "concurrent_idempotent_requests"
	ErrMethodNotAllowed             = "method_not_allowed"
	ErrInternalServer               = "internal_server_error"
	// ErrApplication is used for failures without an API response: network
	// errors, timeouts, cancelled contexts and unparsable responses.
	ErrApplication = "application_error"
)

// Error is returned by every client method that fails.
//
// API errors carry the HTTP StatusCode and the API's machine-readable Name and
// Message. Failures without a response (network error, timeout, cancelled
// context) have StatusCode 0, Name "application_error", and wrap the cause, so
// errors.Is(err, context.Canceled) and similar checks work.
type Error struct {
	// StatusCode is the HTTP status, or 0 when no response was received.
	StatusCode int
	// Name is the machine-readable error name, such as "validation_error".
	Name string
	// Message is a human-readable description.
	Message string
	// Headers are the response headers, or nil when no response was received.
	Headers http.Header
	// Err is the underlying cause for failures without a response.
	Err error
}

func (e *Error) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("pidgely: %s: %s", e.Name, e.Message)
	}
	return fmt.Sprintf("pidgely: %s (HTTP %d): %s", e.Name, e.StatusCode, e.Message)
}

// Unwrap returns the underlying cause, if any.
func (e *Error) Unwrap() error { return e.Err }

// AsError returns err as a *Error when it is one (or wraps one).
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsNotFound reports whether err is a 404 / not_found API error.
func IsNotFound(err error) bool {
	e, ok := AsError(err)
	return ok && (e.StatusCode == http.StatusNotFound || e.Name == ErrNotFound)
}

// IsRateLimited reports whether err is a 429 API error (rate limit or quota).
func IsRateLimited(err error) bool {
	e, ok := AsError(err)
	return ok && e.StatusCode == http.StatusTooManyRequests
}
