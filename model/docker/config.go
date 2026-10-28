package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	pb "scmc/rpc/pb/container"
	"strconv"
	"strings"
	"time"
)

var dockerRootDir string

const (
	NoneBit  = 0
	EnvBit   = 1
	MountBit = 2
)

var setters = map[string]string{
	"disk_wio": "blkio.throttle.write_iops_device",
	"disk_rio": "blkio.throttle.read_iops_device",
	"disk_rbw": "blkio.throttle.read_bps_device",
	"disk_wbw": "blkio.throttle.write_bps_device",
}

func getDockerRootDir() (string, error) {
	if dockerRootDir != "" {
		return dockerRootDir, nil
	}

	cli, err := Client()
	if err != nil {
		log.Errorf("docker.Client err:%s", err)
		return "", err
	}

	info, err := cli.Info(context.Background())
	if err != nil {
		log.Errorf("cli.Info err:%s", err)
		return "", err
	}

	dockerRootDir = info.DockerRootDir
	log.Infof("DockerRootDir:%v", dockerRootDir)
	return dockerRootDir, nil
}

func parseConfigFile(fileName string) (map[string]interface{}, error) {
	dataBuf := make(map[string]interface{})
	buffer, err := readConfigFile(fileName)
	if err != nil {
		return dataBuf, err
	}
	if err = json.Unmarshal([]byte(buffer), &dataBuf); err != nil {
		log.Errorf("Json解析失败！error info:%v, buffer:%+v", err, string(buffer))
		return dataBuf, err
	}

	return dataBuf, nil
}

func readConfigFile(fileName string) ([]byte, error) {
	var data []byte
	rfile, err := os.Open(fileName)
	if err != nil {
		log.Errorf("open file err: %v", err)
		return data, err
	}
	defer rfile.Close()

	fileInfo, err := rfile.Stat()
	fileSize := fileInfo.Size()
	reader := bufio.NewReader(rfile)
	buffer := make([]byte, fileSize)
	if _, err := reader.Read(buffer); err != nil {
		log.Errorf("read file %v err: %v", fileName, err)
		return data, err
	}

	return buffer, nil
}

func writeConfigFile(fileName string, dataBuf map[string]interface{}) error {
	data, err := json.Marshal(dataBuf)
	//data, err := json.MarshalIndent(dataBuf, "", "\t")
	if err != nil {
		log.Errorf("json err: %v", err)
		return err
	}

	wfile, err := os.Create(fileName)
	if err != nil {
		log.Errorf("cannot create file: %v", fileName)
		return err
	}
	defer wfile.Close()

	_, err = wfile.Write(data)
	if err != nil {
		log.Errorf("cannot write json data to file: %v", err)
		return err
	}

	buffer, _ := readConfigFile(fileName)
	log.Tracef("read new file: %+v", string(buffer))
	return nil
}

func modifyEnvs(dataBuf map[string]interface{}, envs []string) {
	if data, ok := dataBuf["Config"].(map[string]interface{}); ok {
		if env, ok := data["Env"]; ok {
			log.Debugf("old env:%+v, new env:%+v", env, envs)
			dataBuf["Config"].(map[string]interface{})["Env"] = envs
		}
	}
}

func modifyMounts(dataBuf map[string]interface{}, mounts map[string]*pb.Mount) {
	data, ok := dataBuf["MountPoints"].(map[string]interface{})
	log.Debugf("old mounts:%+v, new mounts:%+v", data, mounts)
	if ok {
		var deleteKey []string
		for k, v := range data {
			// k不在新数据内
			if _, ok := mounts[k]; !ok {
				deleteKey = append(deleteKey, k)
			} else {
				// 替换value
				if value, ok := v.(map[string]interface{}); ok {
					value["RW"] = !mounts[k].ReadOnly
					value["Source"] = mounts[k].Source
					value["Type"] = mounts[k].Type
					if spec, ok := value["Spec"].(map[string]interface{}); ok {
						spec["ReadOnly"] = mounts[k].ReadOnly
						spec["Source"] = mounts[k].Source
						spec["Type"] = mounts[k].Type
					}
					data[k] = value
				}
			}
		}

		// 删除新数据内没有的key
		for _, k := range deleteKey {
			delete(data, k)
		}

		for k, v := range mounts {
			// 新增
			if _, ok := data[k]; !ok {
				spec := make(map[string]interface{})
				spec["ReadOnly"] = v.ReadOnly
				spec["Source"] = v.Source
				spec["Type"] = v.Type
				spec["Target"] = k
				m := make(map[string]interface{})
				m["RW"] = !v.ReadOnly
				m["Source"] = v.Source
				m["Type"] = v.Type
				m["Destination"] = k
				m["Spec"] = spec
				m["SkipMountpointCreation"] = true
				m["Propagation"] = "rprivate"
				m["Driver"] = ""
				m["Name"] = ""
				data[k] = m
			}
		}
	}
}

func ModifyContainerConfig(containerId string, which int, envs []string, mounts map[string]*pb.Mount) error {
	dir, err := getDockerRootDir()
	if err != nil {
		return err
	}

	fileName := dir + "/containers/" + containerId + "/config.v2.json"
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return err
	}

	log.Infof("fileName:%v", fileName)
	dataBuf, err := parseConfigFile(fileName)
	if err != nil {
		return err
	}

	// modify envs
	if which&EnvBit != 0 {
		log.Debugf("modify %v envs", containerId)
		modifyEnvs(dataBuf, envs)
	}

	// modify mounts
	if which&MountBit != 0 {
		log.Debugf("modify %v mounts", containerId)
		modifyMounts(dataBuf, mounts)
	}

	if err := writeConfigFile(fileName, dataBuf); err != nil {
		return err
	}

	cmd := exec.Command("/bin/bash", "-c", "systemctl restart docker; sleep 1")
	err = cmd.Run()
	log.Debugf("CMD[%s]", cmd)
	if err != nil {
		log.Warnf("CMD[%s] error=%v", cmd, err)
	}

	return err
}

func CompareMounts(src, dst []*pb.Mount) (map[string]*pb.Mount, bool) {
	srcMap := make(map[string]*pb.Mount)
	dstMap := make(map[string]*pb.Mount)
	for _, v := range src {
		if v != nil {
			srcMap[v.Target] = v
		}
	}

	for _, v := range dst {
		if v != nil {
			dstMap[v.Target] = v
		}
	}

	if len(srcMap) != len(dstMap) {
		return dstMap, false
	}

	// compare key
	for k, _ := range srcMap {
		if _, ok := dstMap[k]; !ok {
			return dstMap, false
		}
	}

	//  compare value
	for k, v := range srcMap {
		m := dstMap[k]
		if v != nil && m != nil {
			if v.Source != m.Source || v.Target != m.Target || v.ReadOnly != m.ReadOnly || v.Type != m.Type {
				return dstMap, false
			}
		} else if v != m { // nil -- !nil
			return dstMap, false
		}
	}

	return nil, true
}
