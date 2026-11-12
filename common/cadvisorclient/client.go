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
