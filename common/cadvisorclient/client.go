package cadvisorclient

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	v1 "github.com/google/cadvisor/info/v1"

	"scmc/common"
	"scmc/common/httpclient"
)

const ContainersStatsRequestStatsCountOnePath = "/api/v1.3/subcontainers/?request=stats&count=1"

type CAdvisorMetricsClient struct {
	cadvisorBaseURL string
	httpClient      *httpclient.HTTPClient
}

func NewCAdvisorMetricsClient() *CAdvisorMetricsClient {
	return &CAdvisorMetricsClient{
		cadvisorBaseURL: "http://" + strings.TrimSpace(common.Config.CAdvisor.Addr),
		httpClient:      httpclient.NewHTTPClientForLongRunningPull(),
	}
}

func (c *CAdvisorMetricsClient) FetchContainerStatsOnce(ctx context.Context) (map[string]*v1.ContainerInfo, error) {
	if c == nil {
		c = NewCAdvisorMetricsClient()
	}
	if c.httpClient == nil {
		c.httpClient = httpclient.NewHTTPClientForLongRunningPull()
