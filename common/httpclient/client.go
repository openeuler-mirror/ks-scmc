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

