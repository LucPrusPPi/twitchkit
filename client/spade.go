package client

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func resolveSpadeURL(httpClient *http.Client, channelLogin string) (string, error) {
	channelURL := "https://www.twitch.tv/" + channelLogin
	html, err := getText(httpClient, channelURL, WebUA)
	if err != nil {
		return "", err
	}
	if u := extractTrackURL(html); u != "" {
		return u, nil
	}
	settings := extractSettingsJSURL(html)
	if settings == "" {
		return "", fmt.Errorf("spade_url not found for %s", channelLogin)
	}
	js, err := getText(httpClient, settings, WebUA)
	if err != nil {
		return "", err
	}
	if u := extractTrackURL(js); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("spade_url not resolved for %s", channelLogin)
}

func getText(httpClient *http.Client, u, ua string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", ua)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET %s status %d", u, resp.StatusCode)
	}
	return string(b), nil
}

func extractTrackURL(text string) string {
	keys := []string{
		`"spade_url":"`,
		`"spadeUrl":"`,
		`"beacon_url":"`,
		`"beaconUrl":"`,
	}
	for _, key := range keys {
		start := strings.Index(text, key)
		if start < 0 {
			continue
		}
		rest := text[start+len(key):]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			continue
		}
		u := rest[:end]
		u = strings.ReplaceAll(u, `\u002F`, "/")
		u = strings.ReplaceAll(u, `\/`, "/")
		if strings.HasPrefix(u, "https://") {
			return u
		}
	}
	return ""
}

func extractSettingsJSURL(html string) string {
	// Look for settings.js asset references used by the Twitch web client.
	const marker = "settings."
	idx := strings.Index(html, marker)
	for idx >= 0 {
		// walk back to quote start of URL
		from := idx
		for from > 0 && html[from] != '"' && html[from] != '\'' {
			from--
		}
		if from == 0 {
			break
		}
		quote := html[from]
		to := strings.IndexByte(html[idx:], quote)
		if to < 0 {
			break
		}
		candidate := html[from+1 : idx+to]
		if strings.Contains(candidate, "settings") && strings.HasSuffix(candidate, ".js") {
			if strings.HasPrefix(candidate, "//") {
				return "https:" + candidate
			}
			if strings.HasPrefix(candidate, "https://") || strings.HasPrefix(candidate, "http://") {
				return candidate
			}
			if strings.HasPrefix(candidate, "/") {
				return "https://www.twitch.tv" + candidate
			}
		}
		next := strings.Index(html[idx+1:], marker)
		if next < 0 {
			break
		}
		idx = idx + 1 + next
	}
	return ""
}
