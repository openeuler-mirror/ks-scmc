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
