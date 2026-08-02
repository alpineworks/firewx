package synoptic

import (
	"net/http"
	"time"
)

// DefaultBaseURL is the base URL of the Synoptic Weather API version 2.
const DefaultBaseURL = "https://api.synopticdata.com/v2"

// defaultUserAgent identifies this client to the server.
const defaultUserAgent = "firewx-fetch/1 (+https://alpineworks.io/firewx)"

// Doer sends an HTTP request and returns the response. The standard
// *http.Client satisfies it. A test can give a stub instead.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client reads data from the Synoptic Weather API.
type Client struct {
	token      string
	baseURL    string
	userAgent  string
	httpClient Doer
}

// Option sets one field of a Client. Give a list of Option values to New.
type Option func(*Client)

// WithToken sets the API token. The Synoptic Weather API needs a token for
// every request.
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithHTTPClient sets the HTTP client. The default is a standard client with a
// 30 second timeout. Give a stub to test without a network.
func WithHTTPClient(d Doer) Option {
	return func(c *Client) {
		if d != nil {
			c.httpClient = d
		}
	}
}

// WithBaseURL sets the base URL. The default is DefaultBaseURL. Give a test
// server URL to test without the real API.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// WithUserAgent sets the User-Agent header for every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// New returns a Client. With no options, it uses the real API base URL, a
// standard HTTP client with a 30 second timeout, and no token. Give WithToken
// before you make a request.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
