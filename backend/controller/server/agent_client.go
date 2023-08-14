package server

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"google.golang.org/grpc"
)

var (
	agentConnLock sync.RWMutex
	agentConns    map[string]*grpc.ClientConn

	agentServicePort uint
)

func getAgentConn(host string) (*grpc.ClientConn, error) {
	agentConnLock.RLock()

	if conn, ok := agentConns[host]; ok && conn != nil {
		agentConnLock.RUnlock()
		return conn, nil
	}

	agentConnLock.RUnlock()

	addr := fmt.Sprintf("%s:%d", host, agentServicePort)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	log.Printf("dial agent service: %v", addr)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Printf("Dial agent %s: %v", addr, err)
		return nil, err
	}

	agentConnLock.Lock()
	defer agentConnLock.Unlock()
	agentConns[host] = conn

	return conn, nil
}

func init() {
	flag.UintVar(&agentServicePort, "agent-service-port", 10051, "Agent service listening port")
	agentConns = make(map[string]*grpc.ClientConn)
}
