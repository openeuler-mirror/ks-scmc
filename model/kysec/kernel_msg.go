package kysec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"

	// "sync"
	// "strings"
	// "io/ioutil"

	"scmc/common"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	NLMSG_COM_TYPE_HELLO   = 1
	NLMSG_COM_TYPE_QUIT    = 2
	NLMSG_COM_TYPE_REQUEST = 3
	NLMSG_COM_TYPE_NOTIFY  = 4

	NLMSG_FUNC_TYPE_WL  = 1 // NLMSG_FUNC_TYPE_PROCESS
	NLMSG_FUNC_TYPE_AC  = 2 // NLMSG_FUNC_TYPE_FILE
	NLMSG_FUNC_TYPE_NET = 3

	NLMSG_ACTION_TYPE_ADD     = 1
	NLMSG_ACTION_TYPE_DEL     = 2
	NLMSG_ACTION_TYPE_FREE    = 3
	NLMSG_ACTION_TYPE_ENABLED = 4

	NETLINK_WL = 30
	/* v6 版本改为了 31 */
	NETLINK_WL_V6 = 31

	WL_HASH_SIZE = 16
	ID_MAX_SIZE  = 64
	MSG_SIZE     = 84

	TASK_COMM_LEN = 16
)

type AuditLog struct {
	TvSec  uint64
	Type   uint8
	Id     [ID_MAX_SIZE]byte
	Sub    [TASK_COMM_LEN]byte
	Op     uint8
	Result uint8
	ObjLen int32
	Path   string
}

var localEndian binary.ByteOrder = binary.LittleEndian

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0x1234)
	switch buf {
	case [2]byte{0x34, 0x12}:
		localEndian = binary.LittleEndian
	case [2]byte{0x12, 0x34}:
		localEndian = binary.BigEndian
	default:
		panic("Could not determine local endianness.")
	}
}

func nlmsgSpace(len int) uint32 {
	alignto := 4
	headlen := (int(unsafe.Sizeof(unix.NlMsghdr{})) + alignto - 1) / alignto * alignto
	length := len + headlen
	alignlen := ((length + alignto - 1) / alignto) * alignto

	//log.Debugf("source len:%v, headlen: %v, alignlen:%v", len, headlen, alignlen)
	return uint32(alignlen)
}

func setNlmsgType(x, y, z int) uint16 {
	return uint16(((x) << 12) | ((y) << 8) | ((z) << 4))
}

// 确认系统是否为 v6 系统
func bCheckSysIsV6() bool {
	/*
		content, err := ioutil.ReadFile("/etc/.kyinfo")
		if err != nil {
			return false
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			str := strings.TrimSpace(line)
			str = strings.ToLower(str)
			if strings.HasPrefix(str, "milestone") {
				kv := strings.Split(str, "=")
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.TrimSpace(kv[1])
					log.Warnf("system: %s, key: %s, value: %s", str, key, value)
					return false
				}
			}
		}

		return false
	*/
	// v6 以及以后的版本 不存在 /etc/.kyinfo 这个文件了
	_, err := os.Stat("/etc/.kyinfo")
	if err != nil {
		return os.IsNotExist(err)
	}
	return false

}

func initNetlink() (int, error) {
	return _initNetlink(0)
}

func initNetlinkRecv() (int, error) {
	return _initNetlink(uint32(os.Getpid()))
}

func _initNetlink(pid uint32) (int, error) {
	// 以配置文件里面的为准
	protocol := NETLINK_WL
	if common.Config.Netlink.Protocol != NETLINK_WL {
		protocol = int(common.Config.Netlink.Protocol)
	} else {
		/* 配置文件里面如果没有配置， 判断是不是 v6 系统*/
		if bCheckSysIsV6() {
			protocol = NETLINK_WL_V6
		} else {
			protocol = NETLINK_WL
		}
	}

	//fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, NETLINK_WL)
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, protocol)
	if err != nil {
		log.Warnf("Failed to create the  whitelist netlink socket:%v, protocol: %d", err, protocol)
		return 0, err
	}

	tv := &unix.Timeval{
		Sec: 1,
	}
	if err = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, tv); err != nil {
		log.Warnf("SetsockoptTimeval err: %v", err)
	}

	lsa := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: 0,
		Pid:    pid,
	}

	if err := unix.Bind(fd, lsa); err != nil {
		unix.Close(fd)
		return 0, err
	}

	msgType := setNlmsgType(NLMSG_COM_TYPE_HELLO, 0, 0)
	if err = sendNetlink(fd, []byte{}, 0, msgType); err != nil {
		return 0, err
	}

	return fd, nil
}

