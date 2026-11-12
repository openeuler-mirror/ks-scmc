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

