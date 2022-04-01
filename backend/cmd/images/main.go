package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"unsafe"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"scmc/model"
)

const imageHeadLen = len("sha256:")

func GetHash(filename, sendHashStr string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("open file %v err: %v", filename, err)
		return err
	}
	defer file.Close()

	hash256 := sha256.New()
	_, err = io.Copy(hash256, file)
	if err != nil {
		log.Printf("io copy error: %v", err)
		return err
	}

	hashStr := hex.EncodeToString(hash256.Sum(nil))
	if hashStr != sendHashStr {
		log.Printf("file hash not match: [%v] != [%v]", hashStr, sendHashStr)
		return errors.New("file hash not match")
	}

	return nil
}

func ImportImages() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("try to connect to container daemon: %v", err)
		return
	}

	imageInfo, err := model.QueryImageByStatus()
	if err != nil {
		log.Printf("get file path: %v", err)
		return
	}

	list, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Printf("ImageList: %v", err)
		return
	}

	var imageIds []string
	for _, image := range list {
		imageIds = append(imageIds, image.ID[imageHeadLen:])
		//log.Printf("image list:[%v][%v][%v][%v]", image.ID, image.RepoDigests, image.RepoTags, len(image.ID))
	}

	sort.Strings(imageIds)
	for _, imageId := range imageIds {
		log.Printf("imageId:[%v]", imageId)
	}

	for _, info := range imageInfo {
		log.Printf("image info:[%v][%v][%v][%v][%v]", info.Name, info.Version, info.FilePath, info.CheckSum, info.ImageId)

		index := sort.SearchStrings(imageIds, info.ImageId)
		if index < len(imageIds) && imageIds[index] == info.ImageId {
			log.Printf("[%v] equal [%v]", imageIds[index], info.ImageId)
			return
		}

		if err = GetHash(info.FilePath, info.CheckSum); err != nil {
			tmpErr := model.RemoveImage([]int64{info.ID})
			log.Printf("image file %v err and remove, remove result:%v", info.FilePath, tmpErr)
			return
		}

		file, err := os.Open(info.FilePath)
		if err != nil {
			log.Printf("open file %v err: %v", info.FilePath, err)
			return
		}
		defer file.Close()

		result, err := cli.ImageLoad(context.Background(), file, false)
		if err != nil {
			log.Printf("ImageLoad: %v err %v", result, err)
			return
		}

		defer result.Body.Close()
		bs, err := ioutil.ReadAll(result.Body)
		if err != nil {
			log.Printf("read result body: %v", err)
			return
		}

		bsStr := *(*string)(unsafe.Pointer(&bs))
		shaIndex := strings.Index(bsStr, "sha256:")
		digests := bsStr[shaIndex : shaIndex+71]
		tagStr := fmt.Sprintf("%s:%s", info.Name, info.Version)

		err = cli.ImageTag(context.Background(), digests, tagStr)
		if err != nil {
			log.Printf("call ImageTag err: %v", err)
			return
		}

		log.Printf("load image [%v] success, and set tag [%v] to [%v]", info.FilePath, tagStr, digests)
	}
}

func main() {
	flag.Parse()
	ImportImages()
}
