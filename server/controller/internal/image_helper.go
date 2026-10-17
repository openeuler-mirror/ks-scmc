package internal

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/openpgp"

	"scmc/common"
	"scmc/model/registry"
)

func imageDir() string {
	return common.Config.Controller.ImageDir
}

func imageSigner() string {
	return common.Config.Controller.ImageSigner
}

func imageMaxSize() int64 {
	return int64(common.Config.Controller.ImageMaxSize)
}

type ImageFileStream interface {
	Read() ([]byte, error)
	Context() context.Context
}

type ImageFile struct {
	f        *os.File
	size     int64 // max size
	wsize    int64 // size writed of image file
	sha2     hash.Hash
	dockerID string
}

func NewImageFile(name, version string, size int64) (*ImageFile, error) {
	f, err := ioutil.TempFile(imageDir(), fmt.Sprintf("%s_%s_*.tar", name, version))
	if err != nil {
		log.Warnf("Create file err=%v", err)
		return nil, err
	}

	return &ImageFile{
		size:  size,
		f:     f,
		wsize: 0,
		sha2:  sha256.New(),
	}, nil
}

func (i *ImageFile) Path() string {
	if i.f != nil {
		return i.f.Name()
	}
	return ""
}

func (i *ImageFile) Size() int64 {
	return i.wsize
}

func (i *ImageFile) ReadStream(s ImageFileStream) error {
	for {
		if err := s.Context().Err(); err != nil {
			log.Errorf("Context err=%v", err)
			return err
		}

		b, err := s.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Errorf("Receive chunk data err=%v", err)
			return err
		}

		i.wsize += int64(len(b))
		if i.wsize > i.size || i.wsize > imageMaxSize() {
			log.Errorf("Image too large, size=%v", i.wsize)
			return fmt.Errorf("Image too large, size=%v, expected=%v", i.wsize, i.size)
		}

		if _, err = i.f.Write(b); err != nil {
			log.Errorf("Write image file=%v err=%v", i.f, err)
			return err
		}

		if _, err = i.sha2.Write(b); err != nil {
			log.Errorf("Write image data to sha256 hash calculator, err=%v", err)
			return err
		}
	}

	return i.parseDockerID()
}

func (i *ImageFile) VerifySignature(signature []byte) (bool, error) {
	if _, err := i.f.Seek(0, 0); err != nil {
		log.Warnf("File=%+v seek to position 0, err=%v", i.f, err)
		return false, err
	}

	kr, err := os.Open(imageSigner())
	if err != nil {
		log.Warnf("Open GPG keyring file=%v err=%v", imageSigner(), err)
		return false, err
	}
	defer kr.Close()

	k, err := openpgp.ReadKeyRing(kr)
	if err != nil {
		log.Warnf("Read GPG keyring err=%v", err)
		return false, err
	}

	_, err = openpgp.CheckDetachedSignature(k, i.f, bytes.NewBuffer(signature))
	if err != nil {
		log.Infof("Check signature tar=%v err=%v", i.f, err)
		return false, nil
	}

	return true, nil
}

func (i *ImageFile) parseDockerID() error {
	if _, err := i.f.Seek(0, 0); err != nil {
		log.Warnf("File=%+v seek to position 0, err=%v", i.f, err)
		return err
	}

	var hasManifest, hasImageID bool

	r := tar.NewReader(i.f)
	for h, err := r.Next(); err != io.EOF; h, err = r.Next() {
		if err != nil {
			log.Warnf("Read tar file=%+v err=%v", i.f, err) // image tar file header
			return err
		}

		fname := h.FileInfo().Name()
		if fname == registry.ManifestFileName {
			hasManifest = true
			continue
		}

		if index := strings.Index(fname, ".json"); index == -1 {
			continue
		} else {
			i.dockerID = fname[0:index]
			hasImageID = true
		}

		if hasManifest && hasImageID {
			return nil
		}
	}

	if !hasManifest {
		return errors.New("镜像缺少manifest.json")
	} else if !hasImageID {
		return errors.New("无法解析镜像ID")
	}
	return nil
}

func (i *ImageFile) SHA256() string {
	return hex.EncodeToString(i.sha2.Sum(nil))
}

func (i *ImageFile) DockerID() string {
	return i.dockerID
}

func (i *ImageFile) Close() {
	if i.f != nil {
		i.f.Close()
	}
}

func (i *ImageFile) Destroy() {
	if i.f != nil {
		i.f.Close()
		err := os.Remove(i.f.Name())
		log.Infof("Remove image tar file=%v, err=%v", i.f, err)
	}
}
