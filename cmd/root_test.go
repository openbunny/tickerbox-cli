package cmd

import (
	"testing"
	"time"
)

func TestRestBase(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "no trailing slash", host: "http://tickerbox.local", want: "http://tickerbox.local/rest/"},
		{name: "trailing slash", host: "http://tickerbox.local/", want: "http://tickerbox.local/rest/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := restBase(tt.host); got != tt.want {
				t.Errorf("restBase(%q) = %q; want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestNewClientAppliesFlags(t *testing.T) {
	origHost, origTimeout, origRetry := resolvedHost, timeoutFlag, retryFlag
	defer func() { resolvedHost, timeoutFlag, retryFlag = origHost, origTimeout, origRetry }()

	resolvedHost = "http://example.invalid"
	timeoutFlag = 5 * time.Second
	retryFlag = 4

	c := newClient()
	if c.Retries != retryFlag {
		t.Errorf("got Retries %d; want %d", c.Retries, retryFlag)
	}
}
