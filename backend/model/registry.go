package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"scmc/common"
	"strings"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

// pushing with the prefix, *docker images* shows simplified repo name after pull.
const imageRepoPrefix = "library/"

func registryUrl() string {
	if common.Config.Registry.Secure {
		return "https://" + common.Config.Registry.Addr
	} else {
		return "http://" + common.Config.Registry.Addr
	}
}

func registryAddr() string {
	return common.Config.Registry.Addr
}

func registryUsername() string {
	return common.Config.Registry.Username
}

func registryPassword() string {
	return common.Config.Registry.Password
}

type registryClient struct {
	*registry.Registry
}

func newRegistryClient() (*registryClient, error) {
	createFunc := registry.NewInsecure
	if common.Config.Registry.Secure {
		createFunc = registry.New
	}

	r, err := createFunc(registryUrl(), registryUsername(), registryPassword())
	if err != nil {
		return nil, err
	}

	cli := &registryClient{r}
	cli.Logf = log.Debugf
	return cli, nil
}

// Copy from "github.com/heroku/docker-registry-client/registry"
func (r *registryClient) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

// Override: add request header
func (r *registryClient) ManifestDigest(repository, reference string) (digest.Digest, error) {
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

type manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

type imagePush struct {
	repo    string
	tag     string
	tarFile string
	hub     *registryClient
}

func (m *imagePush) fullRepo() string {
	return imageRepoPrefix + m.repo
}

func (m *imagePush) parseManifest(path string) ([]*manifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifests []*manifest
	if err := json.Unmarshal(data, &manifests); err != nil {
		return nil, err
	}

	return manifests, err
}

func (m *imagePush) uploadBlob(f *fileInfo) error {
	d := digest.NewDigestFromHex(string(digest.SHA256), f.sha256sum)
	exists, err := m.hub.HasBlob(m.fullRepo(), d)
	if err != nil {
		log.Infof("HasBlob repo=%s err=%v", m.fullRepo(), err)
		return err
	}

	if !exists {
		stream, err := os.Open(f.filePath)
		if err != nil {
			log.Infof("open file=%s %v", f.filePath, err)
			return err
		}

		defer stream.Close()
		return m.hub.UploadBlob(m.fullRepo(), d, stream)
	}

	return nil
}

func (m *imagePush) uploadManifest(layers []*fileInfo, config *fileInfo) error {
	var man schema2.Manifest
	man.SchemaVersion = schema2.SchemaVersion.SchemaVersion
	man.MediaType = schema2.SchemaVersion.MediaType
	man.Config = distribution.Descriptor{
		MediaType: schema2.MediaTypeImageConfig,
		Size:      config.fileSize,
		Digest:    digest.NewDigestFromHex(string(digest.SHA256), config.sha256sum),
	}

	for _, l := range layers {
		man.Layers = append(man.Layers, distribution.Descriptor{
			MediaType: schema2.MediaTypeUncompressedLayer,
			Size:      l.fileSize,
			Digest:    digest.NewDigestFromHex(string(digest.SHA256), l.sha256sum),
		})
	}

	man_, err := schema2.FromStruct(man)
	if err != nil {
		log.Warnf("create manifest struct err=%v", err)
		return err
	}

	return m.hub.PutManifest(m.fullRepo(), m.tag, man_)
}

func (m *imagePush) handle(baseDir string, man *manifest) error {
	// TODO ignore RepoTags in manifest
	if len(man.RepoTags) < 1 {
		log.Warnf("manifest:%v has no repo tags", man)
	}

	layers := make([]*fileInfo, 0, len(man.Layers))
	for _, l := range man.Layers {
		path := baseDir + "/" + l
		f, err := getFileInfo(path)
		if err != nil {
			return err
		}

		if err := m.uploadBlob(f); err != nil {
			return err
		}

		layers = append(layers, f)
	}

	config, err := getFileInfo(baseDir + "/" + man.Config)
	if err != nil {
		log.Warnf("get config file info err=%v", err)
		return err
	}

	if err := m.uploadBlob(config); err != nil {
		return err
	}

	if err := m.uploadManifest(layers, config); err != nil {
		return err
	}

	return nil
}

func (m *imagePush) push() {
	// create tmp dir for tar unpack
	dir := fmt.Sprintf("%s:%s-%d", m.repo, m.tag, time.Now().UnixNano())
	tmpDir, err := ioutil.TempDir(os.TempDir(), dir)
	if err != nil {
		log.Warnf("ioutil.TempDir %s err=%v", tmpDir, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// unpack tar file
	arch := archive.NewDefaultArchiver()
	if err := arch.UntarPath(m.tarFile, tmpDir); err != nil {
		log.Warnf("Untar src=%s dst=%s err=%v", m.tarFile, tmpDir, err)
		return
	}

	manifests, err := m.parseManifest(tmpDir + "/manifest.json")
	if err != nil {
		log.Warnf("parseManifest dir=%s err=%v", tmpDir, err)
		return
	}
	if len(manifests) < 1 {
		log.Warnf("no valid manifest")
		return
	}
	m.handle(tmpDir, manifests[0])
}

func pullImage(cli *client.Client, repoTag string) (io.ReadCloser, error) {
	var auth = types.AuthConfig{
		Username:      registryUsername(),
		Password:      registryPassword(),
		ServerAddress: registryAddr(),
	}

	encodedAuth, err := command.EncodeAuthToBase64(auth)
	if err != nil {
		return nil, err
	}

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	// defer cancel()
	ctx := context.Background()

	return cli.ImagePull(ctx, repoTag, types.ImagePullOptions{
		All:          false,
		RegistryAuth: encodedAuth,
	})
}

func IsImageExist(repoTag string) (bool, error) {
	strs := strings.Split(repoTag, ":")
	if len(strs) != 2 {
		return false, nil
	}

	repo, tag := imageRepoPrefix+strs[0], strs[1]
	cli, err := newRegistryClient()
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

func PullImage(repoTag string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Errorf("try to connect to container daemon: %v", err)
		return err
	}

	reader, err := pullImage(cli, repoTag)
	if err != nil {
		log.Errorf("pullImage %v", err)
		return err
	}

	defer reader.Close()
	if _, err := ioutil.ReadAll(reader); err != nil {
		log.Errorf("readall %v", err)
		return err
	}
	log.Debugf("pull %v done", repoTag)

	return nil
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

func PushImage(info *ImageInfo) {
	hub, err := newRegistryClient()
	if err != nil {
		log.Errorf("connect to registry %s err=%v", registryUrl(), err)
		return
	}

	repos, _ := hub.Repositories()
	for _, repo := range repos {
		tags, _ := hub.Tags(repo)
		for _, tag := range tags {
			if repo == info.Name && tag == info.Version {
				log.Warnf("[%v]:[%v] is exist", repo, tag)
				return
			}
		}
	}

	p := imagePush{
		repo:    info.Name,
		tag:     info.Version,
		tarFile: info.FilePath,
		hub:     hub,
	}
	p.push()
}

func PushImgae() {
	for {
		CheckImagePush()
		time.Sleep(time.Minute)
	}
}

func CheckImagePush() {
	imageInfo, err := QueryImageByStatus()
	if err != nil {
		log.Warnf("get file path: %v", err)
		return
	}

	hub, err := newRegistryClient()
	if err != nil {
		log.Warnf("connect to registry [%v][%v][%v], err=%v", registryUrl(), registryUsername(), registryPassword(), err)
		return
	}

	images := make(map[string]struct{})
	repositories, err := hub.Repositories()
	if err != nil {
		log.Warnf("get repositories err:%v", err)
		return
	}
	for _, repository := range repositories {
		tags, _ := hub.Tags(repository)
		for _, tag := range tags {
			k := repository + ":" + tag
			images[k] = struct{}{}
		}
	}

	for _, info := range imageInfo {
		k := imageRepoPrefix + info.Name + ":" + info.Version
		if _, ok := images[k]; !ok {
			log.Infof("[%v] is not in registry, now push[%v]", k, info.FilePath)
			p := imagePush{
				repo:    info.Name,
				tag:     info.Version,
				tarFile: info.FilePath,
				hub:     hub,
			}
			p.push()
		}
	}

	log.Debugf("CheckImagePush end")
}

func RemoveRegistryImage(repoTag string) error {
	hub, err := newRegistryClient()
	if err != nil {
		log.Warnf("connect to registry [%v][%v][%v], err=%v", registryUrl(), registryUsername(), registryPassword(), err)
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
