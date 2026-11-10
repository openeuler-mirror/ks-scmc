package internal

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
)

// 修改keepalived配置中的vrrp_instance VI_1的interface字段
func modifyInterface(configPath, newInterface string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	// 获取文件信息以确定读取缓冲区大小
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 读取文件内容
	content := make([]byte, fileInfo.Size())
	_, err = file.Read(content)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 正则匹配 vrrp_instance VI_1 块中的 interface 行
	re := regexp.MustCompile(`(vrrp_instance VI_1\s*\{\s*.*?interface\s+)\S+`)
	newContent := re.ReplaceAllStringFunc(string(content), func(match string) string {
		// 替换interface后的旧值为新值
		return regexp.MustCompile(`interface\s+\S+`).ReplaceAllString(match, "interface "+newInterface)
	})

	// 关闭文件后重新以可写方式打开
	file.Close()
	file, err = os.OpenFile(configPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 写入修改后的内容
	_, err = file.WriteString(newContent)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

func ModifyInterface(newInterface string) {
	configPath := "/etc/keepalived/keepalived.conf"
	err := modifyInterface(configPath, newInterface)
	if err != nil {
		log.Warnf("modify keepalived config err: %v", err)
	}
}
