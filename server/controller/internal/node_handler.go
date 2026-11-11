package internal

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model/db"
	"scmc/rpc"
	pb "scmc/rpc/pb/node"
	"scmc/server/args"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func (s *NodeServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	nodes, err := db.ListNodes()
	if err != nil {
		return nil, rpc.ErrDatabaseFail
	}

	var reply pb.ListReply
	for _, n := range nodes {
		s, _ := GetNodeStatus(n)
		reply.Nodes = append(reply.Nodes, &pb.NodeInfo{
			Id:         n.ID,
			Name:       n.Name,
			Address:    n.Address,
			Comment:    n.Comment,
			UnreadWarn: n.UnreadWarn,
			RscLimit: &pb.ResourceLimit{
				CpuLimit:    n.CpuLimit,
				MemoryLimit: n.MemoryLimit,
				DiskLimit:   n.DiskLimit,
			},
			Status: s,
		})
	}

	// 校验节点是否已被删除
	currentNodes, err := db.ListNodes()
	if err != nil {
		log.Warnf("ListNodes failed: %v", err)
		reply.Nodes = []*pb.NodeInfo{}
		return &reply, nil
	}

	validNodeMap := make(map[int64]bool)
	for _, n := range currentNodes {
		validNodeMap[n.ID] = true
	}

	var filteredNodes []*pb.NodeInfo
	for _, node := range reply.Nodes {
		if validNodeMap[node.Id] {
			filteredNodes = append(filteredNodes, node)
		}
	}
	reply.Nodes = filteredNodes
	return &reply, nil
}

