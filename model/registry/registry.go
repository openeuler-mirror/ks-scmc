package registry

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"

	"scmc/common"
)

// pushing with the prefix, *docker images* shows simplified repo name after pull.
const (
	DefaultRepository = "library"
	DefaultTag        = "latest"
	ManifestFileName  = "manifest.json"
)

func url() string {
	if common.Config.Registry.Secure {
		return "https://" + common.Config.Registry.Addr
	} else {
		return "http://" + common.Config.Registry.Addr
	}
}

func username() string {
	return common.Config.Registry.Username
}

func password() string {
	return common.Config.Registry.Password
}

type client struct {
	*registry.Registry
}

func NewClient() (*client, error) {
	return newClient()
}

func newClient() (*client, error) {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if common.Config.Registry.Secure {
		t.TLSHandshakeTimeout = 10 * time.Second
	} else {
		t.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rt := registry.WrapTransport(t, url(), username(), password())
	url := strings.TrimSuffix(url(), "/")
	r := &registry.Registry{
		URL: url,
		Client: &http.Client{
			Transport: rt,
		},
		Logf: log.Debugf,
	}

	if err := r.Ping(); err != nil {
		return nil, err
	}

	return &client{r}, nil
}

// Copy from "github.com/heroku/docker-registry-client/registry"
func (r *client) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

// Override: add request header
func (r *client) ManifestDigest(repository, reference string) (digest.Digest, error) {
	url := r.url("/v2/%s/manifests/%s", repository, reference)
	r.Logf("registry.manifest.head url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", schema2.MediaTypeManifest)
	resp, err := r.Client.Do(req)
	if err != nil {
		return "", err
	}

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	return digest.Parse(resp.Header.Get("Docker-Content-Digest"))
}

type fileInfo struct {
	filePath  string
	sha256sum string
	fileSize  int64
}

type Manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

type ClientSession struct {
	Repo       string
	Tag        string
	TarFile    string
	Hub        *client
	Manifestv2 *schema2.DeserializedManifest
}

func IsImageExist(repoTag string) (bool, error) {
	strs := strings.Split(repoTag, ":")
	if len(strs) != 2 {
		return false, nil
	}

	repo, tag := filepath.Join(DefaultRepository, strs[0]), strs[1]
	cli, err := newClient()
	if err != nil {
		log.Warnf("create registry client err=%v", err)
		return false, err
	}

	tags, err := cli.Tags(repo)
	if err != nil {
		log.Warnf("get tags of repo=%v err=%v", repo, err)
		return false, err
	}

	for _, t := range tags {
		if t == tag {
			return true, nil
		}
	}

	return false, nil
}

func getFileInfo(path string) (*fileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &fileInfo{
		filePath:  path,
		sha256sum: hex.EncodeToString(hash.Sum(nil)),
		fileSize:  info.Size(),
	}, nil
}

func RemoveRegistryImage(repoTag string) error {
	hub, err := newClient()
	if err != nil {
		log.Warnf("connect to registry [%v][%v][%v], err=%v", url(), username(), password(), err)
		return err
	}

	n := strings.IndexRune(repoTag, ':')
	var repo, tag string
	if n > -1 {
		repo = repoTag[0:n]
		tag = repoTag[n+1:]
	} else {
		log.Warnf("image %v format err", repoTag)
		return fmt.Errorf("invalid image repotag=%s", repoTag)
	}

	d, err := hub.ManifestDigest(repo, tag)
	if err != nil {
		return err
	}

	err = hub.DeleteManifest(repo, d)
	if err != nil {
		log.Warnf("registry client DeleteManifest image=%s digest=%s err=%v", repoTag, d, err)
		return err
	}

	return nil
}
