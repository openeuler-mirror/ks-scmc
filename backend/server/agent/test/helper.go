package test

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"

	"ksc-mcube/common"
)

func testRunner(fn func(context.Context, *grpc.ClientConn)) error {
	var opts []grpc.DialOption
	/*
		if *tls {
			if *caFile == "" {
				*caFile = data.Path("x509/ca_cert.pem")
			}
			creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
			if err != nil {
				log.Fatalf("Failed to create TLS credentials %v", err)
			}
			opts = append(opts, grpc.WithTransportCredentials(creds))
		} else {
			opts = append(opts, grpc.WithInsecure())
		}
	*/
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithTimeout(time.Second*5))

	addr := fmt.Sprintf("127.0.0.1:%d", common.AgentPort)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Printf("grpc.Dial: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	fn(ctx, conn)
	return nil
}
