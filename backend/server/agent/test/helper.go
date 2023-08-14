package test

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
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
	opts = append(opts, grpc.WithTimeout(time.Second*20))

	conn, err := grpc.Dial("127.0.0.1:10051", opts...)
	if err != nil {
		log.Printf("grpc.Dial: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	fn(ctx, conn)
	return nil
}
