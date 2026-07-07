package client

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/LucPrusPPi/twitchkit/auth"
	"github.com/LucPrusPPi/twitchkit/gql"
)

const (
	ClientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"

	DropsEnabledTagID = "6ea6bca4-4712-4ab9-a906-e3336a9d8039"

	UserAgent = "Dalvik/2.1.0 (Linux; U; Android 15; SM-G990B Build/AP3A.240905.015.A2) tv.twitch.android.app/25.3.0/2503006"
	WebUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"

	gqlURL       = "https://gql.twitch.tv/gql"
	helixURL     = "https://api.twitch.tv/helix"
	integrityURL = "https://passport.twitch.tv/integrity"
)

// StreamInfo describes a live channel.
type StreamInfo struct {
	ID          string
	UserID      string
	UserLogin   string
	GameID      string
	GameName    string
	ViewerCount uint64
	HasDropsTag bool
}

// DropSession is the current drop watch context for a channel.
type DropSession struct {
	DropID          string
	CurrentMinutes  float64
	RequiredMinutes float64
	GameName        string
	ChannelName     string
}

// Client is a Twitch GQL/Helix client bound to one auth token.
type Client struct {
	http         *http.Client
	token        string
	sessionID    string
	userAgent    string
	webUserAgent string
	mu           sync.Mutex
	deviceID     string
	integrity    string
	spadeByLogin map[string]string
}

// Token returns the normalized auth token.
func (c *Client) Token() string { return c.token }

// Validate checks the token and returns login + user id.
func (c *Client) Validate() (auth.TokenInfo, error) {
	_ = c.fetchIntegrity()
	return auth.Validate(c.http, c.token)
}

