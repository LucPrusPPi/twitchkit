// Package retry runs short backoff loops for transient Twitch failures.
package retry

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LucPrusPPi/twitchkit/auth"
	"github.com/LucPrusPPi/twitchkit/client"
)

// Config controls Do. Zero Attempts means 3. Zero Base means 500ms.
type Config struct {
	Attempts int
	Base     time.Duration
}

func (c Config) withDefaults() Config {
	if c.Attempts <= 0 {
		c.Attempts = 3
	}
	if c.Base <= 0 {
		c.Base = 500 * time.Millisecond
	}
	return c
}

// Do calls fn until it succeeds, Attempts are exhausted, ctx is cancelled,
// or the error is not transient (for example an invalid token).
func Do(ctx context.Context, cfg Config, fn func() error) error {
	cfg = cfg.withDefaults()
	var last error
	delay := cfg.Base
	for i := 0; i < cfg.Attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if err == nil {
			return nil
		}
		last = err
		if auth.IsInvalid(err) || !isRetryable(err) {
			return err
		}
		if i == cfg.Attempts-1 {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay *= 2
	}
	return last
}

func isRetryable(err error) bool {
	if client.IsTransient(err) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	type temporary interface{ Temporary() bool }
	var t temporary
	if errors.As(err, &t) && t.Temporary() {
		return true
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"connection reset",
		"connection refused",
		"i/o timeout",
		"tls handshake",
		"unexpected eof",
		"eof",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}
