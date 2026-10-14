package cloud

import (
	"encoding/json"
	"errors"
	"net/http"
)

// APIError represents an error returned by the Simplifyd Cloud API.
type APIError struct {
	Message string `json:"message"`
	// Code is the API's machine-readable error code, such as
	// "ERR_PRICE_CHANGED", when it sends one.
	Code       string `json:"code,omitempty"`
	StatusCode int    `json:"-"`

	// body is the raw response, for errors that carry more than a message.
	body []byte
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

// IsConflict reports whether err is an HTTP 409 response — the request was
// understood but refused because of the resource's current state. Deleting a
// project's last environment is the common case.
func IsConflict(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusConflict
}

// IsRateLimited reports whether err is an HTTP 429 response.
func IsRateLimited(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusTooManyRequests
}

// PriceConfirmationError is a domain purchase or renewal refused because its
// price was not confirmed: ConfirmTotal was left out where it is required (a
// workspace token must always send it), or it was not the current price.
// Nothing was charged. Total is what it costs now; confirm it and try again.
//
// It is also an *APIError with status 409, so IsConflict reports true.
type PriceConfirmationError struct {
	*APIError
	// Total is the current price in centiKobo, VAT included.
	Total int64
	// Changed is true when a ConfirmTotal was sent and did not match, and
	// false when none was sent.
	Changed bool
}

func (e *PriceConfirmationError) Unwrap() error { return e.APIError }

// AsPriceConfirmation reports whether err is a price that must be confirmed,
// and returns it.
func AsPriceConfirmation(err error) (*PriceConfirmationError, bool) {
	var e *PriceConfirmationError
	return e, errors.As(err, &e)
}

// priceConfirmation turns the API's answer to an unconfirmed price into a
// *PriceConfirmationError, and leaves any other error as it is.
func priceConfirmation(err error) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		return err
	}
	if apiErr.Code != "ERR_PRICE_CHANGED" && apiErr.Code != "ERR_PRICE_CONFIRMATION_REQUIRED" {
		return err
	}
	var body struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(apiErr.body, &body)
	return &PriceConfirmationError{APIError: apiErr, Total: body.Total, Changed: apiErr.Code == "ERR_PRICE_CHANGED"}
}
