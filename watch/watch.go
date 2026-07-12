package watch

import (
	"context"
	"time"

	"github.com/LucPrusPPi/twitchkit/client"
	"github.com/LucPrusPPi/twitchkit/retry"
)

// Target identifies what to farm.
type Target struct {
	ChannelLogin string
	ChannelID    string
	BroadcastID  string
	UserID       string
	GameName     string
	GameID       string
}

// FromStream builds a Target from StreamInfo + authenticated user id.
func FromStream(s client.StreamInfo, userID string) Target {
	return Target{
		ChannelLogin: s.UserLogin,
		ChannelID:    s.UserID,
		BroadcastID:  s.ID,
		UserID:       userID,
		GameName:     s.GameName,
		GameID:       s.GameID,
	}
}

// Loop sends minute-watched events until ctx is cancelled.
// interval defaults to 55s when <= 0 (Twitch client cadence).
// Transient send failures are retried a few times; permanent errors stop the loop.
func Loop(ctx context.Context, c *client.Client, t Target, interval time.Duration) error {
	if interval <= 0 {
		interval = 55 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	send := func() error {
		return retry.Do(ctx, retry.Config{Attempts: 3, Base: 400 * time.Millisecond}, func() error {
			return c.SendWatch(t.ChannelLogin, t.ChannelID, t.BroadcastID, t.UserID, t.GameName, t.GameID)
		})
	}

	if err := send(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := send(); err != nil {
				return err
			}
		}
	}
}