func (s *NodeServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if !args.IsValidNodeAddr(in.Address) {
		return nil, status.Errorf(codes.InvalidArgument, "节点地址参数错误")
	} else if !args.NodeName.Check(in.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "节点名称参数错误")
	} else if !args.NodeComment.Check(in.Comment) {
		return nil, status.Errorf(codes.InvalidArgument, "节点备注参数错误")
	}

	conn, err := getAgentConn(in.Address)
	if err != nil {
		log.Warnf("get agent connection addr=%v err=%v", in.Address, err)
		return nil, rpc.ErrInternal
	}

	// agent 服务检查是否自身是否已被添加
	uuid := uuid.New().String()
	_, err = pb.NewNodeClient(conn).Create(ctx, &pb.CreateRequest{Address: in.Address, Uuid: uuid})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Unavailable {
				return nil, status.Errorf(codes.Internal, "节点连接失败")
			}
		}
		return nil, err
	}

	n, err := db.QueryNodeByAddr(in.Address)
	if err == db.ErrRecordNotFound {
		n := db.NodeInfo{
			Name:    in.Name,
			Address: in.Address,
			Comment: in.Comment,
			UUID:    uuid,
		}
		if e := db.CreateNode(&n); e != nil {
			if e == db.ErrDuplicateKey {
				return nil, rpc.ErrAlreadyExists
			}
			return nil, rpc.ErrDatabaseFail
		}
	} else if err != nil {
		log.Warnf("query node by addr err=%v", err)
		return nil, rpc.ErrDatabaseFail
	} else {
		if !n.Deleted {
			return nil, rpc.ErrAlreadyExists
		}

		// 记录存在, 更新字段, 将删除标记位设置为false
		n.Name = in.Name
		n.Comment = in.Comment
		n.Deleted = false
		n.UUID = uuid
		if e := db.UpdateNode(n); e != nil {
			log.Warnf("update node err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	return &pb.CreateReply{}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if len(in.Ids) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodes, err := db.QueryNodeByIDs(in.Ids)
	if err != nil {
		return nil, rpc.ErrDatabaseFail
	} else if len(nodes) == 0 {
		return nil, status.Error(codes.NotFound, "节点不存在")
	}

	for _, n := range nodes {
		conn, err := getAgentConn(n.Address)
		if err != nil {
			log.Warnf("get agent connection addr=%v err=%v", n.Address, err)
			return nil, rpc.ErrInternal
		}
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer timeoutCancel()
		_, err = pb.NewNodeClient(conn).Remove(timeoutCtx, &pb.RemoveRequest{Ids: in.Ids, Uuid: n.UUID})
		if err != nil { // 离线（换了IP）或者关机的时候会因为这里失败, 导致不能Remove
			/*
			   if st, ok := status.FromError(err); ok {
			       if st.Code() == codes.Unavailable {
			           return nil, status.Errorf(codes.Internal, "节点连接失败")
			       }
			   }
			   return nil, err
			*/
			// 打印日志，记录一下，不做具体逻辑处理，不返回失败
			log.Warnf("Remove: ip: %v, err=%v", n.Address, err)
		}

		db.DeleteWarnLogsByNodeIDs(n.ID)
		n.Deleted = true
		n.Name = uuid.New().String() // issue #67161
		if err := db.UpdateNode(n); err != nil {
			return nil, rpc.ErrDatabaseFail
		}
	}
	return &pb.RemoveReply{}, nil
}

// Since `List` returns the data of `Status` too, client never calls `Status`, therefore abandon this interface
func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	return nil, rpc.ErrUnimplemented
}

func (s *NodeServer) UpdateFileProtect(ctx context.Context, in *pb.UpdateFileProtectRequest) (*pb.UpdateFileProtectReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	_, err = cli.UpdateFileProtect(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return &pb.UpdateFileProtectReply{}, nil
}

func (s *NodeServer) UpdateNetworkRule(ctx context.Context, in *pb.UpdateNetworkRuleRequest) (*pb.UpdateNetworkRuleReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	_, err = cli.UpdateNetworkRule(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return &pb.UpdateNetworkRuleReply{}, nil
}

func (s *NodeServer) UpdateNetworkProcessWhiteList(ctx context.Context, in *pb.UpdateNetworkProcessWhiteListRequest) (*pb.UpdateNetworkProcessWhiteListReply, error) {
	log.Debugf("UpdateNetworkProcessWhiteList called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	_, err = cli.UpdateNetworkProcessWhiteList(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return &pb.UpdateNetworkProcessWhiteListReply{}, nil
}

func (s *NodeServer) GetNetworkProcessWhiteList(ctx context.Context, in *pb.GetNetworkProcessWhiteListRequest) (*pb.GetNetworkProcessWhiteListReply, error) {
	log.Debugf("GetNetworkProcessWhiteList called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNetworkProcessWhiteList(ctx, in)
	if err != nil {
		log.Warnf("GetNetworkProcessWhiteList node: %v", err)
		return nil, err
	}

	log.Debugf("GetNetworkProcessWhiteList called: node_id=%v  at server reply:%v", in.NodeId, reply)

	return reply, nil
}

func (s *NodeServer) GetNodeDefaultNetProWhiteList(ctx context.Context, in *pb.GetNodeDefaultNetProWhiteListRequest) (*pb.GetNodeDefaultNetProWhiteListReply, error) {
	log.Debugf("GetNodeDefaultNetProWhiteList called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNodeDefaultNetProWhiteList(ctx, in)
	if err != nil {
		log.Warnf("GetNodeDefaultNetProWhiteList node: %v", err)
		return nil, err
	}

	log.Debugf("GetNodeDefaultNetProWhiteList called: node_id=%v  at server reply:%v", in.NodeId, reply)

	return reply, nil
}

func (s *NodeServer) GetFileProtect(ctx context.Context, in *pb.GetFileProtectRequest) (*pb.GetFileProtectReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetFileProtect(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	log.Debugf("UpdateFileProtect called: node_id=%v  at server reply:%v", in.NodeId, reply)

	return reply, nil
}

func (s *NodeServer) GetNetworkRule(ctx context.Context, in *pb.GetNetworkRuleRequest) (*pb.GetNetworkRuleReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNetworkRule(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return reply, nil
}

func (s *NodeServer) GetNodeSecurityConfig(ctx context.Context, in *pb.GetNodeSecurityConfigRequest) (*pb.GetNodeSecurityConfigReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNodeSecurityConfig(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return reply, nil
}

func (s *NodeServer) GetNodeDefaultPort(ctx context.Context, in *pb.GetNodeDefaultPortRequest) (*pb.GetNodeDefaultPortReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNodeDefaultPort(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return reply, nil
}

func (s *NodeServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	if utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 50 {
		return nil, status.Errorf(codes.InvalidArgument, "节点名长度限制1-50")
	} else if utf8.RuneCountInString(in.Comment) > 200 {
		return nil, status.Errorf(codes.InvalidArgument, "节点备注长度限制0-200")
	}

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}

	if in.Comment != "" {
		n.Comment = in.Comment
	}
	if in.Name != "" {
		n.Name = in.Name
	}
	if in.RscLimit != nil {
		n.CpuLimit = in.RscLimit.CpuLimit
		n.MemoryLimit = in.RscLimit.MemoryLimit
		n.DiskLimit = in.RscLimit.DiskLimit

		conn, err := getAgentConn(n.Address)
		if err != nil {
			return nil, rpc.ErrInternal
		}

		cli := pb.NewNodeClient(conn)
		_, err = cli.Update(ctx, in)
		if err != nil {
			log.Warnf("Update node: %v", err)
			return nil, err
		}
	}

	if err := db.UpdateNode(n); err != nil {
		log.Infof("UpdateNode %+v err=%v", n, err)
		return nil, rpc.ErrDatabaseFail
	}

	return &pb.UpdateReply{}, nil
}

func GetNodeStatus(n *db.NodeInfo) (*pb.NodeStatus, error) {
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, err
	}

	cli := pb.NewNodeClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	r, err := cli.Status(ctx_, &pb.StatusRequest{})
	if err != nil {
		log.Warnf("get node status ID=%v address=%v: %v", n.ID, n.Address, err)
		s, ok := status.FromError(err)
		if ok && s.Code() == codes.Unavailable {
			return &pb.NodeStatus{
				State: int64(pb.NodeState_Offline),
			}, nil
		}
		return &pb.NodeStatus{
			State: int64(pb.NodeState_Unknown),
		}, nil
	}

	var s *pb.NodeStatus
	if len(r.StatusList) > 0 {
		s = r.StatusList[0]
	}
	return s, nil
}

func (s *NodeServer) UpdateProcessWhiteList(ctx context.Context, in *pb.UpdateProcessWhiteListRequest) (*pb.UpdateProcessWhiteListReply, error) {
	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	_, err = cli.UpdateProcessWhiteList(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return &pb.UpdateProcessWhiteListReply{}, nil
}

func (s *NodeServer) GetNodeProcessWhiteList(ctx context.Context, in *pb.GetProcessWhiteListRequest) (*pb.GetProcessWhiteListReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNodeProcessWhiteList(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return reply, nil
}

func (s *NodeServer) GetNodeDefaultProcessWhiteList(ctx context.Context, in *pb.GetNodeDefaultProcessWhiteListRequest) (*pb.GetNodeDefaultProcessWhiteListReply, error) {
	log.Debugf("UpdateFileProtect called: node_id=%v  at server ", in.NodeId)

	n, err := db.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrDatabaseFail
	}
	conn, err := getAgentConn(n.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	reply, err := cli.GetNodeDefaultProcessWhiteList(ctx, in)
	if err != nil {
		log.Warnf("Update node: %v", err)
		return nil, err
	}

	return reply, nil
}
