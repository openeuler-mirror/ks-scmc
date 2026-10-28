package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	pb "scmc/rpc/pb/container"

	log "github.com/sirupsen/logrus"
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
func SetContainerDiskLimit(key, containerID string, value string) error {
	file, ok := setters[key]
	if !ok {
		return errors.New("invalid key: " + key)
	}
	var numValue int64
	var err error
	switch {
	case key == "disk_rbw" || key == "disk_wbw":
		var num float64
		num, err = strconv.ParseFloat(value, 64)
		num = math.Round(num * 1024 * 1024)
		numValue = int64(num)
	case key == "disk_rio" || key == "disk_wio":
		var num int64
		num, err = strconv.ParseInt(value, 10, 32)
		numValue = num
	default:
		return errors.New("unsupported key type")
	}
	if err != nil {
		return fmt.Errorf("invalid value %s for key %s: %v", value, key, err)
	}

	device, err := findProcessDevice(containerID)
	if err != nil {
		return err
	}

	path := filepath.Join("/sys/fs/cgroup/blkio/docker", containerID, file)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("cgroup path not found")
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load cgroup file $s failed: %v", file, err)
	}
	lines := strings.Split(string(data), "\n")
	newLines := make([]string, 0)
	found := false
	entry := fmt.Sprintf("%s %d", device, value)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // 跳过空行
		}
		if strings.HasPrefix(line, device) {
			found = true
			if numValue != 0 {
				// 如果值非0，替换原有行
				newLines = append(newLines, entry)
			}
			// 如果值为0，跳过（即删除）
		} else {
			// 保留其他设备的设置
			newLines = append(newLines, line)
		}
	}
	// 如果未找到且值非0，追加新行
	if !found && numValue != 0 {
		newLines = append(newLines, entry)
	}
	// 写入文件
	err = ioutil.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write to the file %s: %v", file, err)
	}
	return nil
}
func findProcessDevice(containerID string) (string, error) {
	// 通过容器进程的cgroup信息查找
	cgroupPath := filepath.Join("/sys/fs/cgroup/blkio/docker", containerID, "blkio.throttle.io_serviced")
	data, err := os.ReadFile(cgroupPath)
	if err != nil {
		return "", err
	}

	// 解析IO操作最多的设备
	deviceMap := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		var devMajor, devMinor, count int
		_, err := fmt.Sscanf(scanner.Text(), "%d:%d Read %d", &devMajor, &devMinor, &count)
		if err != nil {
			continue
		}
		device := fmt.Sprintf("%d:%d", devMajor, devMinor)
		deviceMap[device] += count
	}

	// 找出使用最多的设备
	maxCount := 0
	var activeDevice string
	for dev, count := range deviceMap {
		if count > maxCount {
			maxCount = count
			activeDevice = dev
		}
	}

	if activeDevice == "" {
		return "", fmt.Errorf("can not find io device")
	}
	return activeDevice, nil
}

func setContainerNetWbw(netWbw float64, containerPid string) error {
	//加载ifb模块
	if err := exec.Command("modprobe", "ifb").Run(); err != nil {
		log.Infof("ifb mod loaded failed: %v", err)
		return err
	}
	return nil
}
func setContainerNetRbw(netRbw float64, containerPid string) error {
	rateMbit := int(netRbw * 8)                 //大B转小b
	latency := 100 * time.Millisecond.Seconds() //0.1s
	burst := int(float64(rateMbit*1e6) * latency)
	tcArgs := []string{
		"-t", fmt.Sprintf("%d", containerPid),
		"-n",
		"tc", "qdisc", "replace", // 使用 replace 避免重复 add/del
		"dev", "eth0", "root",
		"tbf",
		"rate", fmt.Sprintf("%dmbit", rateMbit),
		"latency", fmt.Sprintf("%dms", latency),
		"burst", fmt.Sprintf("%d", burst),
	}
	cmd := exec.Command("nsenter", tcArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Infof("set container:%s netrbw err: %v", containerPid, err)
		return err
	}
	return nil
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
