package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

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
	s := grpc.NewServer()
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
