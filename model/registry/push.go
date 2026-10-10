package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/pkg/archive"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"

	"scmc/model/db"
)

// PushImage 初始化仓库客户端，并push image
func PushImage(info *db.ImageInfo) {
	hub, err := newClient()
	if err != nil {
		log.Errorf("connect to registry %s err=%v", url(), err)
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

	p := ClientSession{
		Repo:    info.Name,
		Tag:     info.Version,
		TarFile: info.FilePath,
		Hub:     hub,
	}
	p.push()
}

// CheckImagePush 查询仓库与数据库数据，push仓库中缺少的image
func CheckImagePush() {
	imageInfo, err := db.QueryImageByStatus()
	if err != nil {
		log.Warnf("get file path: %v", err)
		return
	}

	hub, err := newClient()
	if err != nil {
		log.Warnf("connect to registry [%v][%v][%v], err=%v", url(), username(), password(), err)
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
		k := filepath.Join(DefaultRepository, info.Name) + ":" + info.Version
		if _, ok := images[k]; !ok {
			log.Infof("[%v] is not in registry, now push[%v]", k, info.FilePath)
			p := ClientSession{
				Repo:    info.Name,
				Tag:     info.Version,
				TarFile: info.FilePath,
				Hub:     hub,
			}
			p.push()
		}
	}

	log.Debugf("CheckImagePush end")
}

// push 解压image归档文件，根据manifest.json进行push
func (m *ClientSession) push() {
	// create tmp dir for tar unpack
	dir := fmt.Sprintf("%s:%s-%d", m.Repo, m.Tag, time.Now().UnixNano())
	tmpDir, err := ioutil.TempDir(os.TempDir(), dir)
	if err != nil {
		log.Warnf("ioutil.TempDir %s err=%v", tmpDir, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// unpack tar file
	arch := archive.NewDefaultArchiver()
	if err := arch.UntarPath(m.TarFile, tmpDir); err != nil {
		log.Warnf("Untar src=%s dst=%s err=%v", m.TarFile, tmpDir, err)
		return
	}

	manifests, err := m.parseManifest(filepath.Join(tmpDir, ManifestFileName))
	if err != nil || len(manifests) < 1 {
		log.Warnf("parseManifest dir=%s err=%v manifests=%v", tmpDir, err, manifests)
		return
	}
	if err := m.handle(tmpDir, manifests[0]); err != nil {
		log.Warnf("handle err: %s", err)
	}
}

// handle 根据manifest.json内容依次上传image每一层，上传image配置信息
func (m *ClientSession) handle(baseDir string, man *Manifest) error {
	layers := make([]*fileInfo, 0, len(man.Layers))
	for _, l := range man.Layers {
		f, err := getFileInfo(filepath.Join(baseDir, l))
		if err != nil {
			return err
		}

		if err := m.uploadBlob(f); err != nil {
			return err
		}

		layers = append(layers, f)
	}

	config, err := getFileInfo(filepath.Join(baseDir, man.Config))
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

	log.Debug("push image successfully")
	return nil
}

func (m *ClientSession) fullRepo() string {
	return filepath.Join(DefaultRepository, m.Repo)
}

func (m *ClientSession) parseManifest(path string) ([]*Manifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifests []*Manifest
	if err := json.Unmarshal(data, &manifests); err != nil {
		return nil, err
	}

	return manifests, err
}

func (m *ClientSession) uploadBlob(f *fileInfo) error {
	d := digest.NewDigestFromHex(string(digest.SHA256), f.sha256sum)
	exists, err := m.Hub.HasBlob(m.fullRepo(), d)
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
		return m.Hub.UploadBlob(m.fullRepo(), d, stream)
	}

	return nil
}

func (m *ClientSession) uploadManifest(layers []*fileInfo, config *fileInfo) error {
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

	dm, err := schema2.FromStruct(man)
	if err != nil {
		log.Warnf("create manifest struct err=%v", err)
		return err
	}

	return m.Hub.PutManifest(m.fullRepo(), m.Tag, dm)
}
