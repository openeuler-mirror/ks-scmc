package httpclient

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const hTPClientTimeout = 10 * time.Second

// longRunningHTTPClientTimeout is for large responses (e.g. cAdvisor subcontainers with stats).
const longRunningHTTPClientTimeout = 120 * time.Second

type HTTPClient struct {
	inner *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		inner: &http.Client{Timeout: hTPClientTimeout},
	}
}

// NewHTTPClientForLongRunningPull uses a longer timeout; callers must not pass a shorter context deadline.
func NewHTTPClientForLongRunningPull() *HTTPClient {
	return &HTTPClient{
		inner: &http.Client{Timeout: longRunningHTTPClientTimeout},
	}
}

func (c *HTTPClient) Get(ctx context.Context, requestURL string) ([]byte, error) {
	if c == nil || c.inner == nil {
		return nil, fmt.Errorf("reusable http client is not initialized")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.inner.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
