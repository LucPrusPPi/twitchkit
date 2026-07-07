package client

import (
	"net/http"
	"net/url"
	"time"

	"github.com/LucPrusPPi/twitchkit/auth"
)

// Options configures a Client. Zero values mean defaults.
type Options struct {
	// Timeout for the underlying HTTP client. Default: 30s.
	Timeout time.Duration

	// Proxy is an optional HTTP(S) proxy URL, e.g. "http://127.0.0.1:8080".
	// Ignored when HTTPClient is set.
	Proxy string

	// HTTPClient overrides Timeout/Proxy entirely when non-nil.
	HTTPClient *http.Client

	// UserAgent for GQL / mobile-style requests. Default: Android Twitch app UA.
	UserAgent string

	// WebUserAgent for HTML/settings fetches (spade URL resolve). Default: Chrome UA.
	WebUserAgent string
}

func (o Options) withDefaults() Options {
	if o.Timeout <= 0 {
		o.Timeout = 30 * time.Second
	}
	if o.UserAgent == "" {
		o.UserAgent = UserAgent
	}
	if o.WebUserAgent == "" {
		o.WebUserAgent = WebUA
	}
	return o
}

func (o Options) httpClient() (*http.Client, error) {
	if o.HTTPClient != nil {
		return o.HTTPClient, nil
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConnsPerHost: 4,
	}
	if o.Proxy != "" {
		u, err := url.Parse(o.Proxy)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(u)
	}
	return &http.Client{
		Timeout:   o.Timeout,
		Transport: transport,
	}, nil
}

// New creates a Client with default options.
func New(token string) *Client {
	c, err := NewWithOptions(token, Options{})
	if err != nil {
		// Defaults never fail; keep New signature simple for farmers.
		panic(err)
	}
	return c
}

// NewWithOptions creates a Client. Returns an error only if Proxy is malformed
// (or HTTPClient construction fails).
func NewWithOptions(token string, opt Options) (*Client, error) {
	opt = opt.withDefaults()
	hc, err := opt.httpClient()
	if err != nil {
		return nil, err
	}
	return &Client{
		http:         hc,
		token:        auth.Normalize(token),
		sessionID:    randomHex(16),
		spadeByLogin: make(map[string]string),
		userAgent:    opt.UserAgent,
		webUserAgent: opt.WebUserAgent,
	}, nil
}
