// Package tarsign 实现 tar 文件签名方案，签名数据与 tar 合并在同一文件中。
// 格式：TAR DATA | SIGNATURE | METADATA | FOOTER(32字节)
package tarsign

import (
	"archive/tar"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
	gmsmx509 "github.com/tjfoc/gmsm/x509"
)

const (
	FooterSize = 32
	Magic      = "SCMCSIGN"

	// VersionV1: 仅对 tar_digest 做签名，不包含 metadata，本仓库早期实现
	VersionV1 = 1
	// VersionV2: 对 tar_digest 与 metadata 一起做签名（签名覆盖 binary_hashes 等字段）
	VersionV2 = 2

	ToolName = "ks-scmc-sign"
)

// Footer 32 字节固定结构
type Footer struct {
	Magic   [8]byte
	SigLen  uint64
	MetaLen uint64
	Version uint64
}

// Metadata 签名元数据
type Metadata struct {
	TarDigest string `json:"tar_digest"`
	HashAlgo  string `json:"hash_algo"`
	SigAlgo   string `json:"sig_algo"`
	KeyID     string `json:"key_id"`
	Timestamp int64  `json:"timestamp"`
	Tool      string `json:"tool"`
	Version   int    `json:"version"`
	// BinaryHashes v2 新增：关键二进制路径 -> hash（algo:hex）
	BinaryHashes map[string]string `json:"binary_hashes,omitempty"`
}

// HashAlgo 摘要算法
type HashAlgo string

const (
	HashSHA256 HashAlgo = "sha256"
	HashSHA512 HashAlgo = "sha512"
	HashSM3    HashAlgo = "sm3"
)

// SigAlgo 签名算法
type SigAlgo string

const (
	SigPKCS1v15 SigAlgo = "pkcs1v15"
	SigPSS      SigAlgo = "pss"
	SigSM2      SigAlgo = "sm2"
)

var (
	ErrNoFooter       = errors.New("未找到签名 Footer，视为未签名")
	ErrMagicMismatch  = errors.New("magic 不匹配，非本方案签名")
	ErrBadStructure   = errors.New("签名区结构损坏")
	ErrDigestMismatch = errors.New("tar 摘要不匹配，内容可能被篡改")
	ErrVerifyFailed   = errors.New("验签失败")
)

// ParseFooter 从文件末尾读取并解析 Footer
func ParseFooter(f *os.File) (*Footer, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() < FooterSize {
		return nil, ErrNoFooter
	}

	_, err = f.Seek(-FooterSize, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	var foot Footer
	if err := binary.Read(f, binary.LittleEndian, &foot.Magic); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &foot.SigLen); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &foot.MetaLen); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &foot.Version); err != nil {
		return nil, err
	}

	if string(foot.Magic[:]) != Magic {
		return nil, ErrMagicMismatch
	}

	totalExtra := int64(foot.SigLen) + int64(foot.MetaLen) + FooterSize
	if stat.Size() < totalExtra {
		return nil, ErrBadStructure
	}

	return &foot, nil
}

// IsSigned 检查文件是否已签名
func IsSigned(f *os.File) (bool, error) {
	foot, err := ParseFooter(f)
	if err != nil {
		if errors.Is(err, ErrNoFooter) || errors.Is(err, ErrMagicMismatch) {
			return false, nil
		}
		return false, err
	}
	_ = foot
	return true, nil
}

// tarSize 计算 tar 部分长度（不含签名区）
func tarSize(f *os.File, foot *Footer) (int64, error) {
	stat, err := f.Stat()
	if err != nil {
		return 0, err
	}
	size := stat.Size() - int64(foot.SigLen) - int64(foot.MetaLen) - FooterSize
	if size <= 0 {
		return 0, ErrBadStructure
	}
	return size, nil
}

