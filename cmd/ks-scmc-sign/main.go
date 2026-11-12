// ks-scmc-sign 对 tar 文件进行签名、验签、查看信息、移除签名
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/common/tarsign"
)

const defaultConfigPath = "/etc/ks-scmc/server.toml"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	// 解析全局 -config，未传入则使用默认路径
	configFile := defaultConfigPath
	fs := flag.NewFlagSet("ks-scmc-sign", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configFlag := fs.String("config", "", "配置文件路径（可选）")
	_ = fs.Parse(os.Args[1:])
	if *configFlag != "" {
		configFile = *configFlag
	}

	rest := fs.Args()
