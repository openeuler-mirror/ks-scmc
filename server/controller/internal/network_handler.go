package internal

import (
	"context"

	log "github.com/sirupsen/logrus"

	"scmc/model/db"
	"scmc/rpc"
	pb "scmc/rpc/pb/network"
	"scmc/server/args"
)

type NetworkServer struct {
	pb.UnimplementedNetworkServer
}

func (s *NetworkServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	var nodes []*db.NodeInfo
	if in.NodeId == -1 {
		var err error
		if nodes, err = db.ListNodes(); err != nil {
			log.Warnf("db.ListNodes err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
	} else {
		n, err := db.QueryNodeByID(in.NodeId)
		if err != nil {
			if err == db.ErrRecordNotFound {
				return nil, rpc.ErrNotFound
			}
			log.Warnf("db.QueryNodeByID err=%v", err)
			return nil, rpc.ErrDatabaseFail
		}
		nodes = append(nodes, n)
	}

	var reply pb.ListReply
	for _, n := range nodes {
		conn, err := getAgentConn(n.Address)
		if err != nil {
			if len(nodes) == 1 {
				return nil, rpc.ErrInternal
			}
			continue
		}

		r, err := pb.NewNetworkClient(conn).List(ctx, in)
		if err != nil {
			log.Warnf("network.List node=%+v err=%v", n, err)
			if len(nodes) == 1 {
				return nil, rpc.ErrInternal
			}
			continue
		}

		for _, v := range r.VirtualIfs {
			v.NodeId = n.ID
		}

		for _, v := range r.RealIfs {
			v.NodeId = n.ID
		}

		reply.RealIfs = append(reply.RealIfs, r.RealIfs...)
		reply.VirtualIfs = append(reply.VirtualIfs, r.VirtualIfs...)
	}

	return &reply, nil
}

func (s *NetworkServer) Connect(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).Connect(ctx, in)
}

func (s *NetworkServer) Disconnect(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).Disconnect(ctx, in)
}

func (s *NetworkServer) ListIPtables(ctx context.Context, in *pb.ListIPtablesRequest) (*pb.ListIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).ListIPtables(ctx, in)
}

func (s *NetworkServer) EnableIPtables(ctx context.Context, in *pb.EnableIPtablesRequest) (*pb.EnableIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).EnableIPtables(ctx, in)
}

func (s *NetworkServer) CreateIPtables(ctx context.Context, in *pb.CreateIPtablesRequest) (*pb.CreateIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).CreateIPtables(ctx, in)
}

func (s *NetworkServer) ModifyIPtables(ctx context.Context, in *pb.ModifyIPtablesRequest) (*pb.ModifyIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).ModifyIPtables(ctx, in)
}

func (s *NetworkServer) RemoveIPtables(ctx context.Context, in *pb.RemoveIPtablesRequest) (*pb.RemoveIPtablesReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).RemoveIPtables(ctx, in)
}

func (*NetworkServer) CreateNic(ctx context.Context, in *pb.CreateNicRequest) (*pb.CreateNicReply, error) {
	if !args.NicName.Check(in.Name) || in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).CreateNic(ctx, in)
}

func (*NetworkServer) RemoveNic(ctx context.Context, in *pb.RemoveNicRequest) (*pb.RemoveNicReply, error) {
	if in.Name == "" || in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).RemoveNic(ctx, in)
}

func (*NetworkServer) UpdateNic(ctx context.Context, in *pb.UpdateNicRequest) (*pb.UpdateNicReply, error) {
	if in.Name == "" || in.NodeId == 0 {
		return nil, rpc.ErrInvalidArgument
	}

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

	return pb.NewNetworkClient(conn).UpdateNic(ctx, in)
}
