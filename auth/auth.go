package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const validateURL = "https://id.twitch.tv/oauth2/validate"

// TokenInfo is returned by OAuth validate.
type TokenInfo struct {
	Login  string
	UserID string
}

// Error classifies validate failures.
type Error struct {
	Invalid   bool
	Transient bool
	Message   string
}

func (e *Error) Error() string {
	if e.Invalid {
		return "auth token invalid"
	}
	return e.Message
}

// Normalize strips an optional "oauth:" prefix and trims whitespace.
func Normalize(token string) string {
	t := strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(t), "oauth:") {
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
		return TokenInfo{}, &Error{Transient: true, Message: err.Error()}
	}
	req.Header.Set("Authorization", "OAuth "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return TokenInfo{}, &Error{Transient: true, Message: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return TokenInfo{}, &Error{Invalid: true, Message: "unauthorized"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenInfo{}, &Error{
			Transient: true,
			Message:   fmt.Sprintf("validate status %d: %s", resp.StatusCode, truncate(string(body), 200)),
		}
	}
	var raw struct {
		Login  string `json:"login"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return TokenInfo{}, &Error{Transient: true, Message: err.Error()}
	}
	if raw.Login == "" || raw.UserID == "" {
		return TokenInfo{}, &Error{Transient: true, Message: "validate missing login/user_id"}
	}
	return TokenInfo{Login: raw.Login, UserID: raw.UserID}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
