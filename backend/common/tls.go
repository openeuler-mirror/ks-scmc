package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"google.golang.org/grpc/credentials"
)

func LoadTLSCredentials() (credentials.TransportCredentials, error) {
	if !Config.TLS.Enable {
		return nil, nil
	}

	if Config.TLS.CA == "" || Config.TLS.ServerCert == "" || Config.TLS.ServerKey == "" {
		return nil, nil
	}

	caCert, err := ioutil.ReadFile(Config.TLS.CA)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	serverCert, err := tls.LoadX509KeyPair(Config.TLS.ServerCert, Config.TLS.ServerKey)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}), nil
}
