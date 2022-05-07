package test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

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

func testRunner(fn func(context.Context, *grpc.ClientConn)) error {
	var opts []grpc.DialOption

	if false {
		creds, err := loadTLSCredentials()
		if err != nil {
			log.Fatalf("load tls error=%v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial("127.0.0.1:10050", opts...)
	if err != nil {
		log.Printf("grpc.Dial: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "1000:fjkdafjkldkl")

	defer cancel()
	fn(ctx, conn)
	return nil
}

func initTestRunner() (context.Context, *grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if false {
		creds, err := loadTLSCredentials()
		if err != nil {
			log.Fatalf("load tls error=%v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial("127.0.0.1:10050", opts...)
	if err != nil {
		log.Printf("grpc.Dial: %v", err)
		return nil, nil, err
	}

	ctx := context.TODO()

	return ctx, conn, nil
}
