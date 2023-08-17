package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"ksc-mcube/common"
	"ksc-mcube/server/agent"
	"ksc-mcube/server/controller"
)

const (
	agentMode      = "agent"
	controllerMode = "controller"
)

var (
	logDir    string
	logStdout bool
	verbose   int
	mode      string
)

func main() {
	flag.Parse()

	if flag.NArg() > 0 {
		mode = flag.Arg(0)
	}

	if flag.NArg() != 1 || (mode != agentMode && mode != controllerMode) {
		fmt.Fprintf(flag.CommandLine.Output(), "%s need one arg, value must be '%s' or '%s':\n", os.Args[0], agentMode, controllerMode)
		flag.Usage()
		os.Exit(1)
	}

	// TODO check log dir, create when it not exists.
	logFile := filepath.Join(logDir, mode)
	common.InitLogger(verbose, logStdout, logFile)

	var addr string
	opts := append(common.UnaryServerInterceptor(),
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
		}))
	s := grpc.NewServer(opts...)
	if mode == agentMode {
		agent.Register(s)
		addr = fmt.Sprintf("%s:%d", common.Host, common.AgentPort)
	} else {
		controller.Register(s)
		addr = fmt.Sprintf("%s:%d", common.Host, common.ControllerPort)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Fatalf("failed to listen(%s): %v", addr, err)
	}

	logrus.Printf("%s server listening at %v", mode, lis.Addr())
	if err := s.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}
}

func init() {
	flag.StringVar(&logDir, "logdir", "/var/log/ksc-mcube/", "log output directory")
	flag.BoolVar(&logStdout, "stdout", false, "log write to stdout")
	flag.IntVar(&verbose, "verbose", int(logrus.InfoLevel), "log verbosity: trace=6 debug=5 info=4 warning=3 error=2")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [OPTIONS] agent|controller:\n", os.Args[0])
		flag.PrintDefaults()
	}
}
