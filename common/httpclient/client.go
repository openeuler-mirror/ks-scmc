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
