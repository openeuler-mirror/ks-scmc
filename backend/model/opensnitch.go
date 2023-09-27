package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"scmc/common"
	"strings"

	log "github.com/sirupsen/logrus"
)

type OpensnitchRule struct {
	Name       string             `json:"name"`
	Enabled    bool               `json:"enabled"`
	Precedence bool               `json:"precedence"`
	Action     string             `json:"action"`
	Duration   string             `json:"duration"`
	Operator   OpensnitchOperator `json:"operator"`
}

// OpensnitchOperator represents what we want to filter of a connection, and how.
type OpensnitchOperator struct {
	Type      string               `json:"type"`
	Operand   string               `json:"operand"`
	Sensitive bool                 `json:"sensitive,omitempty"`
	Data      string               `json:"data,omitempty"`
	List      []OpensnitchOperator `json:"list,omitempty"`
}

func (r *OpensnitchRule) Save(filePath string) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("error while saving rule %v to %s: %s", r, filePath, err)
	}

	if err = ioutil.WriteFile(filePath, raw, 0644); err != nil {
		return fmt.Errorf("error while saving rule %v to %s: %s", r, filePath, err)
	}

	return nil
}

func opensnitchAllowRulePath(containerID string) string {
	return filepath.Join(common.Config.Agent.OpensnitchRuleDir, "0002-"+containerID+"-allow.json")
}

func opensnitchDenyRulePath(containerID string) string {
	return filepath.Join(common.Config.Agent.OpensnitchRuleDir, "0003-"+containerID+"-deny.json")
}

// Exists checks if a path exists.
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// LoadOpensnitchRule loads rules files from disk.
func LoadOpensnitchRule(path string, containerID string) (*OpensnitchRule, error) {
	if !Exists(path) {
		return nil, fmt.Errorf("path '%s' does not exist", path)
	}
	fileName := containerID + ".json"
	filePath := filepath.Join(path, fileName)

	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error while reading %s: %s", fileName, err)
	}

	var r OpensnitchRule
	err = json.Unmarshal(raw, &r)
	if err != nil {
		return nil, fmt.Errorf("error parsing rule from %s: %s", fileName, err)
	}

	return &r, nil
}

// SaveOpensnitchRule 将网络进程白名单规则写入文件
func SaveOpensnitchRule(p *ProcProtection, containerID, uuid string) error {
	if uuid == "" {
		return fmt.Errorf("opensnitch rule cannot has empty uuid") // 保存空uuid导致所有连接都异常
	}

	var allowRule = OpensnitchRule{
		Name:       containerID + "-allow",
		Enabled:    p.IsOn,
		Precedence: true,
		Duration:   "always",
		Action:     "allow",
		Operator: OpensnitchOperator{
			Type:    "list",
			Operand: "list",
			List: []OpensnitchOperator{
				{
					Type:    "regexp",
					Operand: "process.path",
					Data:    fmt.Sprintf("^(%s)$", strings.Join(p.ExeList, "|")),
				},
				{
					Type:    "simple",
					Operand: "process.env.KS_SCMC_UUID",
					Data:    uuid,
				},
			},
		},
	}
	if err := allowRule.Save(opensnitchAllowRulePath(containerID)); err != nil {
		return err
	}

	var denyRule = OpensnitchRule{
		Name:       containerID + "-deny",
		Enabled:    p.IsOn,
		Precedence: false,
		Duration:   "always",
		Action:     "deny",
		Operator: OpensnitchOperator{
			Type:    "simple",
			Operand: "process.env.KS_SCMC_UUID",
			Data:    uuid,
		},
	}
	if err := denyRule.Save(opensnitchDenyRulePath(containerID)); err != nil {
		return err
	}

	// TODO 重启systemd服务
	return nil
}

// RemoveOpensnitchRule删除容器网络进程白名单规则文件
func RemoveOpensnitchRule(containerID string) error {
	err0 := os.Remove(opensnitchAllowRulePath(containerID))
	err1 := os.Remove(opensnitchDenyRulePath(containerID))
	if err0 != nil || err1 != nil {
		log.Warnf("RemoveOpensnitchRule err0=%v err1=%v", err0, err1)
	}

	// TODO 重启systemd服务
	return nil
}
