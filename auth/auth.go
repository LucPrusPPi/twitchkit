// Package auth validates Twitch OAuth access tokens.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const validateURL = "https://id.twitch.tv/oauth2/validate"

// ErrInvalidToken means Twitch rejected the access token (401).
// Callers should stop farming that account until the user re-auths.
var ErrInvalidToken = errors.New("auth: invalid token")

// TokenInfo is returned by a successful validate call.
type TokenInfo struct {
	Login  string
	UserID string
}

// Error is a validate failure with a stable classification.
type Error struct {
	// Cause is ErrInvalidToken for 401, or a descriptive error for soft failures.
	Cause error
	// Status is the HTTP status when the response was received (0 if none).
	Status int
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return "auth: unknown error"
	}
	if e.Status > 0 {
		return fmt.Sprintf("%v (http %d)", e.Cause, e.Status)
	}
	return e.Cause.Error()
}

func (e *Error) Unwrap() error { return e.Cause }

// Invalid reports whether the token must be discarded.
func (e *Error) Invalid() bool {
	return errors.Is(e.Cause, ErrInvalidToken)
}

// Transient reports whether a retry may help (network / 5xx / malformed body).
func (e *Error) Transient() bool {
	return !e.Invalid()
}

// Normalize strips an optional "oauth:" prefix and surrounding whitespace.
func Normalize(token string) string {
	t := strings.TrimSpace(token)
	if len(t) >= 6 && strings.EqualFold(t[:6], "oauth:") {
		return strings.TrimSpace(t[6:])
	}
	return t
}

// Validate checks the token against id.twitch.tv.
func Validate(httpClient *http.Client, token string) (TokenInfo, error) {
	token = Normalize(token)
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	req, err := http.NewRequest(http.MethodGet, validateURL, nil)
	if err != nil {
		return TokenInfo{}, &Error{Cause: err}
	}
	req.Header.Set("Authorization", "OAuth "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return TokenInfo{}, &Error{Cause: err}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return TokenInfo{}, &Error{Cause: ErrInvalidToken, Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenInfo{}, &Error{
			Cause:  fmt.Errorf("auth: validate failed: %s", truncate(string(body), 200)),
			Status: resp.StatusCode,
		}
	}

	var raw struct {
		Login  string `json:"login"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return TokenInfo{}, &Error{Cause: fmt.Errorf("auth: decode validate: %w", err), Status: resp.StatusCode}
	}
	if raw.Login == "" || raw.UserID == "" {
		return TokenInfo{}, &Error{Cause: errors.New("auth: validate missing login/user_id"), Status: resp.StatusCode}
	}
	return TokenInfo{Login: raw.Login, UserID: raw.UserID}, nil
}

// IsInvalid is a convenience for errors.Is(err, ErrInvalidToken).
func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalidToken)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
