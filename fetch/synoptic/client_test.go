package synoptic

import "testing"

func TestNewDefaults(t *testing.T) {
	c := New()
	if c.baseURL != DefaultBaseURL {
		t.Errorf("base URL: got %q, want %q", c.baseURL, DefaultBaseURL)
	}
	if c.userAgent != defaultUserAgent {
		t.Errorf("user agent: got %q", c.userAgent)
	}
	if c.httpClient == nil {
		t.Error("default HTTP client must not be nil")
	}
	if c.token != "" {
		t.Errorf("default token must be empty, got %q", c.token)
	}
}

func TestOptionsSetFields(t *testing.T) {
	cases := []struct {
		name  string
		opt   Option
		check func(*Client) bool
	}{
		{"token", WithToken("abc"), func(c *Client) bool { return c.token == "abc" }},
		{"base URL", WithBaseURL("http://example.test"), func(c *Client) bool { return c.baseURL == "http://example.test" }},
		{"user agent", WithUserAgent("ua/1"), func(c *Client) bool { return c.userAgent == "ua/1" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.opt)
			if !tc.check(c) {
				t.Errorf("option %s did not set the field", tc.name)
			}
		})
	}
}

func TestEmptyOptionsKeepDefaults(t *testing.T) {
	// An empty value must not clear a sensible default.
	c := New(WithBaseURL(""), WithUserAgent(""), WithHTTPClient(nil))
	if c.baseURL != DefaultBaseURL {
		t.Errorf("empty base URL must keep the default, got %q", c.baseURL)
	}
	if c.userAgent != defaultUserAgent {
		t.Errorf("empty user agent must keep the default, got %q", c.userAgent)
	}
	if c.httpClient == nil {
		t.Error("nil HTTP client must keep the default")
	}
}
