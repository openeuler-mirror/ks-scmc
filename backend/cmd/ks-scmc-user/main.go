package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "scmc/rpc/pb/user"
)

func makeCall(f func(context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "0:xxxxx")
	defer cancel()

	f(ctx)
}

func parseRequest(args string, req interface{}) {
	if err := json.Unmarshal([]byte(args), req); err != nil {
		log.Fatalf("parse request to %T: %v", req, err)
	}
}

func printReply(reply interface{}) {
	if s, err := json.MarshalIndent(reply, "", "  "); err != nil {
		log.Printf("reply: %+v", reply)
	} else {
		log.Printf("reply: %s", s)
	}
}

func listUser(cli pb.UserClient, args string) {
	var req pb.ListUserRequest

	makeCall(func(ctx context.Context) {
		rep, err := cli.ListUser(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func createUser(cli pb.UserClient, args string) {
	var req pb.CreateUserRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.CreateUser(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func updateUser(cli pb.UserClient, args string) {
	var req pb.UpdateUserRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.UpdateUser(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func removeUser(cli pb.UserClient, args string) {
	var req pb.RemoveUserRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.RemoveUser(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func listRole(cli pb.UserClient, args string) {
	var req pb.ListRoleRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.ListRole(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func createRole(cli pb.UserClient, args string) {
	var req pb.CreateRoleRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.CreateRole(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func updateRole(cli pb.UserClient, args string) {
	var req pb.UpdateRoleRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.UpdateRole(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func removeRole(cli pb.UserClient, args string) {
	var req pb.RemoveRoleRequest
	parseRequest(args, &req)

	makeCall(func(ctx context.Context) {
		rep, err := cli.RemoveRole(ctx, &req)
		if err != nil {
			log.Fatal(err)
		}
		printReply(rep)
	})
}

func handleInput(serverAddr, cmd, args string) {
	var f func(pb.UserClient, string)
	switch strings.ToLower(cmd) {
	case "list_user":
		f = listUser
	case "create_user":
		f = createUser
	case "update_user":
		f = updateUser
	case "remove_user":
		f = removeUser
	case "list_role":
		f = listRole
	case "create_role":
		f = createRole
	case "update_role":
		f = updateRole
	case "remove_role":
		f = removeRole
	default:
		log.Fatalf("unknown cmd[%s]", cmd)
	}

	conn, err := grpc.Dial(
		serverAddr,
		[]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}...)
	if err != nil {
		log.Fatalf("dial server[%s]: %v", serverAddr, err)
	}

	f(pb.NewUserClient(conn), args)
}

func main() {
	serverAddr := flag.String("s", "127.0.0.1:10050", "server address")
	cmd := flag.String("c", "", "user command(list|create|update|remove + _ + user|role)")
	flag.Parse()

	argsLen := len(flag.Args())
	if argsLen == 0 {
		log.Fatal("need arguments")
	}
	args := flag.Args()[0]
	log.Print("args: ", args)

	handleInput(*serverAddr, *cmd, args)
}
