package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LucPrusPPi/twitchkit/auth"
	"github.com/gorilla/websocket"
)

const (
	pubsubURL      = "wss://pubsub-edge.twitch.tv"
	pingInterval = 4 * time.Minute
	writeWait    = 10 * time.Second
)

// Event is a drops PubSub notification.
type Event interface {
	isEvent()
}

// ProgressEvent is drop-progress.
type ProgressEvent struct {
	DropID   string
	Current  float64
	Required float64
}

func (ProgressEvent) isEvent() {}

// ClaimEvent is drop-claim.
type ClaimEvent struct {
	DropInstanceID string
}

func (ClaimEvent) isEvent() {}

// Listen connects to PubSub and delivers drop events until ctx is done.
// Reconnects with exponential backoff on disconnect.
func Listen(ctx context.Context, token, userID string, out chan<- Event) error {
	token = auth.Normalize(token)
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := connectOnce(ctx, token, userID, out)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > time.Minute {
				backoff = time.Minute
			}
			continue
		}
		backoff = time.Second
	}
}

func connectOnce(ctx context.Context, token, userID string, out chan<- Event) error {
	d := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}
	conn, _, err := d.DialContext(ctx, pubsubURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	listen := map[string]any{
		"type":  "LISTEN",
		"nonce": fmt.Sprintf("%d", time.Now().UnixNano()),
		"data": map[string]any{
			"topics":     []string{"user-drop-events." + userID},
			"auth_token": token,
		},
	}
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(listen); err != nil {
		return err
	}

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	errCh := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			ev, ok := parseMessage(data)
			if !ok {
				var envelope struct {
					Type string `json:"type"`
				}
				_ = json.Unmarshal(data, &envelope)
				switch envelope.Type {
				case "PING":
					_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
					_ = conn.WriteJSON(map[string]string{"type": "PONG"})
				case "RECONNECT":
					errCh <- fmt.Errorf("pubsub reconnect")
					return
				}
				continue
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteJSON(map[string]string{"type": "PING"}); err != nil {
				return err
			}
		}
	}
}

func parseMessage(data []byte) (Event, bool) {
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || envelope.Type != "MESSAGE" {
		return nil, false
	}
	var msg struct {
		Type string `json:"type"`
		Data struct {
			DropID              string  `json:"drop_id"`
			CurrentProgressMin  float64 `json:"current_progress_min"`
			RequiredProgressMin float64 `json:"required_progress_min"`
			DropInstanceID      string  `json:"drop_instance_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(envelope.Data.Message), &msg); err != nil {
		return nil, false
	}
	switch msg.Type {
	case "drop-progress":
		return ProgressEvent{
			DropID:   msg.Data.DropID,
			Current:  msg.Data.CurrentProgressMin,
			Required: msg.Data.RequiredProgressMin,
		}, true
	case "drop-claim":
		return ClaimEvent{DropInstanceID: msg.Data.DropInstanceID}, true
	default:
		return nil, false
	}
}
