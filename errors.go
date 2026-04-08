package cloud

import (
	"errors"
	"net/http"
)

// APIError represents an error returned by the Simplifyd Cloud API.
type APIError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}

// IsNotFound reports whether err is an HTTP 404 response.
func IsNotFound(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}

// IsUnauthorized reports whether err is an HTTP 401 response.
func IsUnauthorized(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusUnauthorized
}

// IsForbidden reports whether err is an HTTP 403 response.
func IsForbidden(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusForbidden
}

// IsRateLimited reports whether err is an HTTP 429 response.
func IsRateLimited(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusTooManyRequests
}
