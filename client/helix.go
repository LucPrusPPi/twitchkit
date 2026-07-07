package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/LucPrusPPi/twitchkit/gql"
)

// GameID resolves a category id by exact name (Helix games, then search).
func (c *Client) GameID(gameName string) (string, error) {
	if id, err := c.fetchGameIDByName(gameName); err == nil && id != "" {
		return id, nil
	}
	return c.searchCategoryID(gameName)
}

// DropsStreams lists live streams for a game, preferring Drops Enabled.
func (c *Client) DropsStreams(gameName string) ([]StreamInfo, error) {
	if gid, err := c.fetchGameIDByName(gameName); err == nil && gid != "" {
		streams, err := c.fetchStreamsForGame(gid, DropsEnabledTagID)
		if err == nil && len(streams) > 0 {
			return streams, nil
		}
		all, err := c.fetchStreamsForGame(gid, "")
		if err == nil {
			var filtered []StreamInfo
			for _, s := range all {
				if s.HasDropsTag {
					filtered = append(filtered, s)
				}
			}
			if len(filtered) > 0 {
				return filtered, nil
			}
		}
	}
	streams, err := c.fetchStreamsGQLDirectory(gameName, true)
	if err == nil && len(streams) > 0 {
		return streams, nil
	}
	return c.fetchStreamsGQLDirectory(gameName, false)
}

// LiveStreamsByGameID lists live streams for a Helix game id.
func (c *Client) LiveStreamsByGameID(gameID string) ([]StreamInfo, error) {
	streams, err := c.fetchStreamsForGame(gameID, DropsEnabledTagID)
	if err == nil && len(streams) > 0 {
		return streams, nil
	}
	all, err := c.fetchStreamsForGame(gameID, "")
	if err != nil {
		return nil, err
	}
	var filtered []StreamInfo
	for _, s := range all {
		if s.HasDropsTag {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) > 0 {
		return filtered, nil
	}
	return all, nil
}

// LiveStreamByLogin returns the live stream for a channel login, if any.
func (c *Client) LiveStreamByLogin(channelLogin string) (*StreamInfo, error) {
	login := strings.ToLower(strings.TrimSpace(channelLogin))
	if login == "" {
		return nil, nil
	}
	if s, err := c.fetchStreamByLoginGQL(login); err == nil && s != nil {
		return s, nil
	}
	if s, err := c.fetchStreamByLoginHelix(login, false); err == nil && s != nil {
		return s, nil
	}
	return c.fetchStreamByLoginHelix(login, true)
}

func (c *Client) helixGet(path string, query url.Values) (json.RawMessage, error) {
	u := helixURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-ID", ClientID)
	req.Header.Set("Authorization", "OAuth "+c.token)
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
		return nil, &StatusError{Op: "helix", Status: resp.StatusCode, Body: truncate(string(data), 200)}
	}
	return json.RawMessage(data), nil
}

func (c *Client) fetchGameIDByName(gameName string) (string, error) {
	raw, err := c.helixGet("/games", url.Values{"name": {gameName}})
	if err != nil {
		return "", err
	}
	var v struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	if len(v.Data) == 0 {
		return "", nil
	}
	return v.Data[0].ID, nil
}

func (c *Client) searchCategoryID(gameName string) (string, error) {
	raw, err := c.helixGet("/search/categories", url.Values{"query": {gameName}})
	if err != nil {
		return "", err
	}
	var v struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	wanted := strings.TrimSpace(gameName)
	for _, cat := range v.Data {
		if strings.EqualFold(cat.Name, wanted) {
			return cat.ID, nil
		}
	}
	return "", nil
}

func (c *Client) fetchStreamsForGame(gameID, tagID string) ([]StreamInfo, error) {
	q := url.Values{
		"game_id": {gameID},
		"first":   {"100"},
		"type":    {"live"},
	}
	if tagID != "" {
		q.Set("tag_id", tagID)
	}
	raw, err := c.helixGet("/streams", q)
	if err != nil {
		return nil, err
	}
	return parseHelixStreams(raw, tagID != "")
}

func (c *Client) fetchStreamByLoginHelix(login string, dropsOnly bool) (*StreamInfo, error) {
	q := url.Values{"user_login": {login}, "first": {"1"}}
	if dropsOnly {
		q.Set("tag_id", DropsEnabledTagID)
	}
	raw, err := c.helixGet("/streams", q)
	if err != nil {
		return nil, err
	}
	streams, err := parseHelixStreams(raw, dropsOnly)
	if err != nil || len(streams) == 0 {
		return nil, err
	}
	s := streams[0]
	return &s, nil
}

