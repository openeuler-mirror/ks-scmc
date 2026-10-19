package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"scmc/common"
	"scmc/rpc/pb/container"
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
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func ListBackupImage(conn *grpc.ClientConn) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := container.NewContainerClient(conn)
	reply, err := cli.ListBackupImage(ctx, &container.ListBackupImageRequest{})
	if err != nil {
		return nil, err
	}

	r := make(map[string]bool, len(reply.Images))
	for _, i := range reply.Images {
		r[i] = true
	}

	return r, nil
}

func init() {
	agentConns = make(map[string]*grpc.ClientConn)
}
