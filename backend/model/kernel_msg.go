package model

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"unsafe"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	NLMSG_COM_TYPE_HELLO   = 1
	NLMSG_COM_TYPE_QUIT    = 2
	NLMSG_COM_TYPE_REQUEST = 3
	NLMSG_COM_TYPE_NOTIFY  = 4

	NLMSG_FUNC_TYPE_WL = 1
	NLMSG_FUNC_TYPE_AC = 2

	NLMSG_ACTION_TYPE_ADD     = 1
	NLMSG_ACTION_TYPE_DEL     = 2
	NLMSG_ACTION_TYPE_FREE    = 3
	NLMSG_ACTION_TYPE_ENABLED = 4

	NETLINK_WL   = 30
	WL_HASH_SIZE = 16
	ID_MAX_SIZE  = 64
	MSG_SIZE     = 84
)

/*
type whiteList struct {
	id      [ID_MAX_SIZE]byte  //白名单列表id --- 容器的id
	hash    [WL_HASH_SIZE]byte //程序 hash
	pathlen int                //全路径长度
	path    []byte             //程序全路径
}

type fileAccessControl struct {
	id             [ID_MAX_SIZE]byte //文件访问控制列表id --- 容器的id
	filePathLen    int               //客体长度
	programPathLen int               //主体长度，保留
	filePath       []byte            //客体全路径
}

type controlSwitch struct {
	id      [ID_MAX_SIZE]byte //列表id --- 容器的id
	enabled bool              //开关
}
*/
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

func initNetlink() (int, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, NETLINK_WL)
	if err != nil {
		log.Warnf("Failed to create the  whitelist netlink socket:%v", err)
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
		Pid:    0,
	}

	if err := unix.Bind(fd, lsa); err != nil {
		unix.Close(fd)
		return 0, err
	}

	msg_type := setNlmsgType(NLMSG_COM_TYPE_HELLO, 0, 0)
	if err = sendNetlink(fd, []byte{}, 0, msg_type); err != nil {
		return 0, err
	}

	return fd, nil
}

func deinitNetlink(fd int) {
	msgtype := setNlmsgType(NLMSG_COM_TYPE_QUIT, 0, 0)
	sendNetlink(fd, []byte{}, 0, msgtype)
	unix.Close(fd)
}

func sendNetlink(fd int, msg []byte, len int, msg_type uint16) error {
	nlh_len := nlmsgSpace(len)

	nlh := &unix.NlMsghdr{
		Len:  nlh_len,
		Type: msg_type,
		Pid:  uint32(os.Getpid()),
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, localEndian, nlh); err != nil {
		log.Warnf("NlMsghdr to []byte err: %v", err)
		return err
	}

	headLen := uint32(buf.Len())
	supLen := nlh_len - headLen - uint32(len)
	if len > 0 {
		buffer := make([]byte, supLen)
		if binary.Write(buf, localEndian, msg) != nil ||
			binary.Write(buf, localEndian, buffer) != nil {
			log.Warnf("NlMsghdr to []byte err")
			return os.ErrInvalid
		}
	}

	log.Debugf("nlh_len:%v, len:%v, headLen:%v, suplen:%v, buflen:%v", nlh_len, len, headLen, supLen, buf.Len())
	if err := unix.Sendto(fd, buf.Bytes(), 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		log.Warnf("Failed to send  whitelist netlink: %v", err)
		return err
	}

	return nil
}

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

	buf := new(bytes.Buffer)
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, realHash) != nil ||
		binary.Write(buf, localEndian, namelen) != nil ||
		binary.Write(buf, localEndian, name) != nil {
		log.Warnf("to []byte err")
		return os.ErrInvalid
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
		log.Warnf("to []byte err")
		return os.ErrInvalid
	}

	msgtype := setNlmsgType(NLMSG_COM_TYPE_REQUEST, who, NLMSG_ACTION_TYPE_FREE)

	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

func setFileAccessToKern(fd int, id, filePath []byte, add bool) error {
	if id == nil || filePath == nil || len(id) > ID_MAX_SIZE {
		return os.ErrInvalid
	}

	buflen := MSG_SIZE + len(filePath) + 1
	buffer := make([]byte, 13) //MSG_SIZE - ID_MAX_SIZE - 4 - 4  + 1
	buf := new(bytes.Buffer)

	var realID [ID_MAX_SIZE]byte
	copy(realID[:], id)
	if binary.Write(buf, localEndian, realID) != nil ||
		binary.Write(buf, localEndian, int32(len(filePath))) != nil ||
		binary.Write(buf, localEndian, int32(0)) != nil ||
		binary.Write(buf, localEndian, filePath) != nil ||
		binary.Write(buf, localEndian, buffer) != nil {
		log.Warnf("to []byte err")
		return os.ErrInvalid
	}

	var msgtype uint16
	if add {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_AC, NLMSG_ACTION_TYPE_ADD)
	} else {
		msgtype = setNlmsgType(NLMSG_COM_TYPE_REQUEST, NLMSG_FUNC_TYPE_AC, NLMSG_ACTION_TYPE_DEL)
	}

	return sendNetlink(fd, buf.Bytes(), buflen, msgtype)
}

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

	log.Debugf("set %v(1:wl, 2:ac) enable: %v", who, isOn)
	msgtype := setNlmsgType(NLMSG_COM_TYPE_REQUEST, who, NLMSG_ACTION_TYPE_ENABLED)

	return sendNetlink(fd, buf.Bytes(), buf.Len(), msgtype)
}

/* 白名单更新(增删) add和del为程序的hash */
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

/* 白名单清除 */
func CleanWhiteList(containerId string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	freeWhiteListToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_WL)
	deinitNetlink(fd)
	return nil
}

/* 文件访问控制 */
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

/* 白名单清除 */
func CleanFileAccess(containerId string) error {
	fd, err := initNetlink()
	if err != nil {
		return err
	}

	freeWhiteListToKern(fd, []byte(containerId), NLMSG_FUNC_TYPE_AC)
	deinitNetlink(fd)
	return nil
}
