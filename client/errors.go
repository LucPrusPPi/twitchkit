package client

import (
	"errors"
	"fmt"
	"net/http"
)

// StatusError is an HTTP failure from Twitch endpoints.
type StatusError struct {
	Op     string // "gql", "helix", "spade", "integrity", "bootstrap"
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("twitch %s: http %d", e.Op, e.Status)
	}
	return fmt.Sprintf("twitch %s: http %d: %s", e.Op, e.Status, e.Body)
}

// Transient reports whether the caller may retry (5xx / 429 / some 408).
func (e *StatusError) Transient() bool {
	return e.Status == http.StatusTooManyRequests ||
		e.Status == http.StatusRequestTimeout ||
		e.Status >= 500
}

// IsTransient reports whether err looks safe to retry.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	var se *StatusError
	if errors.As(err, &se) {
		return se.Transient()
	}
	// Network errors from net/http are usually transient.
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return false
}
