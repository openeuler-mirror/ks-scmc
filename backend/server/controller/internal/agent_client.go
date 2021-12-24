package internal

import (
	"fmt"
	"ksc-mcube/common"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	agentConnLock sync.RWMutex
	agentConns    map[string]*grpc.ClientConn
)

func getAgentConn(host string) (*grpc.ClientConn, error) {
	agentConnLock.RLock()

	if conn, ok := agentConns[host]; ok && conn != nil {
		agentConnLock.RUnlock()
		return conn, nil
	}

	agentConnLock.RUnlock()

	addr := fmt.Sprintf("%s:%d", host, common.AgentPort)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	log.Infof("dial agent service: %v", addr)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Warnf("dial agent %s: %v", addr, err)
		return nil, err
	}

	agentConnLock.Lock()
	defer agentConnLock.Unlock()
	agentConns[host] = conn

	return conn, nil
}

func init() {
	agentConns = make(map[string]*grpc.ClientConn)
}