// EnsureDeviceID bootstraps unique_id from www.twitch.tv (once).
func (c *Client) EnsureDeviceID() error {
	c.mu.Lock()
	if c.deviceID != "" {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	uid, err := c.bootstrapUniqueID()
	if err != nil {
		return err
	}
	c.mu.Lock()
	if c.deviceID == "" {
		c.deviceID = uid
	}
	c.mu.Unlock()
	return nil
}

// GQL posts a GraphQL / persisted-query body.
func (c *Client) GQL(body any, userID, channelLogin string) (json.RawMessage, error) {
	if userID != "" {
		_ = c.EnsureDeviceID()
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, gqlURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c.applyGQLHeaders(req, userID, channelLogin)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &StatusError{Op: "gql", Status: resp.StatusCode, Body: truncate(string(data), 200)}
	}
	return json.RawMessage(data), nil
}

// Inventory fetches the Inventory persisted query.
func (c *Client) Inventory(userID string) (json.RawMessage, error) {
	return c.GQL(gql.Inventory(), userID, "")
}

// Campaigns fetches ViewerDropsDashboard.
func (c *Client) Campaigns(userID string) (json.RawMessage, error) {
	return c.GQL(gql.Campaigns(), userID, "")
}

// ClaimDrop claims a drop instance.
func (c *Client) ClaimDrop(dropInstanceID, userID string) (json.RawMessage, error) {
	return c.GQL(gql.ClaimDrop(dropInstanceID), userID, "")
}

// CampaignDetails fetches DropCampaignDetails.
func (c *Client) CampaignDetails(channelLogin, campaignID, userID string) (json.RawMessage, error) {
	return c.GQL(gql.CampaignDetails(channelLogin, campaignID), userID, "")
}

// CurrentDropSession parses DropCurrentSessionContext for a channel.
func (c *Client) CurrentDropSession(channelID, userID string) (*DropSession, error) {
	raw, err := c.GQL(gql.CurrentDrop(channelID), userID, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			CurrentUser struct {
				DropCurrentSession *struct {
					DropID                 string  `json:"dropID"`
					CurrentMinutesWatched  float64 `json:"currentMinutesWatched"`
					RequiredMinutesWatched float64 `json:"requiredMinutesWatched"`
					Game                   struct {
						DisplayName string `json:"displayName"`
					} `json:"game"`
					Channel struct {
						Name string `json:"name"`
					} `json:"channel"`
				} `json:"dropCurrentSession"`
			} `json:"currentUser"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	s := resp.Data.CurrentUser.DropCurrentSession
	if s == nil || s.DropID == "" {
		return nil, nil
	}
	return &DropSession{
		DropID:          s.DropID,
		CurrentMinutes:  s.CurrentMinutesWatched,
		RequiredMinutes: s.RequiredMinutesWatched,
		GameName:        s.Game.DisplayName,
		ChannelName:     s.Channel.Name,
	}, nil
}

// SendWatch reports a minute-watched event (spade first, GQL fallback).
func (c *Client) SendWatch(channelLogin, channelID, broadcastID, userID, gameName, gameID string) error {
	if err := c.EnsureDeviceID(); err != nil {
		return err
	}
	if err := c.sendSpadeWatch(channelLogin, channelID, broadcastID, userID, gameName, gameID); err == nil {
		return nil
	}
	return c.sendGQLWatch(channelLogin, channelID, broadcastID, userID, gameName, gameID)
}

func (c *Client) sendGQLWatch(channelLogin, channelID, broadcastID, userID, gameName, gameID string) error {
	payload, err := json.Marshal([]map[string]any{{
		"event":      "minute-watched",
		"properties": watchProperties(channelLogin, channelID, broadcastID, userID, gameName, gameID),
	}})
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	raw, err := c.GQL(gql.SendSpadeEvents(b64), userID, channelLogin)
	if err != nil {
		return err
	}
	var resp struct {
		Data struct {
			SendSpadeEvents struct {
				StatusCode int `json:"statusCode"`
			} `json:"sendSpadeEvents"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	if resp.Data.SendSpadeEvents.StatusCode != 204 {
		return &StatusError{Op: "gql-spade", Status: resp.Data.SendSpadeEvents.StatusCode}
	}
	return nil
}

func (c *Client) sendSpadeWatch(channelLogin, channelID, broadcastID, userID, gameName, gameID string) error {
	login := strings.ToLower(strings.TrimSpace(channelLogin))
	spadeURL, err := c.resolveSpadeURL(login)
	if err != nil {
		return err
	}
	payload, err := json.Marshal([]map[string]any{{
		"event":      "minute-watched",
		"properties": watchProperties(channelLogin, channelID, broadcastID, userID, gameName, gameID),
	}})
	if err != nil {
		return err
	}
	data := base64.StdEncoding.EncodeToString(payload)
	form := url.Values{"data": {data}}
	req, err := http.NewRequest(http.MethodPost, spadeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Cookie", c.cookieHeader(userID))
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 204 {
		c.mu.Lock()
		delete(c.spadeByLogin, login)
		c.mu.Unlock()
		return &StatusError{Op: "spade", Status: resp.StatusCode}
	}
	return nil
}

func (c *Client) resolveSpadeURL(channelLogin string) (string, error) {
	c.mu.Lock()
	if u, ok := c.spadeByLogin[channelLogin]; ok {
		c.mu.Unlock()
		return u, nil
	}
	c.mu.Unlock()

	u, err := resolveSpadeURL(c.http, channelLogin, c.webUserAgent)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.spadeByLogin[channelLogin] = u
	c.mu.Unlock()
	return u, nil
}

func (c *Client) fetchIntegrity() error {
	req, err := http.NewRequest(http.MethodPost, integrityURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Client-ID", ClientID)
	req.Header.Set("Authorization", "OAuth "+c.token)
	req.Header.Set("User-Agent", c.webUserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &StatusError{Op: "integrity", Status: resp.StatusCode, Body: truncate(string(body), 200)}
	}
	var v struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return err
	}
	if v.Token != "" {
		c.mu.Lock()
		c.integrity = v.Token
		c.mu.Unlock()
	}
	return nil
}

func (c *Client) applyGQLHeaders(req *http.Request, userID, channelLogin string) {
	c.mu.Lock()
	device := c.deviceID
	integrity := c.integrity
	session := c.sessionID
	c.mu.Unlock()

	req.Header.Set("Client-ID", ClientID)
	req.Header.Set("Authorization", "OAuth "+c.token)
	req.Header.Set("Client-Session-Id", session)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Device-Id", device)
	req.Header.Set("Cookie", c.cookieHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://www.twitch.tv")
	if channelLogin != "" {
		req.Header.Set("Referer", "https://www.twitch.tv/"+strings.TrimPrefix(channelLogin, "@"))
	} else {
		req.Header.Set("Referer", "https://www.twitch.tv/")
	}
	if integrity != "" {
		req.Header.Set("Client-Integrity", integrity)
	}
}

func (c *Client) cookieHeader(userID string) string {
	c.mu.Lock()
	device := c.deviceID
	c.mu.Unlock()
	parts := []string{
		"auth-token=" + c.token,
		"unique_id=" + device,
	}
	if userID != "" {
		parts = append(parts, "persistent="+userID)
	}
	return strings.Join(parts, "; ")
}

func watchProperties(channelLogin, channelID, broadcastID, userID, gameName, gameID string) map[string]any {
	return map[string]any{
		"broadcast_id":   broadcastID,
		"channel_id":     channelID,
		"channel":        channelLogin,
		"client_time":    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"game":           gameName,
		"game_id":        gameID,
		"hidden":         false,
		"is_live":        true,
		"live":           true,
		"location":       "channel",
		"logged_in":      true,
		"minutes_logged": 1,
		"muted":          false,
		"player":         "site",
		"user_id":        userID,
	}
}

func (c *Client) bootstrapUniqueID() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.twitch.tv", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Cookie", "auth-token="+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	for _, sc := range resp.Header.Values("Set-Cookie") {
		if uid := parseSetCookie(sc, "unique_id"); uid != "" {
			return uid, nil
		}
	}
	return "", &StatusError{Op: "bootstrap", Status: resp.StatusCode, Body: "unique_id cookie missing"}
}

func parseSetCookie(setCookie, name string) string {
	part := strings.SplitN(setCookie, ";", 2)[0]
	kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
	if len(kv) != 2 {
		return ""
	}
	if strings.TrimSpace(kv[0]) == name {
		return strings.TrimSpace(kv[1])
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
