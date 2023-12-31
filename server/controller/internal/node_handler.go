package internal

import (
	"context"
	"time"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/node"
)

type NodeServer struct {
	pb.UnimplementedNodeServer
}

func (s *NodeServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	nodes, err := model.ListNodes()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	for _, node := range nodes {
		s, _ := getNodeStatus(&node)
		reply.Nodes = append(reply.Nodes, &pb.NodeInfo{
			Id:         node.ID,
			Name:       node.Name,
			Address:    node.Address,
			Comment:    node.Comment,
			UnreadWarn: node.UnreadWarn,
			RscLimit: &pb.ResourceLimit{
				CpuLimit:    node.CpuLimit,
				MemoryLimit: node.MemoryLimit,
				DiskLimit:   node.DiskLimit,
			},
			Status: s,
		})
	}
	return &reply, nil
}

func (s *NodeServer) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateReply, error) {
	if !isValidNodeAddr(in.Address) {
		return nil, status.Errorf(codes.InvalidArgument, "节点地址参数错误")
	} else if !isValidNodeName(in.Name) {
		return nil, status.Errorf(codes.InvalidArgument, "节点名称参数错误")
	} else if !isValidNodeComment(in.Comment) {
		return nil, status.Errorf(codes.InvalidArgument, "节点备注参数错误")
	}

	conn, err := getAgentConn(in.Address)
	if err != nil {
		log.Warnf("get agent connection addr=%v err=%v", in.Address, err)
		return nil, rpc.ErrInternal
	}

	cli := pb.NewNodeClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err = cli.Status(ctx_, &pb.StatusRequest{})
	if err != nil {
		log.Warnf("check agent service err=%v", err)
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Unavailable {
				return nil, status.Errorf(codes.Internal, "节点连接失败")
			}
		}
		return nil, rpc.ErrInternal
	}

	// connect agent address, check is alive

	nodeInfo, err := model.QueryNodeByAddr(in.Address)
	if err == model.ErrRecordNotFound {
		if e := model.CreateNode(in.Name, in.Address, in.Comment); e != nil {
			if e == model.ErrDuplicateKey {
				return nil, rpc.ErrAlreadyExists
			}
			return nil, rpc.ErrInternal
		}
	} else if err != nil {
		log.Warnf("query node by addr err=%v", err)
		return nil, rpc.ErrInternal
	} else {
		if !nodeInfo.Deleted {
			return nil, rpc.ErrAlreadyExists
		}

		// 记录存在, 更新字段, 将删除标记位设置为false
		nodeInfo.Name = in.Name
		nodeInfo.Comment = in.Comment
		nodeInfo.Deleted = false
		if e := model.UpdateNode(nodeInfo); e != nil {
			log.Warnf("update node err=%v", err)
			return nil, rpc.ErrInternal
		}
	}

	return &pb.CreateReply{}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if err := model.RemoveNode(in.Ids); err != nil {
		return nil, rpc.ErrInternal
	}
	return &pb.RemoveReply{}, nil
}

func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	reply := pb.StatusReply{}

	nodes, err := model.ListNodes()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	var nodeToQuery []*model.NodeInfo
	for _, nodeID := range in.NodeIds {
		for i, node := range nodes {
			if node.ID == nodeID {
				nodeToQuery = append(nodeToQuery, &nodes[i])
				goto next
			}
		}
		log.Warnf("node ID=%v not found", nodeID)
		return nil, rpc.ErrNotFound
	next:
	}

	for _, node := range nodeToQuery {
		conn, err := getAgentConn(node.Address)
		if err != nil {
			log.Warnf("Failed to connect to agent service")
			reply.StatusList = append(reply.StatusList, &pb.NodeStatus{
				NodeId: node.ID,
				State:  int64(pb.NodeState_Offline),
			})
			continue
		}

		cli := pb.NewNodeClient(conn)
		ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		subReply, err := cli.Status(ctx_, in)
		if err != nil {
			log.Warnf("get node status ID=%v address=%v: %v", node.ID, node.Address, err)
			return nil, rpc.ErrInternal
		}

		log.Debugf("subReply: %+v", subReply)
		for i := range subReply.StatusList {
			subReply.StatusList[i].NodeId = node.ID
			reply.StatusList = append(reply.StatusList, subReply.StatusList[i])
		}
	}

	return &reply, nil
}

func (s *NodeServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateReply, error) {
	if utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 50 {
		return nil, status.Errorf(codes.InvalidArgument, "节点名长度限制1-50")
	} else if utf8.RuneCountInString(in.Comment) > 200 {
		return nil, status.Errorf(codes.InvalidArgument, "节点备注长度限制0-200")
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	// TODO 检查参数是否超过节点资源上限值
	// get node status

	if in.Comment != "" {
		nodeInfo.Comment = in.Comment
	}
	if in.Name != "" {
		nodeInfo.Name = in.Name
	}
	if in.RscLimit != nil {
		nodeInfo.CpuLimit = in.RscLimit.CpuLimit
		nodeInfo.MemoryLimit = in.RscLimit.MemoryLimit
		nodeInfo.DiskLimit = in.RscLimit.DiskLimit
	}

	if err := model.UpdateNode(nodeInfo); err != nil {
		log.Infof("UpdateNode %+v err=%v", nodeInfo, err)
		return nil, rpc.ErrInternal
	}

	return &pb.UpdateReply{}, nil
}
