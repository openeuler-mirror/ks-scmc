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
