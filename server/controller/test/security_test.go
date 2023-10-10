package test

import (
	"context"
	pb "scmc/rpc/pb/security"
	"testing"

	"google.golang.org/grpc"
)

func TestFileProtectionList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSecurityClient(conn)
		request := pb.ListFileProtectionRequest{
			NodeId:      1,
			ContainerId: "fde401449bc5d2d7e12023c582e28c17157889cfd72c7f05962f35cc78af2370",
		}

		reply, err := cli.ListFileProtection(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("reply: %+v", reply)
	})
}

func TestFileProtectionUpdate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSecurityClient(conn)
		request := pb.UpdateFileProtectionRequest{
			NodeId: 1,
			//ContainerId: "localhost",
			ContainerId: "372c933f30c217990b854a0d1db9addfab9e524764fe6207dfc433d41173cf01",
			IsOn:        true,
			//ToRemove:    []string{"/tmp/file1", "/tmp/file12", "/tmp/file123", "/tmp/file1234", "/tmp/file12345"},
			ToAppend: []string{"/tmp/file1", "/tmp/file12", "/tmp/file123", "/tmp/file1234", "/tmp/file12345"},
			//ToAppend: []string{"/root/tmp/test/file1", "/root/tmp/test/file12", "/root/tmp/test/file123", "/root/tmp/test/file1234", "/root/tmp/test/file12345"},
			//ToRemove: []string{"/root/tmp/test/file1", "/root/tmp/test/file12", "/root/tmp/test/file123", "/root/tmp/test/file1234", "/root/tmp/test/file12345"},
		}

		reply, err := cli.UpdateFileProtection(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("UpdateFileProtection reply: %+v", reply)
	})
}

func TestListProcProtection(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSecurityClient(conn)
		request := pb.ListProcProtectionRequest{
			NodeId:         12,
			ContainerId:    "a23sdf3asf",
			ProtectionType: 1,
		}

		reply, err := cli.ListProcProtection(ctx, &request)
		if err != nil {
			t.Errorf("Remove: %v", err)
		}

		t.Logf("Remove reply: %v", reply)
	})
}

func TestUpdateProcProtection(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSecurityClient(conn)
		request := pb.UpdateProcProtectionRequest{
			NodeId:         1,
			ContainerId:    "bbbbbbbbbb",
			ProtectionType: 2,
			IsOn:           true,
			ToAppend: []string{
				"/usr/bin/curl",
				"/usr/bin/wget",
				"/usr/bin/ls",
			},
			ToRemove: []string{
				"/usr/bin/ls",
				"/usr/bin/wget",
			},
		}

		reply, err := cli.UpdateProcProtection(ctx, &request)
		if err != nil {
			t.Errorf("Remove: %v", err)
		}

		t.Logf("Remove reply: %v", reply)
	})
}
