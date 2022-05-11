package test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"google.golang.org/grpc"

	pb "scmc/rpc/pb/container"
)

func TestContainerList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.ListRequest{
			NodeIds: []int64{1},
			ListAll: true,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		for _, c := range reply.Containers {
			t.Logf("container info: %+v", c)
		}

	})
}

func TestContainerCreate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.CreateRequest{
			NodeId: 1,
			Name:   "cadvisor",
			Config: &pb.ContainerConfig{
				Image: "gcr.io/cadvisor/cadvisor:v0.36.0",
				Env: map[string]string{
					"ENV_TEST": "1",
				},
				Cmd: []string{
					"-storage_driver=influxdb",
					"-storage_driver_db=cadvisor",
					"-storage_driver_host=influxdb:8086",
				},
			},
			HostConfig: &pb.HostConfig{
				Privileged: true,
				RestartPolicy: &pb.RestartPolicy{
					Name: "always",
				},
				ResourceConfig: &pb.ResourceConfig{
					NanoCpus:     1e9,
					MemLimit:     1 << 30,
					MemSoftLimit: 512 * (1 << 20),
					Devices: []*pb.DeviceMapping{
						{
							PathOnHost:        "/dev/kmsg",
							PathInContainer:   "/dev/kmsg",
							CgroupPermissions: "rwm",
						},
					},
				},
				Mounts: []*pb.Mount{
					{
						Type:     "bind",
						Source:   "/",
						Target:   "/rootfs",
						ReadOnly: true,
					},
					{
						Type:     "bind",
						Source:   "/var/run",
						Target:   "/var/run",
						ReadOnly: true,
					},
					{
						Type:     "bind",
						Source:   "/sys",
						Target:   "/sys",
						ReadOnly: true,
					},
					{
						Type:     "bind",
						Source:   "/var/lib/docker/",
						Target:   "/var/lib/docker/",
						ReadOnly: true,
					},
					{
						Type:     "bind",
						Source:   "/dev/disk/",
						Target:   "/dev/disk/",
						ReadOnly: true,
					},
				},
			},
			// NetworkConfig: map[string]*pb.EndpointSetting{
			// 	"bridge": {},
			// },
		}

		reply, err := cli.Create(ctx, &request)
		if err != nil {
			t.Errorf("Create: %v", err)
		}

		t.Logf("Create reply: %+v", reply)
	})
}

func TestContainerCreate2(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.CreateRequest{
			NodeId: 1,
			Name:   "gparted",
			Config: &pb.ContainerConfig{
				Image: "jess/gparted",
			},
			EnableGraphic: true,
		}

		reply, err := cli.Create(ctx, &request)
		if err != nil {
			t.Errorf("Create: %v", err)
		}

		t.Logf("Create reply: %+v", reply)
	})
}

func TestContainerCreateWithCmd(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.CreateRequest{
			NodeId: 1,
			Name:   "busybox",
			Config: &pb.ContainerConfig{
				Image: "busybox",
				Cmd:   []string{"ip", "addr"},
			},
			HostConfig: &pb.HostConfig{
				Privileged: true,
			},
			// NetworkConfig: map[string]*pb.EndpointSetting{
			// 	"br0": {
			// 		IpamConfig: &pb.IPAMConfig{
			// 			Ipv4Address: "172.28.5.10",
			// 		},
			// 		MacAddress: "12:34:56:78:9a:bc",
			// 	},
			// },
		}

		reply, err := cli.Create(ctx, &request)
		if err != nil {
			t.Errorf("Create: %v", err)
		}

		t.Logf("Create reply: %+v", reply)
	})
}

func TestContainerStart(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.StartRequest{
			Ids: []*pb.ContainerIdList{
				{
					NodeId:       1,
					ContainerIds: []string{"cadvisor"},
				},
			},
		}

		reply, err := cli.Start(ctx, &request)
		if err != nil {
			t.Errorf("Start: %v", err)
		}

		t.Logf("Start reply: %v", reply)
	})
}

func TestContainerStop(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.StopRequest{
			Ids: []*pb.ContainerIdList{
				{
					NodeId:       1,
					ContainerIds: []string{"cadvisor"},
				},
			},
		}

		reply, err := cli.Stop(ctx, &request)
		if err != nil {
			t.Errorf("Stop: %v", err)
		}

		t.Logf("Stop reply: %v", reply)
	})
}

