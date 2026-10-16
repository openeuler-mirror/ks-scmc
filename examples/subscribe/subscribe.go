package main

/*
struct WarnLog{
	long id;
	long eventType;
	long eventModule;
	long nodeId;
	char* nodeInfo;
	char* containerId;
	char* containerName;
	char* detail;
	long haveRead;
	long createdAt;
	long updatedAt;
};
*/
import "C"

import (
	pblogging "scmc/rpc/pb/logging"
	pbuser "scmc/rpc/pb/user"
	"strconv"

	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var stream pblogging.Logging_SubscribeWarnClient

//export Subscribe
func Subscribe(addr, user, password, tls *C.char) {
	var opts []grpc.DialOption

	if C.GoString(tls) == "ture" {
		creds, err := loadTLSCredentials()
		if err != nil {
			log.Fatalf("load tls error=%v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(C.GoString(addr), opts...)
	if err != nil {
		log.Fatalf("grpc.Dial: %v", err)

	}

	cli := pbuser.NewUserClient(conn)
	request := pbuser.LoginRequest{
		Username: C.GoString(user),
		Password: C.GoString(password),
	}
	ctx := context.Background()
	reply, err := cli.Login(ctx, &request)
	if err != nil {
		log.Errorf("Login: %v", err)
	}

	authorization := fmt.Sprintf("%s:%s", strconv.FormatInt(reply.UserId, 10), reply.GetAuthKey())

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authorization)

	logCli := pblogging.NewLoggingClient(conn)
	logRequest := pblogging.SubscribeWarnRequest{}

	stream, err = logCli.SubscribeWarn(ctx, &logRequest)
	if err != nil {
		log.Fatalf("SubscribeWarn: %v", err)
	}

}

//export GetMessage
func GetMessage() C.struct_WarnLog {

	if stream.Context().Err() == context.Canceled || stream.Context().Err() == context.DeadlineExceeded {
		log.Errorf("context error: %v", stream.Context().Err())
	}

	reply, err := stream.Recv()
	if err == io.EOF {
		log.Info("no more data")
		return C.struct_WarnLog{}
	}

	if err != nil {
		log.Errorf("cannot receive chunk data: %v", err)
		return C.struct_WarnLog{}
	}

	log := reply.Logs[0]
	haveRead := 0
	if log.HaveRead {
		haveRead = 1
	}
	return C.struct_WarnLog{
		C.long(log.Id),
		C.long(log.EventType),
		C.long(log.EventModule),
		C.long(log.NodeId),
		C.CString(log.NodeInfo),
		C.CString(log.ContainerId),
		C.CString(log.ContainerName),
		C.CString(log.Detail),
		C.long(haveRead),
		C.long(log.CreatedAt),
		C.long(log.UpdatedAt),
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	caPEM, err := ioutil.ReadFile("/etc/ks-scmc/x509/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	cert, err := tls.LoadX509KeyPair("/etc/ks-scmc/x509/client-cert.pem", "/etc/ks-scmc/x509/client-key.pem")
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
	}), nil
}

func main() {}
