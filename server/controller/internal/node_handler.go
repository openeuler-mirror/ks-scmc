package internal

import (
	"context"
	"time"
	"unicode/utf8"

	"scmc/model/db"
	"scmc/rpc"
	pb "scmc/rpc/pb/node"
	"scmc/server/args"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	// connect agent address, check is alive
	_, err = pb.NewNodeClient(conn).Status(ctx, &pb.StatusRequest{})
	if err != nil {
		log.Warnf("check agent service err=%v", err)
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Unavailable {
				return nil, status.Errorf(codes.Internal, "节点连接失败")
			}
		}
		return nil, rpc.ErrInternal
	}

	n, err := db.QueryNodeByAddr(in.Address)
	if err == db.ErrRecordNotFound {
		if e := db.CreateNode(in.Name, in.Address, in.Comment); e != nil {
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
		if e := db.UpdateNode(n); e != nil {
			log.Warnf("update node err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	}

	return &pb.CreateReply{}, nil
}

func (s *NodeServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	if err := db.RemoveNode(in.Ids); err != nil {
		return nil, rpc.ErrDatabaseFail
	}
	return &pb.RemoveReply{}, nil
}

// Since `List` returns the data of `Status` too, client never calls `Status`, therefore abandon this interface
func (s *NodeServer) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusReply, error) {
	return nil, rpc.ErrUnimplemented
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
		log.Warnf("Failed to connect to agent service, node=%+v", n)
		return nil, err
	}

	cli := pb.NewNodeClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()
	r, err := cli.Status(ctx_, &pb.StatusRequest{})
	if err != nil {
		log.Warnf("get node status ID=%v address=%v: %v", n.ID, n.Address, err)
		return nil, err
	}

	var s *pb.NodeStatus
	if len(r.StatusList) > 0 {
		s = r.StatusList[0]
	}
	return s, nil
}
