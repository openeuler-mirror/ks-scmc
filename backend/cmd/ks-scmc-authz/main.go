package main

import (
	"flag"
	"log"
	"os/user"
	"strconv"

	"github.com/docker/go-plugins-helpers/authorization"
)

var (
	pluginSocket = "/run/docker/plugins/authz-plugin.sock"
	rpcSocket    = "/var/lib/ks-scmc/authz.sock"
	configPath   = "/etc/ks-scmc/authz.json"
)

func main() {
	flag.Parse()

	if err := initConfig(configPath); err != nil {
		log.Printf("initConfig: %v", err)
	}

	if err := initRPCServer(rpcSocket); err != nil {
		log.Fatalf("initRPCServer: %v", err)
	}

	u, _ := user.Lookup("root")
	gid, _ := strconv.Atoi(u.Gid)
	handler := authorization.NewHandler(&AuthZPlugin{magicUser: "KS-SCMC-SERVICE"})
	if err := handler.ServeUnix(pluginSocket, gid); err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.StringVar(&pluginSocket, "plugin-sock", "/run/docker/plugins/authz-plugin.sock", "Plugin serve socket, under dir /run/docker/plugins/")
	flag.StringVar(&rpcSocket, "rpc-sock", "/var/lib/ks-scmc/authz.sock", "RPC socket to receive setting changes")
	flag.StringVar(&configPath, "config-path", "/etc/ks-scmc/authz.json", "Config file path")
}