func deinitNetlink(fd int) {
	msgtype := setNlmsgType(NLMSG_COM_TYPE_QUIT, 0, 0)
	sendNetlink(fd, []byte{}, 0, msgtype)
	unix.Close(fd)
}

func sendNetlink(fd int, msg []byte, len int, msgType uint16) error {
	nlhLen := nlmsgSpace(len)

	nlh := &unix.NlMsghdr{
		Len:  nlhLen,
		Type: msgType,
		Pid:  uint32(os.Getpid()),
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, localEndian, nlh); err != nil {
		log.Warnf("NlMsghdr to []byte err: %v", err)
		return err
	}

	headLen := uint32(buf.Len())
	supLen := nlhLen - headLen - uint32(len)
	if len > 0 {
		buffer := make([]byte, supLen)
		if binary.Write(buf, localEndian, msg) != nil ||
			binary.Write(buf, localEndian, buffer) != nil {
			log.Warnf("NlMsghdr to []byte err")
			return os.ErrInvalid
		}
	}

	// log.Debugf("nlh_len:%v, len:%v, headLen:%v, suplen:%v, buflen:%v", nlh_len, len, headLen, supLen, buf.Len())
	if err := unix.Sendto(fd, buf.Bytes(), 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		log.Warnf("Failed to send  whitelist netlink: %v", err)
		return err
	}

	return nil
}

// 安全相关审计日志
func recvNetlink() ([]byte, error) {
	fd, err := initNetlinkRecv()
	if err != nil {
		// log.Warnf("recvNetlink initNetlink failed: %v", err)
		return nil, err
	}

	recvLen := 0
	buf := make([]byte, 1024)
	for tryCnt := 0; tryCnt < 3; tryCnt++ {
		n, _, err := unix.Recvfrom(fd, buf, unix.MSG_PEEK)
		if err == nil {
			recvLen = n
			break
		}

		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		deinitNetlink(fd)
		return nil, err
	}

	if recvLen <= 0 {
		deinitNetlink(fd)
		return nil, fmt.Errorf("recvNet try max times")
	}

	if recvLen > len(buf) {
		buf = make([]byte, recvLen)
	}

	recvLen, _, err = unix.Recvfrom(fd, buf, 0)
	deinitNetlink(fd)
	if err != nil {
		return nil, err
	}

	nlhdrLen := int(unsafe.Sizeof(unix.NlMsghdr{}))
	if recvLen <= nlhdrLen {
		return nil, fmt.Errorf("recvNetlink msg too short: %v", recvLen)
	}

	res := make([]byte, recvLen-nlhdrLen)
	copy(res, buf[nlhdrLen:recvLen])
	return res, nil
}

func GetAuditLog() {
	for {
		buf, err := recvNetlink()
		if err != nil {
			//log.Warnf("GetAuditLog recvNetlink failed: %v", err)
			continue
		} else {
			logHdrlen := int32(unsafe.Sizeof(AuditLog{})) - int32(unsafe.Sizeof(""))
			if int32(len(buf)) < int32(logHdrlen) {
				continue
			}
			adtLog := *(*AuditLog)(unsafe.Pointer(&buf[0]))
			if int32(len(buf)) < int32(logHdrlen+adtLog.ObjLen) {
				continue
			}
			adtLog.Path = string(buf[logHdrlen : logHdrlen+adtLog.ObjLen])
			//log.Warnf("adtLog: %++v", adtLog)
			auditLogChan <- &adtLog
		}
	}
}

/*
 * 容器进程白名单数据
 */
func updateWhiteListToKern(fd int, id, hash, name []byte, add bool) error {
	if id == nil || hash == nil {
		log.Warnf("param is nil")
		return os.ErrInvalid
	}

	if len(hash) > WL_HASH_SIZE || len(id) > ID_MAX_SIZE {
		log.Warnf("param len err: %v, %v", len(hash), len(id))
		return os.ErrInvalid
	}

	var realID [ID_MAX_SIZE]byte
	copy(realID[:], id)
	var realHash [WL_HASH_SIZE]byte
	copy(realHash[:], hash)
	var namelen int32
	if name != nil {
		namelen = int32(len(name))
	}

	pad := make([]byte, 4)

	buf := new(bytes.Buffer)
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, realHash) != nil ||
		binary.Write(buf, localEndian, namelen) != nil ||
		binary.Write(buf, localEndian, pad) != nil ||
		binary.Write(buf, localEndian, name) != nil {
		log.Warnf("updateWhiteListToKern to []byte err")
		return os.ErrInvalid
	}

	/* 这里是为了保证 8 字节对齐 */
	currentLen := WL_HASH_SIZE + 4 + 4 + len(name)
	padAfterName := (8 - (currentLen % 8)) % 8
	if padAfterName > 0 {
		padName := make([]byte, padAfterName)
		if binary.Write(buf, localEndian, padName) != nil {
			log.Warnf("updateWhiteListToKern to []byte err")
			return os.ErrInvalid
		}
	}

	var msgtype uint16
	if add {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_WL, NLMSG_ACTION_TYPE_ADD)
	} else {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_WL, NLMSG_ACTION_TYPE_DEL)
	}

	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