// computeDigestStream 流式计算 tar 部分摘要，避免大文件 OOM
func computeDigestStream(f *os.File, size int64, algo string) ([]byte, error) {
	var h hash.Hash
	switch algo {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	case "sm3":
		h = sm3.New()
	default:
		return nil, fmt.Errorf("不支持的摘要算法: %s", algo)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err := io.Copy(h, io.LimitReader(f, size)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// ReadSignatureAndMeta 读取签名和元数据，同时返回原始 metadata 字节（供 v2 验签使用）
func ReadSignatureAndMeta(f *os.File, foot *Footer) (sig []byte, meta *Metadata, metaRaw []byte, err error) {
	stat, _ := f.Stat()
	start := stat.Size() - FooterSize - int64(foot.MetaLen) - int64(foot.SigLen)

	_, err = f.Seek(start, io.SeekStart)
	if err != nil {
		return nil, nil, nil, err
	}

	sig = make([]byte, foot.SigLen)
	if _, err = io.ReadFull(f, sig); err != nil {
		return nil, nil, nil, err
	}

	metaBytes := make([]byte, foot.MetaLen)
	if _, err = io.ReadFull(f, metaBytes); err != nil {
		return nil, nil, nil, err
	}

	meta = &Metadata{}
	if err = json.Unmarshal(metaBytes, meta); err != nil {
		return nil, nil, nil, fmt.Errorf("解析元数据失败: %w", err)
	}
	return sig, meta, metaBytes, nil
}

// Info 返回签名信息
func Info(path string) (*Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	foot, err := ParseFooter(f)
	if err != nil {
		if errors.Is(err, ErrNoFooter) || errors.Is(err, ErrMagicMismatch) {
			return nil, ErrNoFooter
		}
		return nil, err
	}

	_, meta, _, err := ReadSignatureAndMeta(f, foot)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

// GetTarDigest 获取 tar 部分的摘要（hex）。若文件已签名则从 metadata 取，否则计算整个文件
func GetTarDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	foot, err := ParseFooter(f)
	if err != nil {
		if errors.Is(err, ErrNoFooter) || errors.Is(err, ErrMagicMismatch) {
			// 未签名，计算整个文件
			_, _ = f.Seek(0, io.SeekStart)
			h := sha256.New()
			if _, err = io.Copy(h, f); err != nil {
				return "", err
			}
			return hex.EncodeToString(h.Sum(nil)), nil
		}
		return "", err
	}

	_, meta, _, err := ReadSignatureAndMeta(f, foot)
	if err != nil {
		return "", err
	}
	// tar_digest 格式为 "algo:hex"（如 sha256:xxx, sha512:xxx, sm3:xxx）
	if idx := strings.Index(meta.TarDigest, ":"); idx >= 0 && idx+1 < len(meta.TarDigest) {
		return meta.TarDigest[idx+1:], nil
	}
	return meta.TarDigest, nil
}

// ComputeBinaryHashes 计算 tar 内指定关键二进制文件的 hash，并返回 path -> "hex" 映射。
// - paths 形如 "/bin/bash"、"/bin/ls" 等，函数会自动去掉前导 "/" 或 "./" 与 tar 内路径匹配。
// - 二进制文件哈希固定使用 md5，返回值中不包含 "md5:" 前缀，与整体签名的 hashAlgo 解耦。
// - 若某个路径在 tar 中不存在，则不会出现在返回 map 中（同时记录日志由调用方自行处理）。
func ComputeBinaryHashes(tarPath string, hashAlgo HashAlgo, protectedPaths []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(protectedPaths) == 0 {
		return result, nil
	}

	f, err := os.Open(tarPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// 计算 tar 数据区大小（排除签名区），兼容未签名文件
	var tarLen int64
	if foot, err := ParseFooter(f); err == nil {
		tarLen, err = tarSize(f, foot)
		if err != nil {
			return nil, err
		}
	} else if errors.Is(err, ErrNoFooter) || errors.Is(err, ErrMagicMismatch) {
		tarLen = stat.Size()
	} else {
		return nil, err
	}

	// 二进制文件 hash 固定使用 md5，与签名使用的摘要算法解耦
	h := md5.New()

	// 预处理待保护路径：构建 tar 内候选路径 -> 原始路径 映射
	targets := make(map[string]string)
	for _, p := range protectedPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		orig := p
		// 统一归一化：去掉前导 "/"、"./"，并使用 path.Clean 规整路径，
		// 确保与 tar 归档中的 "./bin/bash"、"bin/bash" 等形式都能匹配。
		p = strings.TrimPrefix(p, "/")
		p = strings.TrimPrefix(p, "./")
		p = path.Clean(p)
		if p == "" || p == "." {
			continue
		}
		targets[p] = orig
	}
	if len(targets) == 0 {
		return result, nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	tr := tar.NewReader(io.LimitReader(f, tarLen))

	found := make(map[string]bool)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}

		// 对 tar 头部名称做与待保护路径一致的归一化处理
		name := strings.TrimPrefix(hdr.Name, "/")
		name = strings.TrimPrefix(name, "./")
		name = path.Clean(name)
		if name == "" || name == "." {
			// 理论上不会命中受保护文件，直接丢弃内容
			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return nil, err
			}
			continue
		}

		orig, ok := targets[name]
		if !ok {
			// 不在受保护列表中，直接跳过内容
			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				return nil, err
			}
			continue
		}

		// 对该文件内容计算 hash（固定为 md5）
		h.Reset()
		if _, err := io.Copy(h, tr); err != nil {
			return nil, err
		}
		digestHex := hex.EncodeToString(h.Sum(nil))
		// 按约定，这里仅返回 md5 的十六进制串，不带 "md5:" 前缀
		result[orig] = digestHex
		found[orig] = true

		// 全部找到可以提前退出
		if len(found) == len(targets) {
			break
		}
	}

	return result, nil
}

// VerifyFromFile 使用公钥文件验签
func VerifyFromFile(path, keyPath string) error {
	pubKey, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("读取公钥失败: %w", err)
	}
	return Verify(path, pubKey)
}

// Verify 使用公钥验签。pubKey 为 PEM 格式的 RSA 公钥或 X.509 证书
// 使用流式计算摘要，避免大文件 OOM
func Verify(path string, pubKey []byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	foot, err := ParseFooter(f)
	if err != nil {
		return err
	}

	sig, meta, metaRaw, err := ReadSignatureAndMeta(f, foot)
	if err != nil {
		return err
	}

	size, err := tarSize(f, foot)
	if err != nil {
		return err
	}

	// 第一阶段：计算 tar 部分摘要（digest1），并与 metadata 中记录的 tar_digest 比对
	digest1, err := computeDigestStream(f, size, meta.HashAlgo)
	if err != nil {
		return err
	}

	expectedDigest := meta.HashAlgo + ":" + hex.EncodeToString(digest1)
	if expectedDigest != meta.TarDigest {
		return ErrDigestMismatch
	}

	// 第二阶段：根据版本选择签名输入：
	// - v1：只对 digest1 做签名
	// - v2：对 H(digest1 || metaRaw) 做签名（metaRaw 为落盘 JSON 字节，保证与签名时一致）
	var signedDigest []byte
	if foot.Version <= VersionV1 || meta.Version <= VersionV1 {
		// 兼容老版本：签名对象仅为 tar 摘要
		signedDigest = digest1
	} else {
		h2, err := selectHash(meta.HashAlgo)
		if err != nil {
			return err
		}
		_, _ = h2.Write(digest1)
		_, _ = h2.Write(metaRaw)
		signedDigest = h2.Sum(nil)
	}

	if err := verifySignature(signedDigest, sig, pubKey, meta.SigAlgo); err != nil {
		return ErrVerifyFailed
	}
