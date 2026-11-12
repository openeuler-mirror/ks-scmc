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
	}

	requestURL := strings.TrimSpace(c.cadvisorBaseURL) + ContainersStatsRequestStatsCountOnePath
	body, err := c.httpClient.Get(ctx, requestURL)
	if err != nil {
		return nil, err
	}

	return decodeContainerInfoMap(body)
}

func decodeContainerInfoMap(body []byte) (map[string]*v1.ContainerInfo, error) {
	var list []*v1.ContainerInfo
	if err := json.Unmarshal(body, &list); err == nil && len(list) > 0 {
		result := make(map[string]*v1.ContainerInfo, len(list))
		for i, info := range list {
			if info == nil {
				continue
			}
			key := strings.TrimSpace(info.Name)
			if key == "" {
				key = "/" + strconv.Itoa(i)
			}
			result[key] = info
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	var single v1.ContainerInfo
	if err := json.Unmarshal(body, &single); err == nil {
		if len(single.Stats) > 0 || strings.TrimSpace(single.Name) != "" {
			key := strings.TrimSpace(single.Name)
			if key == "" {
				key = "/"
			}
			return map[string]*v1.ContainerInfo{key: &single}, nil
		}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]*v1.ContainerInfo, len(raw))
	for key, value := range raw {
		if len(value) == 0 {
			continue
