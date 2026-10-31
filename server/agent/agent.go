package agent

import (
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"scmc/model/kysec"
	"scmc/model/litedb"
	"scmc/rpc/middleware/debug"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/image"
	"scmc/rpc/pb/network"
	"scmc/rpc/pb/node"
	"scmc/server/agent/internal"
)

func resumeNodeConfigs() {
	cfgs, err := litedb.NodeConfigList()
	if err != nil {
		log.Warnf("ContainerConfigList err=%v", err)
		return
	}

	for _, c := range cfgs {
		if c.SecurityConfig == nil {
			continue
		}
		if c.SecurityConfig.FileProtection != nil {
			fp := c.SecurityConfig.FileProtection
			if err := kysec.UpdateFileAccess(c.NodeID, fp.IsOn, fp.FileList, nil); err != nil {
				log.Warnf("UpdateFileAccess container=%v err=%v", c.NodeID, err)
			}
		}
		if c.SecurityConfig.ProcProtection != nil {
			pp := c.SecurityConfig.ProcProtection
			if err := kysec.UpdateWhiteList(c.NodeID, pp.IsOn, pp.ExeList, nil); err != nil {
				log.Warnf("UpdateWhiteList container=%v err=%v", c.NodeID, err)
			}
		}
	}
}

func resumeContainerConfigs() {
	cfgs, err := litedb.ContainerConfigList()
	if err != nil {
		log.Warnf("ContainerConfigList err=%v", err)
		return
	}

	for _, c := range cfgs {
		if c.SecurityConfig == nil {
			continue
		}
		if c.SecurityConfig.FileProtection != nil {
			fp := c.SecurityConfig.FileProtection
			if err := kysec.UpdateFileAccess(c.ContainerID, fp.IsOn, fp.FileList, nil); err != nil {
				log.Warnf("UpdateFileAccess container=%v err=%v", c.ContainerID, err)
			}
		}
		if c.SecurityConfig.ProcProtection != nil {
			pp := c.SecurityConfig.ProcProtection
			if err := kysec.UpdateWhiteList(c.ContainerID, pp.IsOn, pp.ExeList, nil); err != nil {
				log.Warnf("UpdateWhiteList container=%v err=%v", c.ContainerID, err)
			}
		}
	}
}

func updateVirtualNicMacAddr() {
	for i := 0; i < 5; i++ {
		internal.UpdataVirtualNicMAC()
		time.Sleep(time.Second * 30)
	}
}

func Server() (*grpc.Server, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(debug.UnaryInterceptor()),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // If a client pings more than once every 10 seconds, terminate the connection
			PermitWithoutStream: true,             // Allow pings even when there are no active streams
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     60 * time.Second, // If a client is idle for 60 seconds, send a GOAWAY
			MaxConnectionAge:      30 * time.Minute, // If any connection is alive for more than 30 minutes, send a GOAWAY
			MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
			Time:                  20 * time.Second, // Ping the client if it is idle for 20 seconds to ensure the connection is still active
			Timeout:               5 * time.Second,  // Wait 5 second for the ping ack before assuming the connection is dead
		}),
	}

	s := grpc.NewServer(opts...)

	container.RegisterContainerServer(s, &internal.ContainerServer{})
	image.RegisterImageServer(s, &internal.ImageServer{})
	network.RegisterNetworkServer(s, &internal.NetworkServer{})
	node.RegisterNodeServer(s, &internal.NodeServer{})

	resumeContainerConfigs()
	resumeNodeConfigs()

	go updateVirtualNicMacAddr()

	go internal.NodeWhitelistConfig()
	go internal.ContainerWhiteConfig()
	go internal.CPUUsageProbe()

	return s, nil
}