func freeWhiteListToKern(fd int, id []byte, who int) error {
	if id == nil || len(id) > ID_MAX_SIZE {
		return os.ErrInvalid
	}

	var realID [ID_MAX_SIZE]byte
	copy(realID[:], id)
	buf := new(bytes.Buffer)
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, [WL_HASH_SIZE]byte{}) != nil ||
		binary.Write(buf, localEndian, int32(0)) != nil ||
		binary.Write(buf, localEndian, []byte{}) != nil {
		log.Warnf("freeWhiteListToKern to []byte err")
		return os.ErrInvalid
	}

	msgtype := setNlmsgType(NLMSG_COM_TYPE_REQUEST, who, NLMSG_ACTION_TYPE_FREE)

	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

/*
 * 容器文件保护消息
 */
func setFileAccessToKern(fd int, id, filePath []byte, add bool) error {
	if id == nil || filePath == nil || len(id) > ID_MAX_SIZE {
		return os.ErrInvalid
	}

	buf := new(bytes.Buffer)
	var realID [ID_MAX_SIZE]byte
	copy(realID[:], id)
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, int32(len(filePath))) != nil ||
		binary.Write(buf, localEndian, int32(0)) != nil ||
		binary.Write(buf, localEndian, filePath) != nil {
		log.Warnf("setFileAccessToKern to []byte err")
		return os.ErrInvalid
	}

	/* 这里是为了保证 8 字节对齐 */
	currentLen := 4 + 4 + len(filePath)
	padAfterName := (8 - (currentLen % 8)) % 8
	if padAfterName > 0 {
		padName := make([]byte, padAfterName)
		if binary.Write(buf, localEndian, padName) != nil {
			log.Warnf("setFileAccessToKern to []byte err")
			return os.ErrInvalid
		}
	}

	var msgtype uint16
	if add {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_AC, NLMSG_ACTION_TYPE_ADD)
	} else {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_AC, NLMSG_ACTION_TYPE_DEL)
	}

	// return sendNetlink(fd, buf.Bytes(), buflen, msgtype)
	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

/*
 * 容器网络进程白名单
 */
func setNProcAccess(fd int, id, filePath []byte, add bool) error {
	if id == nil || filePath == nil || len(id) > ID_MAX_SIZE {
		return os.ErrInvalid
	}

	var realID [ID_MAX_SIZE]byte
	copy(realID[:], id) // 容器ID
	pad := make([]byte, 4)
	buf := new(bytes.Buffer)
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, int32(len(filePath))) != nil ||
		binary.Write(buf, localEndian, pad) != nil ||
		binary.Write(buf, localEndian, filePath) != nil {
		log.Warnf("setNProcAccess to []byte err")
		return os.ErrInvalid
	}

	/* 这里是为了保证 8 字节对齐 */
	currentLen := 4 + 4 + len(filePath)
	padAfterName := (8 - (currentLen % 8)) % 8
	if padAfterName > 0 {
		padName := make([]byte, padAfterName)
		if binary.Write(buf, localEndian, padName) != nil {
			log.Warnf("updateWhiteListToKern to []byte err")
			return os.ErrInvalid
		}
	}

	var msgtype uint16
	if add {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_NET, NLMSG_ACTION_TYPE_ADD)
	} else {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_NET, NLMSG_ACTION_TYPE_DEL)
	}

	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

/*
 *使能处理，也就是开关
 */
