package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"scmc/model/db"
	"strings"
	"path/filepath"

	pb "scmc/rpc/pb/container"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	// "scmc/server/agent/internal"
	"strconv"
	// "strings"
	"sync"
	"time"
	"scmc/model/litedb"
)

var dockerRootDir string

const (
	NoneBit  = 0
	EnvBit   = 1
	MountBit = 2
	IpcBit = 4
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

/*
 * IPC 共享， 修改 IPC 配置信息
 * /config.v2.json 里面的 ShmPath 路径
 * /hostconfig.json 里面的 IpcMode 字段
 * "IpcMode":"container:f9873b27177a53b2b6a6da561fe9d56cc7508528a25ac0109c29f000daa11ae2"
 * "ShmPath":"/var/lib/docker/containers/f9873b27177a53b2b6a6da561fe9d56cc7508528a25ac0109c29f000daa11ae2/mounts/shm"
 * 其中 f9873b27177a53b2b6a6da561fe9d56cc7508528a25ac0109c29f000daa11ae2 为基准容器的容器ID
 */
func modifyIPC(dataBuf map[string]interface{}, containerId string, baseIPCID string) {
	baseIPCID = strings.TrimSpace(baseIPCID)
	dir, err := getDockerRootDir()
	if err != nil {
		log.Warnf("getDockerRootDir error: %v", err)
		return
	}

	// 修改 config.v2.json 里面的 ShmPath
	shmPathValue := filepath.Join(dir, "containers", baseIPCID, "mounts", "shm")
	if len(baseIPCID) == 0 {
		shmPathValue = ""
	}
	_, ok := dataBuf["ShmPath"].(string)
	if ok {
		dataBuf["ShmPath"] = shmPathValue
	}

	// 修改 IpcMode 
	hostconfigPath := filepath.Join(dir, "containers", containerId, "hostconfig.json")
	if _, err := os.Stat(hostconfigPath); os.IsNotExist(err) {
		log.Warnf("find file : %s error!, err: %v", hostconfigPath, err)
		return
	}

	hostconfigDataBuf, err := parseConfigFile(hostconfigPath)
	if err != nil {
		log.Warnf("parseConfigFile: %s, err: %v", hostconfigPath, err)
		return
	}

	ipcModeValue := "container:" + baseIPCID
	if len(baseIPCID) == 0 {
		ipcModeValue = "private"
	}
	_, hok := hostconfigDataBuf["IpcMode"].(string)
	if hok {
		hostconfigDataBuf["IpcMode"] = ipcModeValue
	}
	if err = writeConfigFile(hostconfigPath, hostconfigDataBuf); err != nil {
		log.Warnf("update %s, err: %v", hostconfigPath, err)
		return
	}

	return
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

	// modify ipc
	if which&IpcBit != 0 {
		ipcmapCfg , err := litedb.GetIpcmapConfigByCID(containerId)
		if err != nil {
			log.Warnf("config GetIpcmapConfigByCID id: %v, err:%v", containerId, err)
		} else {
			modifyIPC(dataBuf, containerId, ipcmapCfg.IpcID)
		}
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
	data, err := ioutil.ReadFile(cgroupPath)
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

type DockerManager struct {
	client       *client.Client
	tcController *TrafficController
	cancel       context.CancelFunc
	startOnce    sync.Once  // 保证启动逻辑只执行一次
	mu           sync.Mutex // 状态锁
	isRunning    bool       // 运行状态标识
}

var (
	instance *DockerManager
	once     sync.Once
)

func GetDockerManager(cli *client.Client) *DockerManager {
	once.Do(func() {
		tc := NewTrafficController()
		instance = &DockerManager{
			client:       cli,
			tcController: tc,
		}
	})
	return instance
}
func (dm *DockerManager) Start() {
	dm.startOnce.Do(func() {
		dm.mu.Lock()
		defer dm.mu.Unlock()

		if !dm.isRunning {
			go dm.startEventLoop() // 真正的启动逻辑
			dm.isRunning = true
		}
	})
}
func (dm *DockerManager) startEventLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	dm.cancel = cancel
	options := types.EventsOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", events.ContainerEventType),
		),
	}
	for {
		msgChan, errs := dm.client.Events(ctx, options)
		select {
		case msg := <-msgChan:
			dm.handleEvent(msg)
		case err := <-errs:
			log.Printf("Docker event error: %v", err)
		case <-ctx.Done():
			return
		}
	}
}
func (dm *DockerManager) Stop() {
	if dm.cancel != nil {
		dm.cancel()
	}
}

