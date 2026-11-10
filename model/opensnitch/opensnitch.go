package opensnitch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model/litedb"
)

type Rule struct {
	Name       string   `json:"name"`
	Enabled    bool     `json:"enabled"`
	Precedence bool     `json:"precedence"`
	Action     string   `json:"action"`
	Duration   string   `json:"duration"`
	Operator   Operator `json:"operator"`
}

// Operator represents what we want to filter of a connection, and how.
type Operator struct {
	Type      string     `json:"type"`
	Operand   string     `json:"operand"`
	Sensitive bool       `json:"sensitive,omitempty"`
	Data      string     `json:"data,omitempty"`
	List      []Operator `json:"list,omitempty"`
}

func (r *Rule) Save(filePath string) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("error while saving rule %v to %s: %s", r, filePath, err)
	}

	if err = ioutil.WriteFile(filePath, raw, 0o644); err != nil {
		return fmt.Errorf("error while saving rule %v to %s: %s", r, filePath, err)
	}

	return nil
}

func allowRulePath(containerID string) string {
	return filepath.Join(common.Config.Agent.OpensnitchRuleDir, "0002-"+containerID+"-allow.json")
}

func denyRulePath(containerID string) string {
	return filepath.Join(common.Config.Agent.OpensnitchRuleDir, "0003-"+containerID+"-deny.json")
}

// Exists checks if a path exists.
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// LoadRule loads rules files from disk.
func LoadRule(path string, containerID string) (*Rule, error) {
	if !Exists(path) {
		return nil, fmt.Errorf("path '%s' does not exist", path)
	}
	fileName := containerID + ".json"
	filePath := filepath.Join(path, fileName)

	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error while reading %s: %s", fileName, err)
	}

	var r Rule
	err = json.Unmarshal(raw, &r)
	if err != nil {
		return nil, fmt.Errorf("error parsing rule from %s: %s", fileName, err)
	}

	return &r, nil
}

// SaveRule 将网络进程白名单规则写入文件
func SaveRule(p *litedb.ProcProtection, containerID, uuid string) error {
	if uuid == "" {
		return fmt.Errorf("opensnitch rule cannot has empty uuid") // 保存空uuid导致所有连接都异常
	}

	allowRule := Rule{
		Name:       containerID + "-allow",
		Enabled:    p.IsOn,
		Precedence: true,
		Duration:   "always",
		Action:     "allow",
		Operator: Operator{
			Type:    "list",
			Operand: "list",
			List: []Operator{
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
	if err := allowRule.Save(allowRulePath(containerID)); err != nil {
		return err
	}

	denyRule := Rule{
		Name:       containerID + "-deny",
		Enabled:    p.IsOn,
		Precedence: false,
		Duration:   "always",
		Action:     "deny",
		Operator: Operator{
			Type:    "simple",
			Operand: "process.env.KS_SCMC_UUID",
			Data:    uuid,
		},
	}
	if err := denyRule.Save(denyRulePath(containerID)); err != nil {
		return err
	}

	return nil
}

func regexpEscapePaths(paths []string) []string {
	escaped := make([]string, len(paths))
	for i, p := range paths {
		// 新增：清理路径首尾空格
		cleanedPath := strings.TrimSpace(p)
		escaped[i] = regexp.QuoteMeta(cleanedPath)
	}
	return escaped
}

func SaveNodeRule(p *litedb.ProcProtection, nodeID string, uuid string) error {
	// 允许规则（仅匹配进程路径）
	allowRule := Rule{
		Name:       nodeID + "-allow",
		Enabled:    p.IsOn,
		Precedence: true, // 高优先级
		Duration:   "always",
		Action:     "allow",
		Operator: Operator{
			Type:    "regexp",
			Operand: "process.path",
			Data:    fmt.Sprintf("^(%s)", strings.Join(regexpEscapePaths(p.ExeList), "|")),
		},
	}

	// 拒绝规则（拒绝所有其他进程）
	denyRule := Rule{
		Name:       nodeID + "-deny",
		Enabled:    p.IsOn,
		Precedence: false, // 低优先级
		Duration:   "always",
		Action:     "deny",
		Operator: Operator{
			Type:    "regexp",
			Operand: "process.path",
			Data:    ".*", // 匹配所有路径
		},
	}

	// 保存规则
	if err := allowRule.Save(allowRulePath(nodeID)); err != nil {
		return err
	}
	if err := denyRule.Save(denyRulePath(nodeID)); err != nil {
		return err
	}
	return nil
}

// RemoveRule删除容器网络进程白名单规则文件
func RemoveRule(containerID string) error {
	err0 := os.Remove(allowRulePath(containerID))
	err1 := os.Remove(denyRulePath(containerID))
	if err0 != nil || err1 != nil {
		log.Warnf("RemoveRule err0=%v err1=%v", err0, err1)
	}

	return nil
}
