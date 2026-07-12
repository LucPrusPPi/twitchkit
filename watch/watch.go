package watch

import (
	"context"
	"time"

	"github.com/LucPrusPPi/twitchkit/client"
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
func Loop(ctx context.Context, c *client.Client, t Target, interval time.Duration) error {
	if interval <= 0 {
		interval = 55 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := c.SendWatch(t.ChannelLogin, t.ChannelID, t.BroadcastID, t.UserID, t.GameName, t.GameID); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.SendWatch(t.ChannelLogin, t.ChannelID, t.BroadcastID, t.UserID, t.GameName, t.GameID); err != nil {
				return err
			}
		}
	}
}