func (dm *DockerManager) handleEvent(msg events.Message) {
	if msg.Type != events.ContainerEventType {
		return
	}

	switch msg.Action {
	case "start", "restart":
		// 获取限速配置
		limit, err := db.QueryContainerLimitsByContainerId(msg.Actor.ID)
		if err != nil {
			log.Errorf("Events: start or restart.Can not get container limit config,err: %v", err)
			return
		}

		// 应用限速规则
		if pid, err := getContainerPID(msg.Actor.ID); err == nil {
			log.Errorf("Events: start or restart.Can not get container pid,err: %v", err)
			if pid == 0 {
				log.Errorf("container pid err.")
				return
			}
			dm.tcController.ApplyLimit(pid, limit.UploadSpeed, limit.DownSpeed)
		}

	case "die", "destroy":
		// 清理规则
		if pid, err := getContainerPID(msg.Actor.ID); err == nil {
			log.Errorf("Events: die or destroy.Can not get container limit config,err: %v", err)
			if pid == 0 {
				log.Errorf("Events: die or destroy.container pid err.")
				return
			}
			dm.tcController.RemoveLimit(pid)
		}
		//dm.dbManager.DeleteLimit(event.ID)
	}
}

type TrafficController struct{}

func NewTrafficController() *TrafficController {
	return &TrafficController{}
}
func (tc *TrafficController) ApplyLimit(pid int, upload, download float64) error {
	if upload == 0 && download == 0 {
		return tc.RemoveLimit(pid)
	}

	if upload > 0 {
		if err := setupIngress(pid, upload); err != nil {
			return err
		}
	}

	if download > 0 {
		if err := setupEgress(pid, download); err != nil {
			return err
		}
	}
	return nil
}
func (tc *TrafficController) RemoveLimit(pid int) error {
	cmds := []string{
		fmt.Sprintf("nsenter -t %d -n tc qdisc del dev eth0 root", pid),
		fmt.Sprintf("nsenter -t %d -n tc qdisc del dev eth0 ingress", pid),
		fmt.Sprintf("nsenter -t %d -n tc qdisc del dev ifb0 root", pid),
		fmt.Sprintf("nsenter -t %d -n ip link del ifb0", pid),
	}

	for _, cmd := range cmds {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			// 忽略"Device or resource busy"等错误
			log.Errorf("cmd: %v. exec error: %v", cmd, err)
		}
	}
	return nil
}
func setupIngress(pid int, rate float64) error {
	if err := createIFB(pid); err != nil {
		return err
	}

	cmds := []string{
		fmt.Sprintf("nsenter -t %d -n tc qdisc add dev eth0 handle ffff: ingress", pid),
		fmt.Sprintf("nsenter -t %d -n tc filter add dev eth0 parent ffff: protocol ip u32 match u32 0 0 action mirred egress redirect dev ifb0", pid),
		fmt.Sprintf("nsenter -t %d -n tc qdisc add dev ifb0 root tbf rate %dmbit latency 100ms burst %d",
			pid, rate*8, calculateBurst(rate)),
	}

	for _, cmd := range cmds {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			log.Errorf("ingress setup failed: %v", err)
			return fmt.Errorf("ingress setup failed: %v", err)
		}
	}
	return nil
}
func setupEgress(pid int, rate float64) error {
	cmd := fmt.Sprintf("nsenter -t %d -n tc qdisc add dev eth0 root tbf rate %dmbit latency 100ms burst %d",
		pid, rate*8, calculateBurst(rate))
	return exec.Command("sh", "-c", cmd).Run()
}
func createIFB(pid int) error {
	cmds := []string{
		"modprobe ifb",
		fmt.Sprintf("ip link add ifb0 type ifb"),
		fmt.Sprintf("ip link set ifb0 netns %d", pid),
		fmt.Sprintf("nsenter -t %d -n ip link set ifb0 up", pid),
	}

	for _, cmd := range cmds {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			return fmt.Errorf("IFB creation failed: %v", err)
		}
	}
	return nil
}

func calculateBurst(rate float64) float64 {
	if rate < 1 {
		return 1000
	}
	return rate * 125000 // 1MB/s = 1,000,000 bytes
}
func getContainerPID(containerID string) (int, error) {
	// 实现获取容器PID的逻辑（通过docker inspect）
	// 返回PID和可能的错误
	s := &internal.ContainerServer{}
	pid, err := s.GetContainerPid(containerID)
	if err != nil {
		return 0, err
	}
	return pid, nil
}
