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
	if len(rest) < 1 {
		usage()
		os.Exit(1)
	}
	cmd := rest[0]
	args := rest[1:]

	if err := common.LoadConfig(configFile); err != nil {
		logrus.Errorf("load config file=%s error=%v", configFile, err)
		fmt.Fprintln(os.Stderr, "加载配置失败")
		os.Exit(1)
	}
	common.InitLogger("ks-scmc-sign")

	switch cmd {
	case "sign":
		runSign(args)
	case "verify":
		runVerify(args)
	case "info":
		runInfo(args)
	case "remove":
		runRemove(args)
	case "genkey":
		runGenkey(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

// failErr 将详细错误写入日志，stderr 仅输出简要描述
func failErr(simpleMsg string, err error) {
	logrus.Errorf("%s: %v", simpleMsg, err)
	fmt.Fprintln(os.Stderr, simpleMsg)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `ks-scmc-sign - tar 文件签名工具

用法:
  ks-scmc-sign [-config <配置文件>] sign   [-key <私钥>] [-hash sha256|sha512|sm3] [-sig pkcs1v15|pss|sm2] <file.tar>
  ks-scmc-sign [-config <配置文件>] verify [-pub <公钥>] <file.tar>
  ks-scmc-sign [-config <配置文件>] info   <file.tar>
  ks-scmc-sign [-config <配置文件>] remove <file.tar>
  ks-scmc-sign [-config <配置文件>] genkey [-key <私钥路径>] [-cert <证书路径>] [-algo rsa|sm2] [-days 365]

配置文件默认: /etc/ks-scmc/server.toml

命令说明:
  sign   对 tar 文件签名（幂等；可选 --sign-file 记录指定文件的 hash）
  verify 使用公钥或证书验签
  info   查看签名信息与元数据
  remove 移除签名，恢复为纯 tar
  genkey 生成签名所需的私钥和自签名证书

算法约束: RSA 支持 sha256/sha512 + pkcs1v15/pss；SM2 支持 sha256/sha512/sm3 + sm2

示例:
  ks-scmc-sign genkey
  ks-scmc-sign sign image.tar
  ks-scmc-sign verify image.tar
  ks-scmc-sign -config /other/server.toml sign image.tar
`)
}

// stringSliceFlag 支持通过重复传入 flag 收集多个值，例如：
// --sign-file /bin/bash --sign-file /bin/ls
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runSign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	keyPath := fs.String("key", "", "私钥文件路径（PKCS#8 格式），未指定时从配置文件读取")
	hashAlgo := fs.String("hash", "sha256", "摘要算法: sha256, sha512, sm3")
	sigAlgo := fs.String("sig", "pkcs1v15", "签名算法: pkcs1v15, pss, sm2")
	var signFiles stringSliceFlag
	fs.Var(
		&signFiles,
		"sign-file",
		"需要记录 hash 的文件路径，可重复指定，如: --sign-file /bin/bash --sign-file /do.sh。\n\n示例：ks-scmc-sign sign --sign-file /bin/bash --sign-file /do.sh image.tar",
	)
	fs.Parse(args)

	path := *keyPath
	if path == "" {
		path = common.Config.Controller.ImageSignKey
	}
	if path == "" {
		failErr("必须指定 -key 参数或配置 controller.image-sign-key", fmt.Errorf("image-sign-key not configured"))
	}
	if fs.NArg() == 0 {
		failErr("必须指定要签名的 tar 文件", fmt.Errorf("missing tar file argument"))
	}
	filePath := fs.Arg(0)

	priv, err := tarsign.LoadPrivateKey(path)
	if err != nil {
		failErr("加载私钥失败", err)
	}

	// 计算指定文件 hash
	binaryHashes, err := tarsign.ComputeBinaryHashes(filePath, tarsign.HashAlgo(*hashAlgo), []string(signFiles))
	if err != nil {
		failErr("计算关键二进制 hash 失败", err)
	}

	err = tarsign.SignV2(filePath, priv, tarsign.HashAlgo(*hashAlgo), tarsign.SigAlgo(*sigAlgo), binaryHashes)
	if err != nil {
		failErr("签名失败", err)
	}
	fmt.Println("签名成功")
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pubPath := fs.String("pub", "", "公钥或证书文件路径，未指定时从配置文件读取")
	fs.Parse(args)

	path := *pubPath
	if path == "" {
		path = common.Config.Controller.ImageSigner
	}
	if path == "" {
		failErr("必须指定 -pub 参数或配置 controller.image-signer", fmt.Errorf("image-signer not configured"))
	}
	if fs.NArg() == 0 {
		failErr("必须指定要验签的 tar 文件", fmt.Errorf("missing tar file argument"))
	}
	filePath := fs.Arg(0)

	err := tarsign.VerifyFromFile(filePath, path)
	if err != nil {
		failErr("验签失败", err)
	}
	fmt.Println("验签成功")
}

func runInfo(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() == 0 {
		failErr("必须指定 tar 文件", fmt.Errorf("missing tar file argument"))
	}
	filePath := fs.Arg(0)

	meta, err := tarsign.Info(filePath)
	if err != nil {
		if err == tarsign.ErrNoFooter {
			fmt.Println("未签名")
			return
		}
		failErr("读取签名信息失败", err)
	}

	b, _ := json.MarshalIndent(meta, "", "  ")
	fmt.Println(string(b))
}

func runRemove(args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() == 0 {
		failErr("必须指定 tar 文件", fmt.Errorf("missing tar file argument"))
	}
	filePath := fs.Arg(0)

	err := tarsign.Remove(filePath)
	if err != nil {
		failErr("移除签名失败", err)
	}
	fmt.Println("已移除签名")
}

func runGenkey(args []string) {
	fs := flag.NewFlagSet("genkey", flag.ExitOnError)
	keyPath := fs.String("key", "", "私钥输出路径（未指定时使用配置 controller.image-sign-key）")
	certPath := fs.String("cert", "", "证书输出路径（未指定时使用配置 controller.image-signer）")
	algo := fs.String("algo", "rsa", "密钥算法: rsa, sm2")
	days := fs.Int("days", 36500, "证书有效天数")
	fs.Parse(args)

	keyOut := *keyPath
	if keyOut == "" {
		keyOut = common.Config.Controller.ImageSignKey
	}
	certOut := *certPath
