package internal

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"scmc/common"
)

var (
	agentConnLock sync.RWMutex
	agentConns    map[string]*grpc.ClientConn
)

func GetAgentConn(host string) (*grpc.ClientConn, error) {
	return getAgentConn(host)
}

func getAgentConn(host string) (*grpc.ClientConn, error) {
	agentConnLock.RLock()

	if conn, ok := agentConns[host]; ok && conn != nil {
		agentConnLock.RUnlock()
		return conn, nil
	}

	agentConnLock.RUnlock()

	addr := fmt.Sprintf("%s:%d", host, common.Config.Agent.Port)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Errorf("GetAgentConn dial addr=%v, err=%v", addr, err)
		return nil, err
	}

	log.Infof("GetAgentConn dial addr=%v OK", addr)

	agentConnLock.Lock()
	defer agentConnLock.Unlock()
	agentConns[host] = conn

	return conn, nil
}

func init() {
	agentConns = make(map[string]*grpc.ClientConn)
}