func TestContainerRestart(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.RestartRequest{
			Ids: []*pb.ContainerIdList{
				{
					NodeId:       1,
					ContainerIds: []string{"cadvisor"},
				},
			},
		}

		reply, err := cli.Restart(ctx, &request)
		if err != nil {
			t.Errorf("Restart: %v", err)
		}

		t.Logf("Restart reply: %v", reply)
	})
}
func TestContainerRemove(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.RemoveRequest{
			Ids: []*pb.ContainerIdList{
				{
					NodeId:       1,
					ContainerIds: []string{"cadvisor"},
				},
			},
		}

		reply, err := cli.Remove(ctx, &request)
		if err != nil {
			t.Errorf("Remove: %v", err)
		}

		t.Logf("Remove reply: %v", reply)
	})
}
func TestContainerInspect(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.InspectRequest{
			NodeId:      1,
			ContainerId: "cadvisor",
		}

		reply, err := cli.Inspect(ctx, &request)
		if err != nil {
			t.Errorf("Inspect: %v", err)
		}

		t.Logf("Inspect reply: %v", reply)
	})
}

func TestContainerStatus(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.StatusRequest{
			NodeId: 1,
		}

		reply, err := cli.Status(ctx, &request)
		if err != nil {
			t.Errorf("Status: %v", err)
		}

		t.Logf("Status reply: %v", reply)
	})
}

func TestContainerUpdate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.UpdateRequest{
			NodeId:      1,
			ContainerId: "cadvisor",
			ResourceConfig: &pb.ResourceConfig{
				NanoCpus:     2e9,
				MemLimit:     512 * (1 << 20),
				MemSoftLimit: 256 * (1 << 20),
			},
			RestartPolicy: &pb.RestartPolicy{
				Name: "unless-stopped",
			},
		}

		reply, err := cli.Update(ctx, &request)
		if err != nil {
			t.Errorf("Update: %v", err)
		}

		t.Logf("Update reply: %+v", reply)
	})
}

func TestCreateTemplate(t *testing.T) {

	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}

	type args struct {
		ctx  context.Context
		conf *pb.CreateTemplateRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.CreateTemplateReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				conf: &pb.CreateTemplateRequest{
					Data: &pb.ContainerTemplate{
						Conf: &pb.ContainerConfigs{
							Name: "test01",
						},
					},
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				conf: &pb.CreateTemplateRequest{
					Data: &pb.ContainerTemplate{
						Conf: &pb.ContainerConfigs{
							Name: "test02",
						},
					},
				},
			},
		},
		{
			name: "test03",
			args: args{
				ctx: ctx,
				conf: &pb.CreateTemplateRequest{
					Data: &pb.ContainerTemplate{
						Conf: &pb.ContainerConfigs{
							Name: "test03",
						},
					},
				},
			},
		},
		{
			name: "test04",
			args: args{
				ctx: ctx,
				conf: &pb.CreateTemplateRequest{
					Data: &pb.ContainerTemplate{
						Conf: &pb.ContainerConfigs{
							Name: "test04",
						},
					},
				},
			},
		},
		{
			name: "test05",
			args: args{
				ctx: ctx,
				conf: &pb.CreateTemplateRequest{
					Data: &pb.ContainerTemplate{
						Conf: &pb.ContainerConfigs{
							Name: "test05",
						},
					},
				},
			},
		},
	}
	cli := pb.NewContainerClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.CreateTemplate(tt.args.ctx, tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}

}

func TestUpdateTemplate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.UpdateTemplateRequest{
			Data: &pb.ContainerTemplate{
				Id: 131,
				Conf: &pb.ContainerConfigs{
					Name:  "sean04",
					Image: "sean:sean",
				},
			},
		}
		reply, err := cli.UpdateTemplate(ctx, &request)
		if err != nil {
			t.Errorf("Update: %v", err)
		}

		t.Logf("Update reply: %+v", reply)
	})
}

func TestListemplate(t *testing.T) {

	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}

	type args struct {
		ctx context.Context
		req *pb.ListTemplateRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.ListTemplateReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  0,
					NextPage: 1,
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  1,
					NextPage: 2,
				},
			},
		},
		{
			name: "test03",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  3,
					NextPage: 0,
				},
			},
		},
		{
			name: "test04",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  3,
					NextPage: 5,
				},
			},
		},
		{
			name: "test05",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  3,
					NextPage: 3,
				},
			},
		},
		{
			name: "test06",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  10,
					NextPage: 10,
				},
			},
		},
		{
			name: "test07",
			args: args{
				ctx: ctx,
				req: &pb.ListTemplateRequest{
					PerPage:  0,
					NextPage: 0,
				},
			},
		},
	}
	cli := pb.NewContainerClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cli.ListTemplate(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			fmt.Printf("got.CurPage: %v, got.PerPage: %v, got.TotalPages: %v, got.TotalRows: %v\n",
				got.CurPage, got.PerPage, got.TotalPages, got.TotalRows)
			for _, record := range got.Data {
				fmt.Printf("id: %v, config: %v\n", record.Id, record.Conf)
			}

		})
	}

}

func TestRemovetemplate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)

		request := pb.RemoveTemplateRequest{
			Ids: []int64{},
		}
		reply, err := cli.RemoveTemplate(ctx, &request)

		if err != nil {
			t.Errorf("Update: %v", err)
		}

		t.Logf("Update reply: %+v", reply)
	})
}
