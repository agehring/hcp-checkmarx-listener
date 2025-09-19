package httpclient

import (
	"context"
	"net"
	"net/http"
	"time"
)

// userAgentTransport wraps the base transport to add a custom User-Agent header
type userAgentTransport struct {
	Transport http.RoundTripper
	UserAgent string
}

func (uat *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	newReq := req.Clone(req.Context())
	// Set the custom User-Agent header
	newReq.Header.Set("User-Agent", uat.UserAgent)
	// Use the underlying transport to make the request
	return uat.Transport.RoundTrip(newReq)
}

// NewClient creates a new HTTP client with the CxOne-HCP User-Agent
func NewClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &userAgentTransport{
			UserAgent: "CxOne-HCP",
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					// Force IPv4
					return (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 30 * time.Second,
						DualStack: false,
					}).DialContext(ctx, "tcp4", addr)
				},
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

var Client = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &userAgentTransport{
		UserAgent: "CxOne-HCP",
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Force IPv4
				return (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: false,
				}).DialContext(ctx, "tcp4", addr)
			},
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	},
}
