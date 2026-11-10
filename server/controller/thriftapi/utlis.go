package thriftapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"scmc/model/db"
	containerPB "scmc/rpc/pb/container"
	"scmc/thrift/gen-go/ksscmc"
)

// 通过node name or ip查找node info
func GetNodeId(nodeNameorip string) (*db.NodeInfo, error) {
	nodeInfo, err := db.QueryNodeByAddr(nodeNameorip)
	if err != nil {
		nodeInfo, err = db.QueryNodeByName(nodeNameorip)
		if err != nil {
			return nil, err
		}
	}
	return nodeInfo, nil
}
func GetContainerId(cli containerPB.ContainerClient, containerName string, ctx context.Context) (containerId string, err error) {

	r, err := cli.List(ctx, &containerPB.ListRequest{
		ListAll: true,
	})
	if err != nil {
		return "", err
	}

	for i := range r.Containers {
		if r.Containers[i].Info.Name == containerName {
			containerId = r.Containers[i].Info.Id
		}
	}
	if containerId == "" {
		return "", errors.New("containerId not found")
	}
	return containerId, nil
}

func Struct2JsonString(obj interface{}) string {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		logrus.Warn("Failed to serialize, because %s", err)
	}
	return string(jsonData)
}

func ResponseFailed(errMsg string) string {
	resp := ksscmc.CommonRespone{
		Status: 1,
	}
	if errMsg == "" {
		resp.Message = "failed"
	} else {
		resp.Message = errMsg
	}
	return Struct2JsonString(resp)
}

func ResponseSucceed() string {
	resp := ksscmc.CommonRespone{
		Status:  0,
		Message: "success",
	}
	return Struct2JsonString(resp)
}

func Response2String(status int32) string {
	var msg string
	if status == 0 {
		msg = "success"
	} else {
		msg = "failed"
	}
	respJson := ResponseJson{
		Status:  status,
		Message: msg,
	}
	jsonData, _ := json.Marshal(respJson)
	return string(jsonData)
}

// 节点告警结构体
type HAlarm struct {
	ID      int64  `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	CTime   string `json:"c_time"` // 时间字段直接使用字符串
}

// 容器告警结构体（比节点告警多一个 c_name 字段）
type CAlarm struct {
	ID      int64  `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	CTime   string `json:"c_time"`
	CName   string `json:"c_name"` // 容器名称字段
}
type AlarmResponse struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	HAlarms []HAlarm `json:"h_alarms"` // 节点告警列表
	CAlarms []CAlarm `json:"c_alarms"` // 容器告警列表
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