func setStatusToKern(fd int, id []byte, who int, enabled bool) error {
	if id == nil || len(id) > ID_MAX_SIZE {
		return os.ErrInvalid
	}

	buf := new(bytes.Buffer)

	var realID [ID_MAX_SIZE]byte
	copy(realID[:], id)
	var isOn int32
	if enabled {
		isOn = 1
	}
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, isOn) != nil {
		log.Warnf("to []byte err")
		return os.ErrInvalid
	}

	// log.Debugf("set %v(1:wl, 2:ac 3:net) enable: %v", who, isOn)
	msgtype := setNlmsgType(NLMSG_COM_TYPE_REQUEST, who, NLMSG_ACTION_TYPE_ENABLED)

	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

/*
 * 网络进程白名单设置
 */
func UpdateNprocProtection(containerId string, enable bool, addfilePath []string, delfilePath []string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	for _, addfile := range addfilePath {
		filePath := strings.TrimSpace(addfile)
		if err := setNProcAccess(fd, []byte(containerId), []byte(filePath), true); err != nil {
			log.Warnf("add %v NProc file(%v) access err: %v", containerId, filePath, err)
		}
	}

	if err := setStatusToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_NET, enable); err != nil {
		log.Warnf("set %v white list status([%v]) err: %v", containerId, enable, err)
	}

	for _, delfile := range delfilePath {
		filePath := strings.TrimSpace(delfile)
		if err := setNProcAccess(fd, []byte(containerId), []byte(filePath), false); err != nil {
			log.Warnf("del %v NProc file(%v) access err: %v", containerId, filePath, err)
		}
	}

	deinitNetlink(fd)

	return nil
}

/*
 * 白名单更新(增删) add和del为程序的hash
 * 进程安全
 */
func UpdateWhiteList(containerId string, enable bool, add []string, del []string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	for _, v := range add {
		decoded, err := hex.DecodeString(v)
		if err != nil {
			log.Warnf("decode %v err: %v", v, err)
			continue
		}
		if err := updateWhiteListToKern(fd, []byte(containerId), decoded, []byte{}, true); err != nil {
			log.Warnf("add %v white list ([%v]) err: %v", containerId, v, err)
		}
	}

	if err := setStatusToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_WL, enable); err != nil {
		log.Warnf("set %v white list status([%v]) err: %v", containerId, enable, err)
	}

	for _, v := range del {
		decoded, err := hex.DecodeString(v)
		if err != nil {
			log.Warnf("decode %v err: %v", v, err)
			continue
		}
		if err := updateWhiteListToKern(fd, []byte(containerId), decoded, []byte{}, false); err != nil {
			log.Warnf("del %v white list ([%v]) err: %v", containerId, v, err)
		}
	}

	deinitNetlink(fd)

	return nil
}

/*
 * 白名单清除
 * 进程安全清除
 */
func CleanWhiteList(containerId string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	freeWhiteListToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_WL)
	deinitNetlink(fd)
	return nil
}

/*
 * 文件访问控制
 * 文件保护
 */
func UpdateFileAccess(containerId string, enable bool, addfilePath []string, delfilePath []string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	for _, filePath := range addfilePath {
		if err := setFileAccessToKern(fd, []byte(containerId), []byte(filePath), true); err != nil {
			log.Warnf("add %v file(%v) access err: %v", containerId, filePath, err)
		}
	}

	if err := setStatusToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_AC, enable); err != nil {
		log.Warnf("set %v file access status(%v) err: %v", containerId, enable, err)
	}

	for _, filePath := range delfilePath {
		if err := setFileAccessToKern(fd, []byte(containerId), []byte(filePath), false); err != nil {
			log.Warnf("del %v file(%v) access err: %v", containerId, filePath, err)
		}
	}

	deinitNetlink(fd)
	return nil
}

/*
 * 白名单清除
 */
func CleanFileAccess(containerId string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	freeWhiteListToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_AC)
	deinitNetlink(fd)
	return nil
}

/*
 * 网络进程白名单清除
 */
func CleanNProcRule(containerId string) error {
	fd, err := initNetlink()
	if err != nil {
		log.Warnf("initNetlink() err: %v", err)
		return err
	}

	freeWhiteListToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_NET)
	deinitNetlink(fd)
	return nil
}

/*
 * 清除设置
 */
func CleanAll(containerID string, force bool) error {
	err := CleanFileAccess(containerID)
	if err != nil {
		log.Warnf("%v CleanFileAccess: %v", containerID, err)
		if !force {
			return err
		}
	}

	err = CleanWhiteList(containerID)
	if err != nil {
		log.Warnf("%v CleanWhiteList: %v", containerID, err)
	}

	err = CleanNProcRule(containerID)
	if err != nil {
		log.Warnf("%v CleanNProcRule: %v", containerID, err)
	}

	return err
}