func (c *Client) fetchStreamByLoginGQL(login string) (*StreamInfo, error) {
	raw, err := c.GQL(gql.StreamInfo(login), "", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			User *struct {
				ID                string `json:"id"`
				Login             string `json:"login"`
				Stream            *struct {
					ID           string `json:"id"`
					ViewersCount uint64 `json:"viewersCount"`
				} `json:"stream"`
				BroadcastSettings struct {
					Game *struct {
						ID          string `json:"id"`
						DisplayName string `json:"displayName"`
						Name        string `json:"name"`
					} `json:"game"`
				} `json:"broadcastSettings"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	u := resp.Data.User
	if u == nil || u.Stream == nil {
		return nil, nil
	}
	gameID, gameName := "", ""
	if u.BroadcastSettings.Game != nil {
		gameID = u.BroadcastSettings.Game.ID
		gameName = u.BroadcastSettings.Game.DisplayName
		if gameName == "" {
			gameName = u.BroadcastSettings.Game.Name
		}
	}
	return &StreamInfo{
		ID:          u.Stream.ID,
		UserID:      u.ID,
		UserLogin:   u.Login,
		GameID:      gameID,
		GameName:    gameName,
		ViewerCount: u.Stream.ViewersCount,
		HasDropsTag: true,
	}, nil
}

func (c *Client) fetchStreamsGQLDirectory(gameName string, dropsOnly bool) ([]StreamInfo, error) {
	slug := gql.GameNameToSlug(gameName)
	raw, err := c.GQL(gql.GameDirectory(slug, 100, dropsOnly), "", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Game *struct {
				Streams struct {
					Edges []struct {
						Node struct {
							ID           string `json:"id"`
							ViewersCount uint64 `json:"viewersCount"`
							Broadcaster  *struct {
								ID    string `json:"id"`
								Login string `json:"login"`
							} `json:"broadcaster"`
							Game *struct {
								ID          string `json:"id"`
								DisplayName string `json:"displayName"`
								Name        string `json:"name"`
							} `json:"game"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"streams"`
			} `json:"game"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Data.Game == nil {
		return nil, nil
	}
	out := make([]StreamInfo, 0, len(resp.Data.Game.Streams.Edges))
	for _, e := range resp.Data.Game.Streams.Edges {
		n := e.Node
		if n.Broadcaster == nil {
			continue
		}
		gameID, gameName := "", ""
		if n.Game != nil {
			gameID = n.Game.ID
			gameName = n.Game.DisplayName
			if gameName == "" {
				gameName = n.Game.Name
			}
		}
		out = append(out, StreamInfo{
			ID:          n.ID,
			UserID:      n.Broadcaster.ID,
			UserLogin:   n.Broadcaster.Login,
			GameID:      gameID,
			GameName:    gameName,
			ViewerCount: n.ViewersCount,
			HasDropsTag: dropsOnly,
		})
	}
	return out, nil
}

func parseHelixStreams(raw json.RawMessage, tagForced bool) ([]StreamInfo, error) {
	var v struct {
		Data []struct {
			ID          string   `json:"id"`
			UserID      string   `json:"user_id"`
			UserLogin   string   `json:"user_login"`
			GameID      string   `json:"game_id"`
			GameName    string   `json:"game_name"`
			ViewerCount uint64   `json:"viewer_count"`
			TagIDs      []string `json:"tag_ids"`
			Tags        []string `json:"tags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	out := make([]StreamInfo, 0, len(v.Data))
	for _, s := range v.Data {
		has := streamHasDropsTag(s.TagIDs, s.Tags)
		if tagForced || has {
			out = append(out, StreamInfo{
				ID:          s.ID,
				UserID:      s.UserID,
				UserLogin:   s.UserLogin,
				GameID:      s.GameID,
				GameName:    s.GameName,
				ViewerCount: s.ViewerCount,
				HasDropsTag: has || tagForced,
			})
		}
	}
	return out, nil
}

func streamHasDropsTag(tagIDs, tags []string) bool {
	for _, t := range tagIDs {
		if t == DropsEnabledTagID {
			return true
		}
	}
	for _, tag := range tags {
		lower := strings.ToLower(tag)
		if strings.Contains(lower, "dropsenabled") ||
			strings.Contains(lower, "drops enabled") ||
			lower == "drops" {
			return true
		}
	}
	return false
}
