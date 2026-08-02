package fems

import (
	"net/http"
	"time"
)

// DefaultBaseURL is the base URL of the FEMS climatology API.
const DefaultBaseURL = "https://fems.fs2c.usda.gov/api/climatology"

// defaultUserAgent identifies this client to the server.
const defaultUserAgent = "firewx-fetch/1 (+https://alpineworks.io/firewx)"

// Doer sends an HTTP request and returns the response. The standard
// *http.Client satisfies it. A test can give a stub instead.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client reads data from the FEMS climatology API. A public request needs no
// token and returns the most recent two weeks.
type Client struct {
	baseURL    string
	userAgent  string
	httpClient Doer
}

// Option sets one field of a Client. Give a list of Option values to New.
type Option func(*Client)

// WithHTTPClient sets the HTTP client. The default is a standard client with a
// 60 second timeout, because a FEMS download can be large. Give a stub to test
// without a network.
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

// New returns a Client. With no options, it uses the real API base URL and a
// standard HTTP client with a 60 second timeout.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
