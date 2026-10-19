package test

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.DialContext(ctx, "127.0.0.1:10051", opts...)
	if err != nil {
		log.Printf("grpc.Dial: %v", err)
		return err
	}

	fn(ctx, conn)
	return nil
}
